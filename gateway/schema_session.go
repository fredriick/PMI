package gateway

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type SchemaValidator struct {
	mu      sync.RWMutex
	schemas map[string]map[string]interface{}
}

func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		schemas: make(map[string]map[string]interface{}),
	}
}

func (sv *SchemaValidator) RegisterSchema(endpoint, method string, schema map[string]interface{}) {
	key := fmt.Sprintf("%s:%s", method, endpoint)
	sv.mu.Lock()
	defer sv.mu.Unlock()
	sv.schemas[key] = schema
}

func (sv *SchemaValidator) Validate(endpoint, method string, data map[string]interface{}) error {
	key := fmt.Sprintf("%s:%s", method, endpoint)
	sv.mu.RLock()
	schema, exists := sv.schemas[key]
	sv.mu.RUnlock()

	if !exists {
		return nil
	}

	for field, rules := range schema {
		if required, ok := rules.(map[string]interface{})["required"].(bool); ok && required {
			if _, present := data[field]; !present {
				return fmt.Errorf("missing required field: %s", field)
			}
		}
	}

	return nil
}

func (sv *SchemaValidator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "DELETE" {
			c.Next()
			return
		}

		endpoint := c.Request.URL.Path
		method := c.Request.Method

		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.Next()
			return
		}

		if err := sv.Validate(endpoint, method, data); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":  "Validation failed",
				"detail": err.Error(),
			})
			return
		}

		c.Next()
	}
}

func (sv *SchemaValidator) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/schemas", sv.createSchemaHandler)
	r.GET("/api/admin/schemas", sv.listSchemasHandler)
}

func (sv *SchemaValidator) createSchemaHandler(c *gin.Context) {
	var req struct {
		Endpoint string                 `json:"endpoint"`
		Method   string                 `json:"method"`
		Schema   map[string]interface{} `json:"schema"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	sv.RegisterSchema(req.Endpoint, req.Method, req.Schema)
	c.JSON(201, gin.H{"status": "created"})
}

func (sv *SchemaValidator) listSchemasHandler(c *gin.Context) {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	c.JSON(200, gin.H{"schemas": sv.schemas})
}

type AdminSession struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastActive time.Time `json:"last_active"`
}

type AdminSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*AdminSession
	timeout  time.Duration
}

func NewAdminSessionManager(timeout time.Duration) *AdminSessionManager {
	if timeout == 0 {
		timeout = 24 * time.Hour
	}

	asm := &AdminSessionManager{
		sessions: make(map[string]*AdminSession),
		timeout:  timeout,
	}

	go asm.cleanupLoop()

	return asm
}

func (asm *AdminSessionManager) CreateSession(username, ip, userAgent string) string {
	sessionID := fmt.Sprintf("admin-%d", time.Now().UnixNano())

	session := &AdminSession{
		ID:         sessionID,
		Username:   username,
		IP:         ip,
		UserAgent:  userAgent,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(asm.timeout),
		LastActive: time.Now(),
	}

	asm.mu.Lock()
	asm.sessions[sessionID] = session
	asm.mu.Unlock()

	return sessionID
}

func (asm *AdminSessionManager) GetSession(sessionID string) *AdminSession {
	asm.mu.RLock()
	defer asm.mu.RUnlock()

	session, exists := asm.sessions[sessionID]
	if !exists {
		return nil
	}

	if time.Now().After(session.ExpiresAt) {
		return nil
	}

	session.LastActive = time.Now()
	return session
}

func (asm *AdminSessionManager) DeleteSession(sessionID string) {
	asm.mu.Lock()
	defer asm.mu.Unlock()
	delete(asm.sessions, sessionID)
}

func (asm *AdminSessionManager) ListSessions() []*AdminSession {
	asm.mu.RLock()
	defer asm.mu.RUnlock()

	sessions := make([]*AdminSession, 0, len(asm.sessions))
	for _, s := range asm.sessions {
		if time.Now().Before(s.ExpiresAt) {
			sessions = append(sessions, s)
		}
	}
	return sessions
}

func (asm *AdminSessionManager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		asm.mu.Lock()
		now := time.Now()
		for id, session := range asm.sessions {
			if now.After(session.ExpiresAt) {
				delete(asm.sessions, id)
			}
		}
		asm.mu.Unlock()
	}
}

func (asm *AdminSessionManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.GetHeader("X-Admin-Session")
		if sessionID == "" {
			c.Next()
			return
		}

		session := asm.GetSession(sessionID)
		if session == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired session",
			})
			return
		}

		c.Set("admin_user", session.Username)
		c.Next()
	}
}

func (asm *AdminSessionManager) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/sessions", asm.createSessionHandler)
	r.GET("/api/admin/sessions", asm.listSessionsHandler)
	r.DELETE("/api/admin/sessions/:id", asm.deleteSessionHandler)
}

func (asm *AdminSessionManager) createSessionHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	sessionID := asm.CreateSession(req.Username, c.ClientIP(), c.Request.UserAgent())
	c.JSON(201, gin.H{"session_id": sessionID})
}

func (asm *AdminSessionManager) listSessionsHandler(c *gin.Context) {
	sessions := asm.ListSessions()
	c.JSON(200, gin.H{"sessions": sessions})
}

func (asm *AdminSessionManager) deleteSessionHandler(c *gin.Context) {
	sessionID := c.Param("id")
	asm.DeleteSession(sessionID)
	c.JSON(200, gin.H{"status": "deleted"})
}
