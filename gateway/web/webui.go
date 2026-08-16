package web

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"proxymesh/matchmaker"
)

type WebUI struct {
	mm *matchmaker.Matchmaker
}

func NewWebUI(mm *matchmaker.Matchmaker) *WebUI {
	return &WebUI{mm: mm}
}

func (w *WebUI) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/web")
	g.GET("/dashboard", w.DashboardHandler)
	g.GET("/api/nodes", w.NodesAPIHandler)
	g.GET("/api/cooldowns", w.CooldownsAPIHandler)
	g.GET("/api/health", w.HealthAPIHandler)
	g.GET("/api/health/:nodeID", w.NodeHealthAPIHandler)
	g.GET("/api/health/:nodeID/history", w.NodeHealthHistoryAPIHandler)
	g.POST("/api/nodes/:id/reset", w.ResetCircuitBreakerHandler)
	g.POST("/api/nodes/:id/eject", w.EjectNodeHandler)
}

func (w *WebUI) DashboardHandler(c *gin.Context) {
	c.File("web/dashboard/index.html")
}

func (w *WebUI) NodesAPIHandler(c *gin.Context) {
	nodes, err := w.mm.GetAllNodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get nodes"})
		return
	}
	var list []map[string]interface{}
	for _, id := range nodes {
		n, err := w.mm.GetNodeStatus(id)
		if err != nil {
			continue
		}
		load, _ := w.mm.GetRedis().GetNodeLoad(id)
		list = append(list, map[string]interface{}{
			"id":         n.ID,
			"country":    n.Country,
			"city":       n.City,
			"isp":        n.ISP,
			"os":         n.OS,
			"reputation": n.Reputation,
			"last_seen":  n.LastSeen.Format(time.RFC3339),
			"load":       load,
		})
	}
	c.JSON(http.StatusOK, gin.H{"nodes": list, "count": len(list)})
}

func (w *WebUI) CooldownsAPIHandler(c *gin.Context) {
	cooldowns, err := w.mm.GetCooldownEntries()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get cooldowns"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cooldowns": cooldowns})
}

func (w *WebUI) HealthAPIHandler(c *gin.Context) {
	peers := w.mm.GetTopHealthPeers(20)
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(peers),
		"peers":  peers,
	})
}

func (w *WebUI) NodeHealthAPIHandler(c *gin.Context) {
	nodeID := c.Param("nodeID")
	score := w.mm.GetHealthScore(nodeID)
	if score == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No health score available"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"node_id": nodeID,
		"score":   score,
	})
}

func (w *WebUI) NodeHealthHistoryAPIHandler(c *gin.Context) {
	nodeID := c.Param("nodeID")
	history := w.mm.GetHealthScoreHistory(nodeID)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"node_id": nodeID,
		"history": history,
		"count":   len(history),
	})
}

func (w *WebUI) ResetCircuitBreakerHandler(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id required"})
		return
	}
	w.mm.ResetCircuitBreaker(nodeID)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Circuit breaker reset"})
}

func (w *WebUI) EjectNodeHandler(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id required"})
		return
	}
	if err := w.mm.DeregisterNode(nodeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
