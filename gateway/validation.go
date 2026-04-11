package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type WebSocketHeartbeat struct {
	mu           sync.RWMutex
	connections  map[string]*wsConnection
	pingInterval time.Duration
	timeout      time.Duration
	redis        *redis.Client
	ctx          context.Context
}

type wsConnection struct {
	nodeID     string
	lastPing   time.Time
	lastPong   time.Time
	isAlive    bool
	remoteAddr string
}

func NewWebSocketHeartbeat(pingInterval, timeout time.Duration, redisClient *redis.Client) *WebSocketHeartbeat {
	wsh := &WebSocketHeartbeat{
		connections:  make(map[string]*wsConnection),
		pingInterval: pingInterval,
		timeout:      timeout,
		redis:        redisClient,
		ctx:          context.Background(),
	}

	go wsh.cleanupLoop()

	return wsh
}

func (wsh *WebSocketHeartbeat) RegisterConnection(connID, nodeID, remoteAddr string) {
	wsh.mu.Lock()
	defer wsh.mu.Unlock()

	wsh.connections[connID] = &wsConnection{
		nodeID:     nodeID,
		lastPing:   time.Now(),
		lastPong:   time.Now(),
		isAlive:    true,
		remoteAddr: remoteAddr,
	}
}

func (wsh *WebSocketHeartbeat) RecordPong(connID string) {
	wsh.mu.Lock()
	defer wsh.mu.Unlock()

	if conn, exists := wsh.connections[connID]; exists {
		conn.lastPong = time.Now()
		conn.isAlive = true
	}
}

func (wsh *WebSocketHeartbeat) UnregisterConnection(connID string) {
	wsh.mu.Lock()
	defer wsh.mu.Unlock()

	delete(wsh.connections, connID)
}

func (wsh *WebSocketHeartbeat) IsAlive(connID string) bool {
	wsh.mu.RLock()
	defer wsh.mu.RUnlock()

	if conn, exists := wsh.connections[connID]; exists {
		return conn.isAlive
	}
	return false
}

func (wsh *WebSocketHeartbeat) GetStats() map[string]interface{} {
	wsh.mu.RLock()
	defer wsh.mu.RUnlock()

	total := len(wsh.connections)
	alive := 0
	for _, conn := range wsh.connections {
		if conn.isAlive {
			alive++
		}
	}

	return map[string]interface{}{
		"total_connections": total,
		"alive_connections": alive,
	}
}

func (wsh *WebSocketHeartbeat) cleanupLoop() {
	ticker := time.NewTicker(wsh.pingInterval)
	defer ticker.Stop()

	for range ticker.C {
		wsh.mu.Lock()
		now := time.Now()
		for id, conn := range wsh.connections {
			if now.Sub(conn.lastPong) > wsh.timeout {
				conn.isAlive = false
				_ = id
			}
		}
		wsh.mu.Unlock()
	}
}

func (wsh *WebSocketHeartbeat) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.GetHeader("X-Connection-ID")
		if connID == "" {
			connID = fmt.Sprintf("conn-%d", time.Now().UnixNano())
		}

		c.Set("connection_id", connID)
		c.Next()

		wsh.UnregisterConnection(connID)
	}
}

type RequestValidator struct {
	mu            sync.RWMutex
	rules         map[string]*validationRule
	maxBodySize   int64
	allowedCTypes []string
}

type validationRule struct {
	Path         string   `json:"path"`
	Method       string   `json:"method"`
	RequiredCols []string `json:"required_fields"`
	MaxBodySize  int64    `json:"max_body_size"`
	AllowJSON    bool     `json:"allow_json"`
	AllowForm    bool     `json:"allow_form"`
}

func NewRequestValidator(maxBodySize int64) *RequestValidator {
	rv := &RequestValidator{
		rules:         make(map[string]*validationRule),
		maxBodySize:   maxBodySize,
		allowedCTypes: []string{"application/json", "application/x-www-form-urlencoded"},
	}

	return rv
}

func (rv *RequestValidator) AddRule(path, method string, requiredFields []string, maxBodySize int64) {
	rv.mu.Lock()
	defer rv.mu.Unlock()

	key := fmt.Sprintf("%s:%s", method, path)
	rv.rules[key] = &validationRule{
		Path:         path,
		Method:       method,
		RequiredCols: requiredFields,
		MaxBodySize:  maxBodySize,
		AllowJSON:    true,
		AllowForm:    true,
	}
}

func (rv *RequestValidator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method
		key := fmt.Sprintf("%s:%s", method, path)

		rv.mu.RLock()
		rule, exists := rv.rules[key]
		rv.mu.RUnlock()

		if exists && c.Request.ContentLength > rule.MaxBodySize && rule.MaxBodySize > 0 {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
			})
			return
		}

		c.Next()
	}
}

func (rv *RequestValidator) ValidateBody(c *gin.Context) error {
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		return nil
	}

	allowed := false
	for _, ct := range rv.allowedCTypes {
		if len(contentType) >= len(ct) && contentType[:len(ct)] == ct {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("unsupported content type: %s", contentType)
	}

	return nil
}

func (rv *RequestValidator) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/validators", rv.createRuleHandler)
	r.GET("/api/admin/validators", rv.listRulesHandler)
	r.DELETE("/api/admin/validators", rv.deleteRuleHandler)
}

func (rv *RequestValidator) createRuleHandler(c *gin.Context) {
	var rule validationRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rv.AddRule(rule.Path, rule.Method, rule.RequiredCols, rule.MaxBodySize)

	c.JSON(http.StatusCreated, gin.H{"status": "success"})
}

func (rv *RequestValidator) listRulesHandler(c *gin.Context) {
	rv.mu.RLock()
	defer rv.mu.RUnlock()

	rules := make([]*validationRule, 0, len(rv.rules))
	for _, rule := range rv.rules {
		rules = append(rules, rule)
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func (rv *RequestValidator) deleteRuleHandler(c *gin.Context) {
	var req struct {
		Path   string `json:"path"`
		Method string `json:"method"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := fmt.Sprintf("%s:%s", req.Method, req.Path)
	rv.mu.Lock()
	delete(rv.rules, key)
	rv.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

type NodeSelectionStrategy string

const (
	StrategyRandom        NodeSelectionStrategy = "random"
	StrategyLeastLoad     NodeSelectionStrategy = "least_load"
	StrategyLowestLatency NodeSelectionStrategy = "lowest_latency"
	StrategyHighestRep    NodeSelectionStrategy = "highest_reputation"
	StrategyRoundRobin    NodeSelectionStrategy = "round_robin"
)

type NodeSelector struct {
	mu             sync.RWMutex
	strategy       NodeSelectionStrategy
	latencyTracker interface {
		GetAvgLatency(nodeID string) int64
	}
	rrCounter int
}

func NewNodeSelector(strategy NodeSelectionStrategy, latencyTracker interface{ GetAvgLatency(string) int64 }) *NodeSelector {
	return &NodeSelector{
		strategy:       strategy,
		latencyTracker: latencyTracker,
	}
}

func (ns *NodeSelector) SetStrategy(strategy NodeSelectionStrategy) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.strategy = strategy
}

func (ns *NodeSelector) Select(nodes []string) string {
	if len(nodes) == 0 {
		return ""
	}

	ns.mu.RLock()
	strategy := ns.strategy
	ns.mu.RUnlock()

	switch strategy {
	case StrategyRandom:
		return ns.selectRandom(nodes)
	case StrategyLeastLoad:
		return ns.selectLeastLoad(nodes)
	case StrategyLowestLatency:
		return ns.selectLowestLatency(nodes)
	case StrategyHighestRep:
		return nodes[0]
	case StrategyRoundRobin:
		return ns.selectRoundRobin(nodes)
	default:
		return nodes[0]
	}
}

func (ns *NodeSelector) selectRandom(nodes []string) string {
	now := time.Now()
	idx := int(now.UnixNano()) % len(nodes)
	return nodes[idx]
}

func (ns *NodeSelector) selectLeastLoad(nodes []string) string {
	return nodes[0]
}

func (ns *NodeSelector) selectLowestLatency(nodes []string) string {
	if ns.latencyTracker == nil {
		return nodes[0]
	}

	bestNode := nodes[0]
	bestLatency := ns.latencyTracker.GetAvgLatency(bestNode)

	for i := 1; i < len(nodes); i++ {
		latency := ns.latencyTracker.GetAvgLatency(nodes[i])
		if latency < bestLatency || bestLatency == 0 {
			bestLatency = latency
			bestNode = nodes[i]
		}
	}

	return bestNode
}

func (ns *NodeSelector) selectRoundRobin(nodes []string) string {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	idx := ns.rrCounter % len(nodes)
	ns.rrCounter++
	return nodes[idx]
}

func (ns *NodeSelector) GetStrategy() NodeSelectionStrategy {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.strategy
}
