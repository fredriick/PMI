package gateway

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"proxymesh/internal/config"
	"proxymesh/internal/models"
	"proxymesh/matchmaker"
)

type Gateway struct {
	router                  *gin.Engine
	matchmaker              *matchmaker.Matchmaker
	compliance              *ComplianceService
	config                  *config.GatewayConfig
	circuitBreakerThreshold int
	nodeFailures            map[string]int
	mu                      sync.RWMutex
}

func NewGateway(cfg *config.Config, mm *matchmaker.Matchmaker, comp *ComplianceService) *Gateway {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	gw := &Gateway{
		router:                  router,
		matchmaker:              mm,
		compliance:              comp,
		config:                  &cfg.Gateway,
		circuitBreakerThreshold: cfg.Gateway.CircuitBreakerThreshold,
		nodeFailures:            make(map[string]int),
	}

	gw.setupRoutes()
	return gw
}

func (g *Gateway) setupRoutes() {
	g.router.Use(g.authMiddleware())
	g.router.Any("/:path", g.proxyHandler)
	g.router.Any("/", g.proxyHandler)
}

func (g *Gateway) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.Request.Header.Get("Proxy-Authorization")
		}

		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authentication",
			})
			return
		}

		if !g.validateAPIKey(authHeader) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
			})
			return
		}

		c.Next()
	}
}

func (g *Gateway) validateAPIKey(auth string) bool {
	return strings.HasPrefix(auth, "Bearer ") || strings.Contains(auth, ":")
}

func (g *Gateway) proxyHandler(c *gin.Context) {
	target := c.Param("path")
	if target == "" {
		target = c.Query("url")
	}

	if target == "" {
		target = c.Request.URL.Path
	}

	user, pass, hasAuth := c.Request.BasicAuth()
	if !hasAuth {
		user = c.Query("user")
		pass = c.Query("pass")
	}

	req := &models.ProxyRequest{
		User:      user,
		Password:  pass,
		Target:    target,
		Country:   g.extractTarget(c, "country"),
		City:      g.extractTarget(c, "city"),
		SessionID: c.Query("session"),
	}

	if g.compliance.IsBlocked(req.Target) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Target domain is blocked",
		})
		return
	}

	if g.compliance.KYCRequired && !g.validateKYC(user) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "KYC verification required",
		})
		return
	}

	start := time.Now()

	node, err := g.matchmaker.SelectNode(req.Country, req.City, req.Target)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error": err.Error(),
		})
		return
	}

	latency := time.Since(start).Milliseconds()

	c.Header("X-Proxy-Node-ID", node.ID)
	c.Header("X-Proxy-Latency", fmt.Sprintf("%d", latency))

	localAddr := g.bindToIPv6Subnet(node)
	c.Header("X-Proxy-Local-Addr", localAddr)

	g.forwardRequest(c, node, localAddr)
}

func (g *Gateway) extractTarget(c *gin.Context, targetType string) string {
	path := c.Param("path")
	prefix := fmt.Sprintf("%s-", targetType)

	if idx := strings.Index(path, prefix); idx != -1 {
		rest := path[idx+len(prefix):]
		if endIdx := strings.Index(rest, "-"); endIdx != -1 {
			return rest[:endIdx]
		}
		return rest
	}

	return c.Query(targetType)
}

func (g *Gateway) bindToIPv6Subnet(node *models.Node) string {
	if node.IPv6Subnet != "" {
		ip := net.ParseIP(node.IPv6Subnet)
		if ip != nil {
			ip[15] = byte(time.Now().UnixNano() & 0xFF)
			return ip.String()
		}
	}
	return node.IP
}

func (g *Gateway) forwardRequest(c *gin.Context, node *models.Node, localAddr string) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	req, err := http.NewRequest("CONNECT", fmt.Sprintf("http://%s", node.IP), nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create request",
		})
		return
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		g.recordFailure(node.ID)
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
			"error": "Failed to connect to node",
		})
		return
	}

	g.recordSuccess(node.ID)
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), nil)
}

func (g *Gateway) recordFailure(nodeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodeFailures[nodeID]++
	if g.nodeFailures[nodeID] >= g.circuitBreakerThreshold {
		g.matchmaker.RecordFailure(nodeID)
	}
}

func (g *Gateway) recordSuccess(nodeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodeFailures[nodeID] = 0
	g.matchmaker.RecordSuccess(nodeID)
}

func (g *Gateway) validateKYC(username string) bool {
	return true
}

func (g *Gateway) Start() error {
	addr := fmt.Sprintf("%s:%d", g.config.Host, g.config.Port)
	return g.router.Run(addr)
}
