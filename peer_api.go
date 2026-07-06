package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"proxymesh/matchmaker"
	"proxymesh/payout"
)

type PeerSessionStore struct {
	tokens map[string]string // token -> nodeID
	mu     sync.RWMutex
}

func NewPeerSessionStore() *PeerSessionStore {
	return &PeerSessionStore{
		tokens: make(map[string]string),
	}
}

func (s *PeerSessionStore) Create(nodeID string) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)
	s.mu.Lock()
	s.tokens[token] = nodeID
	s.mu.Unlock()
	return token
}

func (s *PeerSessionStore) Validate(token string) (string, bool) {
	s.mu.RLock()
	nodeID, ok := s.tokens[token]
	s.mu.RUnlock()
	return nodeID, ok
}

func (s *PeerSessionStore) Revoke(token string) {
	s.mu.Lock()
	delete(s.tokens, token)
	s.mu.Unlock()
}


func peerHealthHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.GetString("nodeID")
		score := mm.GetHealthScore(nodeID)
		if score == nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"node_id": nodeID,
				"score":   nil,
				"message": "No health score available yet",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"node_id": nodeID,
			"score":   score,
		})
	}
}

func setupPeerRoutes(r *gin.Engine, mm *matchmaker.Matchmaker, ps *payout.PayoutService) {
	sessions := NewPeerSessionStore()

	peer := r.Group("/api/peer")
	peer.Use(peerAuthMiddleware(sessions))
	{
		peer.GET("/status", peerStatusHandler(mm))
		peer.GET("/bandwidth", peerBandwidthHandler(mm))
		peer.GET("/earnings", peerEarningsHandler(mm, ps))
		peer.GET("/health", peerHealthHandler(mm))
		peer.POST("/consent", peerConsentHandler(mm))
		peer.POST("/disconnect", peerDisconnectHandler(mm, sessions))
	}

	// Auth endpoint outside the middleware group
	r.POST("/api/peer/auth", peerAuthHandler(mm, sessions))

	// Streaming telemetry (SSE) for the peer dashboard. Token is supplied via
	// query param because EventSource cannot set request headers.
	r.GET("/api/peer/stream", peerStreamHandler(mm, ps, sessions))

	// Serve peer PWA
	r.GET("/peer", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/peer/")
	})
	r.Static("/peer", "web/peer")
}

func peerStreamHandler(mm *matchmaker.Matchmaker, ps *payout.PayoutService, sessions *PeerSessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			token = c.GetHeader("X-Peer-Token")
		}
		nodeID, ok := sessions.Validate(token)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
			return
		}

		send := func() {
			payload := buildPeerTelemetry(mm, ps, nodeID)
			data, err := json.Marshal(payload)
			if err != nil {
				return
			}
			fmt.Fprintf(c.Writer, "event: telemetry\ndata: %s\n\n", data)
			flusher.Flush()
		}

		send()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.Request.Context().Done():
				return
			case <-ticker.C:
				send()
			}
		}
	}
}

func buildPeerTelemetry(mm *matchmaker.Matchmaker, ps *payout.PayoutService, nodeID string) map[string]interface{} {
	telemetry := map[string]interface{}{"node_id": nodeID}

	if node, err := mm.GetNodeStatus(nodeID); err == nil {
		load, _ := mm.GetRedis().GetNodeLoad(nodeID)
		telemetry["node"] = node
		telemetry["load"] = load
	}

	if bw, err := mm.GetRedis().GetBandwidth(nodeID, time.Now()); err == nil {
		history, _ := mm.GetRedis().GetBandwidthHistory(nodeID)
		telemetry["current"] = bw
		telemetry["history"] = history
	}

	if ps != nil {
		if payoutData, err := ps.CalculatePayout(nodeID, time.Now()); err == nil {
			rates := ps.GetRates()
			tiers := ps.GetTiers()
			history, _ := ps.GetPayoutHistory(nodeID, 10)
			telemetry["payout"] = payoutData
			telemetry["rates"] = rates
			telemetry["tiers"] = tiers
			telemetry["payout_history"] = history
		}
	}

	if score := mm.GetHealthScore(nodeID); score != nil {
		telemetry["score"] = score
	}

	return telemetry
}

func peerAuthMiddleware(sessions *PeerSessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Peer-Token")
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing peer token",
			})
			return
		}
		nodeID, ok := sessions.Validate(token)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			return
		}
		c.Set("nodeID", nodeID)
		c.Next()
	}
}

func peerAuthHandler(mm *matchmaker.Matchmaker, sessions *PeerSessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			NodeID string `json:"node_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := mm.GetNodeStatus(req.NodeID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Node not found. Register your node first."})
			return
		}

		token := sessions.Create(req.NodeID)
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"token":   token,
			"node_id": req.NodeID,
		})
	}
}

func peerStatusHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.GetString("nodeID")

		node, err := mm.GetNodeStatus(nodeID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
			return
		}

		load, _ := mm.GetRedis().GetNodeLoad(nodeID)

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"node":   node,
			"load":   load,
		})
	}
}

func peerBandwidthHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.GetString("nodeID")

		bandwidth, err := mm.GetRedis().GetBandwidth(nodeID, time.Now())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		history, _ := mm.GetRedis().GetBandwidthHistory(nodeID)

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"current": bandwidth,
			"history": history,
		})
	}
}

func peerEarningsHandler(mm *matchmaker.Matchmaker, ps *payout.PayoutService) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.GetString("nodeID")

		payoutData, err := ps.CalculatePayout(nodeID, time.Now())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rates := ps.GetRates()
		tiers := ps.GetTiers()

		history, _ := ps.GetPayoutHistory(nodeID, 10)

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"payout":  payoutData,
			"rates":   rates,
			"tiers":   tiers,
			"history": history,
		})
	}
}

func peerConsentHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.GetString("nodeID")

		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if !req.Enabled {
			if err := mm.DeregisterNode(nodeID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"node_id": nodeID,
			"consent": req.Enabled,
		})
	}
}

func peerDisconnectHandler(mm *matchmaker.Matchmaker, sessions *PeerSessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.GetString("nodeID")
		token := c.GetHeader("X-Peer-Token")

		mm.DeregisterNode(nodeID)
		sessions.Revoke(token)

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Disconnected",
		})
	}
}
