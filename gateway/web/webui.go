package web

import (
	"fmt"
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
	nodes, _ := w.mm.GetAllNodes()
	cooldowns, _ := w.mm.GetCooldownEntries()

	var rows string
	for _, nodeID := range nodes {
		node, err := w.mm.GetNodeStatus(nodeID)
		if err != nil {
			continue
		}
		load, _ := w.mm.GetRedis().GetNodeLoad(nodeID)
		rows += fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%.1f</td><td>%d</td><td>%s</td>
			<td><a href="/web/api/nodes/%s/reset">Reset</a> | <a href="/web/api/nodes/%s/eject">Eject</a></td></tr>`,
			node.ID[:min(len(node.ID), 12)], node.Country, node.City, node.ISP, node.Reputation, load,
			node.LastSeen.Format("15:04:05"), node.ID, node.ID)
	}

	var cdRows string
	for domain, ids := range cooldowns {
		cdRows += fmt.Sprintf("<tr><td>%s</td><td>%v</td></tr>", domain, ids)
	}

	html := fmt.Sprintf(`<!DOCTYPE html><html><head><title>ProxyMesh</title>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0f0f1a;color:#e0e0e0;min-height:100vh}
.header{background:linear-gradient(135deg,#1a1a2e,#16213e);padding:24px 32px;border-bottom:1px solid #2a2a4a}
.header h1{color:#e94560;font-size:28px;font-weight:700}
.header p{color:#888;margin-top:4px;font-size:14px}
.container{max-width:1400px;margin:0 auto;padding:24px}
.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px;margin-bottom:24px}
.stat-card{background:#16213e;border-radius:12px;padding:20px;border:1px solid #2a2a4a}
.stat-card .value{font-size:36px;font-weight:700;color:#e94560;margin:8px 0}
.stat-card .label{color:#888;font-size:13px;text-transform:uppercase;letter-spacing:1px}
.stat-card.green .value{color:#4caf50}
.stat-card.orange .value{color:#ff9800}
.section{background:#16213e;border-radius:12px;padding:20px;margin-bottom:20px;border:1px solid #2a2a4a}
.section h2{color:#e94560;font-size:18px;margin-bottom:16px;font-weight:600}
table{width:100%%;border-collapse:collapse}
th{background:#1a1a2e;color:#e94560;font-size:12px;text-transform:uppercase;letter-spacing:.5px;padding:10px 12px;text-align:left;border-bottom:1px solid #2a2a4a}
td{padding:10px 12px;border-bottom:1px solid #1e1e3a;font-size:13px}
tr:hover{background:#1e1e3a}
tr:last-child td{border-bottom:none}
.badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:11px;font-weight:600}
.badge.active{background:#1b5e20;color:#4caf50}
.badge.inactive{background:#b71c1c;color:#ef5350}
a{color:#e94560;text-decoration:none;font-weight:500}
a:hover{text-decoration:underline}
pre{background:#1a1a2e;padding:12px;border-radius:8px;overflow-x:auto;font-size:12px;color:#888}
.footer{text-align:center;padding:20px;color:#555;font-size:12px}
.empty{text-align:center;padding:40px;color:#555}
</style></head><body>
<div class="header"><h1>🛡️ ProxyMesh</h1><p>Hybrid Proxy Network Dashboard</p></div>
<div class="container">
<div class="stats">
<div class="stat-card"><div class="label">Total Nodes</div><div class="value">%d</div></div>
<div class="stat-card green"><div class="label">Active</div><div class="value">%d</div></div>
<div class="stat-card orange"><div class="label">Cooldowns</div><div class="value">%d</div></div>
</div>
<div class="section"><h2>📡 Nodes (%d)</h2>
<table><tr><th>ID</th><th>Country</th><th>City</th><th>ISP</th><th>Reputation</th><th>Load</th><th>Last Seen</th><th>Actions</th></tr>
%s</table></div>
<div class="section"><h2>⏱️ Domain Cooldowns (%d)</h2>
<table><tr><th>Domain</th><th>Node IDs</th></tr>%s</table></div>
</div>
<div class="footer">ProxyMesh %s</div></body></html>`,
		len(nodes), len(nodes), len(cooldowns), len(nodes), rows, len(cooldowns), cdRows, time.Now().Format("2006-01-02"))

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
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
