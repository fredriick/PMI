package gateway

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type Client struct {
	hub    *Hub
	conn   *gin.Context
	send   chan []byte
	prefix string
}

type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type NodeUpdate struct {
	NodeID    string `json:"node_id"`
	Status    string `json:"status"`
	Country   string `json:"country"`
	Load      int64  `json:"load"`
	Timestamp int64  `json:"timestamp"`
}

type MetricsUpdate struct {
	Requests   uint64  `json:"requests"`
	Success    uint64  `json:"success"`
	Failed     uint64  `json:"failed"`
	LatencyAvg float64 `json:"latency_avg"`
	Timestamp  int64   `json:"timestamp"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WS client connected: %s", client.conn.ClientIP())

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("WS client disconnected: %s", client.conn.ClientIP())

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(c *Client) {
	h.register <- c
}

func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) BroadcastNodeUpdate(update NodeUpdate) {
	msg := WSMessage{
		Type:    "node_update",
		Payload: mustMarshal(update),
	}
	h.Broadcast(mustMarshal(msg))
}

func (h *Hub) BroadcastMetricsUpdate(update MetricsUpdate) {
	msg := WSMessage{
		Type:    "metrics_update",
		Payload: mustMarshal(update),
	}
	h.Broadcast(mustMarshal(msg))
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

func (g *Gateway) setupWebSocket() {
	g.wsHub = NewHub()
	go g.wsHub.Run()

	g.router.GET("/ws", g.wsHandler)
}

func (g *Gateway) wsHandler(c *gin.Context) {
	client := &Client{
		hub:    g.wsHub,
		conn:   c,
		send:   make(chan []byte, 256),
		prefix: "admin",
	}
	g.wsHub.Register(client)
	defer g.wsHub.Unregister(client)

	go func() {
		for msg := range client.send {
			c.Writer.Write([]byte(msg))
		}
	}()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Writer.Write([]byte(": heartbeat\n\n"))
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

func (g *Gateway) StartNodeBroadcast() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if g.wsHub.ClientCount() == 0 {
			continue
		}

		nodes, err := g.matchmaker.GetAllNodes()
		if err != nil {
			continue
		}

		for _, nodeID := range nodes {
			node, err := g.matchmaker.GetNodeStatus(nodeID)
			if err != nil {
				continue
			}
			load, _ := g.matchmaker.GetRedis().GetNodeLoad(nodeID)

			update := NodeUpdate{
				NodeID:    nodeID,
				Status:    "healthy",
				Country:   node.Country,
				Load:      load,
				Timestamp: time.Now().Unix(),
			}
			g.wsHub.BroadcastNodeUpdate(update)
		}
	}
}
