package gateway

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"proxymesh/gateway/web"
	"proxymesh/internal/config"
	"proxymesh/internal/models"
	"proxymesh/matchmaker"
)



type Gateway struct {
	router                  *gin.Engine
	matchmaker              *matchmaker.Matchmaker
	compliance              *ComplianceService
	rateLimiter             RateLimiter
	tracing                 *Tracer
	webUI                   *web.WebUI
	apiKeyService           *APIKeyService
	prometheusPusher        *PrometheusPusher
	auditLogger             *AuditLogger
	config                  *config.GatewayConfig
	connPool                *ConnPool
	wsHub                   *Hub
	circuitBreakerThreshold int
	nodeFailures            map[string]int
	nodeConnections         map[string]int64
	mu                      sync.RWMutex
}

func (g *Gateway) Router() *gin.Engine {
	return g.router
}

func (g *Gateway) SetAPIKeyService(svc *APIKeyService) {
	g.apiKeyService = svc
}

func (g *Gateway) SetPrometheusPusher(p *PrometheusPusher) {
	g.prometheusPusher = p
}

func (g *Gateway) ReloadCompliance(cfg *config.ComplianceConfig) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.compliance = NewComplianceService(cfg)
}

func NewGateway(cfg *config.Config, mm *matchmaker.Matchmaker, comp *ComplianceService, tracer *Tracer, rateLimiter RateLimiter) *Gateway {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	connPool := NewConnPool(10, 5*time.Second)
	gw := &Gateway{
		router:                  router,
		matchmaker:              mm,
		compliance:              comp,
		rateLimiter:             rateLimiter,
		tracing:                 tracer,
		webUI:                   web.NewWebUI(mm),
		apiKeyService:           nil,
		prometheusPusher:        nil,
		auditLogger:             nil,
		config:                  &cfg.Gateway,
		connPool:                connPool,
		wsHub:                   NewHub(),
		circuitBreakerThreshold: cfg.Gateway.CircuitBreakerThreshold,
		nodeFailures:            make(map[string]int),
		nodeConnections:         make(map[string]int64),
	}

	gw.setupRoutes()
	gw.setupWebSocket()
	go gw.wsHub.Run()
	return gw
}

func (g *Gateway) setupRoutes() {
	InitLogger(LevelInfo, "")
	SetupMetricsRoutes(g.router)
	g.router.Use(RequestLogger())
	g.router.Use(g.rateLimiter.Middleware())

	g.router.GET("/health", g.healthHandler)
	g.router.GET("/dashboard", g.serveDashboard)
	g.router.GET("/v1/health", g.healthHandler)
	g.router.GET("/v1/metrics", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain")
		c.String(200, metrics.String())
	})
	g.router.GET("/v1/dashboard", g.serveDashboard)

	g.webUI.RegisterRoutes(g.router)

	g.router.Use(g.authMiddleware())
	g.router.Use(g.tracingMiddleware())

	g.router.Any("/v1/:path", g.proxyHandler)
	g.router.Any("/v1/", g.proxyHandler)
	g.router.Any("/:path", g.proxyHandler)
	g.router.Any("/", g.proxyHandler)
}

func (g *Gateway) healthHandler(c *gin.Context) {
	status := gin.H{
		"status":  "healthy",
		"version": "1.0.0",
	}
	c.JSON(http.StatusOK, status)
}

func (g *Gateway) serveDashboard(c *gin.Context) {
	g.webUI.DashboardHandler(c)
}

func (g *Gateway) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if path == "/health" || path == "/metrics" || path == "/dashboard" ||
			path == "/peer" || strings.HasPrefix(path, "/peer/") ||
			strings.HasPrefix(path, "/api/") ||
			path == "/v1/health" || path == "/v1/metrics" || path == "/v1/dashboard" ||
			strings.HasPrefix(path, "/v1/api/") {
			c.Next()
			return
		}

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

		if g.apiKeyService != nil {
			key := extractBearerToken(authHeader)
			if key == "" {
				key = authHeader
			}
			valid, err := g.apiKeyService.ValidateKey(key)
			if err != nil || !valid {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid API key",
				})
				return
			}
		}

		c.Next()
	}
}

func (g *Gateway) tracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if g.tracing != nil {
			ctx, span := g.tracing.StartSpan(c.Request.Context(), "gateway.request",
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
			)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			span.End()
		} else {
			c.Next()
		}
	}
}

func extractBearerToken(auth string) string {
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
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

	sessionID := c.Query("session")
	if sessionID == "" {
		sessionID = g.extractSessionFromPath(target)
	}

	req := &models.ProxyRequest{
		User:      user,
		Password:  pass,
		Target:    target,
		Country:   g.extractTarget(c, "country"),
		City:      g.extractTarget(c, "city"),
		SessionID: sessionID,
	}

	metrics.IncRequestsTotal()

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

	var node *models.Node
	var err error

	if req.SessionID != "" {
		nodeID, err := g.matchmaker.GetSessionNode(req.SessionID)
		if err == nil && nodeID != "" {
			node, err = g.matchmaker.GetNodeStatus(nodeID)
			if err == nil && node != nil && g.isNodeHealthy(node.ID) {
				c.Header("X-Session-Cached", "true")
			} else {
				node = nil
			}
		}
	}

	if node == nil {
		node, err = g.matchmaker.SelectNode(req.Country, req.City, req.Target)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": err.Error(),
			})
			return
		}

		if req.SessionID != "" {
			g.matchmaker.SetSessionNode(req.SessionID, node.ID, 3600)
		}
	}

	latency := time.Since(start).Milliseconds()

	metrics.IncNodesSelected()
	metrics.AddLatency(latency)

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

func (g *Gateway) extractSessionFromPath(path string) string {
	if path == "" {
		return ""
	}
	prefix := "session-"
	if idx := strings.Index(path, prefix); idx != -1 {
		rest := path[idx+len(prefix):]
		if endIdx := strings.Index(rest, "-"); endIdx != -1 {
			return rest[:endIdx]
		}
		if endIdx := strings.Index(rest, "/"); endIdx != -1 {
			return rest[:endIdx]
		}
		return rest
	}
	return ""
}

func (g *Gateway) isNodeHealthy(nodeID string) bool {
	return true
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
	defer g.matchmaker.DecrementNodeLoad(node.ID)

	target := c.Query("url")
	if target == "" {
		target = c.Param("path")
	}

	targetHost := g.extractHostFromTarget(target)
	if targetHost == "" {
		targetHost = "80.80.80.80:80"
	}

	conn, err := g.connPool.Get(node.IP)
	if err != nil {
		g.recordFailure(node.ID)
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
			"error": "Failed to connect to node: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", targetHost, targetHost)
	_, err = conn.Write([]byte(connectReq))
	if err != nil {
		g.recordFailure(node.ID)
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
			"error": "Failed to send CONNECT to node",
		})
		return
	}

	buf := make([]byte, 1024)
	conn.Read(buf)

	g.recordSuccess(node.ID)
	c.JSON(http.StatusOK, gin.H{
		"status":    "connected",
		"node_id":   node.ID,
		"localAddr": localAddr,
		"target":    targetHost,
	})
}

func (g *Gateway) extractHostFromTarget(target string) string {
	if target == "" {
		return ""
	}

	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")

	if idx := strings.Index(target, "/"); idx != -1 {
		target = target[:idx]
	}

	if idx := strings.Index(target, "?"); idx != -1 {
		target = target[:idx]
	}

	return target
}

func (g *Gateway) recordFailure(nodeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodeFailures[nodeID]++
	metrics.IncRequestsFailed()
	if g.nodeFailures[nodeID] >= g.circuitBreakerThreshold {
		g.matchmaker.RecordFailure(nodeID)
	}
}

func (g *Gateway) recordSuccess(nodeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodeFailures[nodeID] = 0
	metrics.IncRequestsSuccess()
	g.matchmaker.RecordSuccess(nodeID)
}

func (g *Gateway) validateKYC(username string) bool {
	return true
}

func (g *Gateway) StartServer() (*http.Server, error) {
	addr := fmt.Sprintf("%s:%d", g.config.Host, g.config.Port)

	server := &http.Server{
		Addr:    addr,
		Handler: g.router,
	}

	if g.config.MTLSEnabled {
		tlsConfig, err := g.buildTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		server.TLSConfig = tlsConfig
		return server, nil
	}

	return server, nil
}

func (g *Gateway) buildTLSConfig() (*tls.Config, error) {
	caCert, err := os.ReadFile(g.config.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add CA cert to pool")
	}

	return &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequestClientCert,
	}, nil
}

func (g *Gateway) Start() error {
	server, err := g.StartServer()
	if err != nil {
		return err
	}

	if g.config.MTLSEnabled {
		return server.ListenAndServeTLS(g.config.ServerCertPath, g.config.ServerKeyPath)
	}
	return server.ListenAndServe()
}

func (g *Gateway) Shutdown(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	Info("Gateway shutting down", nil)
	return nil
}