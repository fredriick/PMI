package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type PeerBroadcaster struct {
	mu         sync.RWMutex
	subs       map[string]map[string]chan []byte
	redis      *redis.Client
	ctx        context.Context
	history    []BroadcastMessage
	maxHistory int
}

type BroadcastMessage struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Payload string    `json:"payload"`
	Target  string    `json:"target,omitempty"`
	SentAt  time.Time `json:"sent_at"`
}

func NewPeerBroadcaster(redisClient *redis.Client, maxHistory int) *PeerBroadcaster {
	if maxHistory <= 0 {
		maxHistory = 100
	}

	return &PeerBroadcaster{
		subs:       make(map[string]map[string]chan []byte),
		redis:      redisClient,
		ctx:        context.Background(),
		history:    make([]BroadcastMessage, 0, maxHistory),
		maxHistory: maxHistory,
	}
}

func (pb *PeerBroadcaster) Subscribe(nodeID string) chan []byte {
	ch := make(chan []byte, 10)

	pb.mu.Lock()
	if pb.subs[nodeID] == nil {
		pb.subs[nodeID] = make(map[string]chan []byte)
	}
	pb.subs[nodeID]["default"] = ch
	pb.mu.Unlock()

	return ch
}

func (pb *PeerBroadcaster) Unsubscribe(nodeID string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if subs, exists := pb.subs[nodeID]; exists {
		close(subs["default"])
		delete(subs, "default")
	}
}

func (pb *PeerBroadcaster) Broadcast(msgType, payload string) {
	msg := BroadcastMessage{
		ID:      fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Type:    msgType,
		Payload: payload,
		SentAt:  time.Now(),
	}

	pb.mu.Lock()
	pb.history = append(pb.history, msg)
	if len(pb.history) > pb.maxHistory {
		pb.history = pb.history[1:]
	}
	pb.mu.Unlock()

	if pb.redis != nil {
		data, _ := json.Marshal(msg)
		pb.redis.Publish(pb.ctx, "peer:broadcast", string(data))
	}

	pb.mu.RLock()
	defer pb.mu.RUnlock()

	for _, subs := range pb.subs {
		if ch, exists := subs["default"]; exists {
			select {
			case ch <- []byte(payload):
			default:
			}
		}
	}
}

func (pb *PeerBroadcaster) SendToNode(nodeID, msgType, payload string) error {
	msg := BroadcastMessage{
		ID:      fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Type:    msgType,
		Payload: payload,
		Target:  nodeID,
		SentAt:  time.Now(),
	}

	pb.mu.Lock()
	pb.history = append(pb.history, msg)
	if len(pb.history) > pb.maxHistory {
		pb.history = pb.history[1:]
	}
	pb.mu.Unlock()

	if pb.redis != nil {
		data, _ := json.Marshal(msg)
		pb.redis.Publish(pb.ctx, fmt.Sprintf("peer:%s:broadcast", nodeID), string(data))
	}

	pb.mu.RLock()
	defer pb.mu.RUnlock()

	if subs, exists := pb.subs[nodeID]; exists {
		if ch, exists := subs["default"]; exists {
			select {
			case ch <- []byte(payload):
				return nil
			default:
			}
		}
	}

	return fmt.Errorf("node %s not connected", nodeID)
}

func (pb *PeerBroadcaster) GetHistory(limit int) []BroadcastMessage {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	if limit <= 0 || limit > len(pb.history) {
		limit = len(pb.history)
	}

	return pb.history[:limit]
}

func (pb *PeerBroadcaster) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/broadcast", pb.broadcastHandler)
	r.POST("/api/admin/broadcast/:nodeId", pb.sendToNodeHandler)
	r.GET("/api/admin/broadcast/history", pb.historyHandler)
}

func (pb *PeerBroadcaster) broadcastHandler(c *gin.Context) {
	var req struct {
		Type    string `json:"type" binding:"required"`
		Payload string `json:"payload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	pb.Broadcast(req.Type, req.Payload)
	c.JSON(200, gin.H{"status": "broadcast sent"})
}

func (pb *PeerBroadcaster) sendToNodeHandler(c *gin.Context) {
	nodeID := c.Param("nodeId")

	var req struct {
		Type    string `json:"type" binding:"required"`
		Payload string `json:"payload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := pb.SendToNode(nodeID, req.Type, req.Payload); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "message sent"})
}

func (pb *PeerBroadcaster) historyHandler(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	history := pb.GetHistory(limit)
	c.JSON(200, gin.H{"messages": history})
}

type ConfigNotifier struct {
	mu       sync.RWMutex
	channels map[string]chan string
}

func NewConfigNotifier() *ConfigNotifier {
	return &ConfigNotifier{
		channels: make(map[string]chan string),
	}
}

func (cn *ConfigNotifier) Subscribe(clientID string) chan string {
	ch := make(chan string, 5)

	cn.mu.Lock()
	cn.channels[clientID] = ch
	cn.mu.Unlock()

	return ch
}

func (cn *ConfigNotifier) Unsubscribe(clientID string) {
	cn.mu.Lock()
	defer cn.mu.Unlock()

	if ch, exists := cn.channels[clientID]; exists {
		close(ch)
		delete(cn.channels, clientID)
	}
}

func (cn *ConfigNotifier) Notify(message string) {
	cn.mu.RLock()
	defer cn.mu.RUnlock()

	for _, ch := range cn.channels {
		select {
		case ch <- message:
		default:
		}
	}
}

func (cn *ConfigNotifier) NotifyConfigReload() {
	cn.Notify("config_reloaded")
}

type ResponseCompressor struct {
	enabled bool
	level   int
}

func NewResponseCompressor(enabled bool, level int) *ResponseCompressor {
	if level <= 0 {
		level = 5
	}
	return &ResponseCompressor{
		enabled: enabled,
		level:   level,
	}
}

func (rc *ResponseCompressor) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rc.enabled {
			c.Next()
			return
		}

		c.Header("X-Compression", "enabled")
		c.Next()
	}
}
