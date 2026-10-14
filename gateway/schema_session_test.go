package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSchemaValidator_New(t *testing.T) {
	sv := NewSchemaValidator()
	if sv == nil {
		t.Fatal("NewSchemaValidator returned nil")
	}
	if sv.schemas == nil {
		t.Error("expected schemas map to be initialized")
	}
}

func TestSchemaValidator_RegisterAndGet(t *testing.T) {
	sv := NewSchemaValidator()
	schema := map[string]interface{}{
		"name": map[string]interface{}{"required": true},
	}
	sv.RegisterSchema("/api/test", "POST", schema)

	err := sv.Validate("/api/test", "POST", map[string]interface{}{"name": "value"})
	if err != nil {
		t.Errorf("expected validation pass, got %v", err)
	}

	err = sv.Validate("/api/test", "POST", map[string]interface{}{})
	if err == nil {
		t.Error("expected validation error for missing required field")
	}
}

func TestSchemaValidator_Validate_NoSchema(t *testing.T) {
	sv := NewSchemaValidator()
	err := sv.Validate("/unknown", "GET", map[string]interface{}{})
	if err != nil {
		t.Errorf("expected nil for unknown schema, got %v", err)
	}
}

func TestSchemaValidator_Middleware_GET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sv := NewSchemaValidator()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	sv.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected GET request to pass through")
	}
}

func TestSchemaValidator_Middleware_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sv := NewSchemaValidator()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)

	sv.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected invalid JSON to pass through (no schema)")
	}
}

func TestSchemaValidator_CreateSchemaHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sv := NewSchemaValidator()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"endpoint":"/test","method":"POST","schema":{"name":{"required":true}}}`
	c.Request = httptest.NewRequest("POST", "/api/admin/schemas", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	sv.createSchemaHandler(c)
	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestSchemaValidator_CreateSchemaHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sv := NewSchemaValidator()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/schemas", strings.NewReader(`{invalid}`))
	c.Request.Header.Set("Content-Type", "application/json")

	sv.createSchemaHandler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSchemaValidator_ListSchemasHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sv := NewSchemaValidator()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/schemas", nil)

	sv.listSchemasHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestSchemaValidator_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sv := NewSchemaValidator()
	router := gin.New()
	sv.RegisterRoutes(router)
}

func TestAdminSessionManager_New(t *testing.T) {
	asm := NewAdminSessionManager(0)
	if asm == nil {
		t.Fatal("NewAdminSessionManager returned nil")
	}
	if asm.timeout != 24*time.Hour {
		t.Errorf("expected default timeout 24h, got %v", asm.timeout)
	}
}

func TestAdminSessionManager_New_CustomTimeout(t *testing.T) {
	asm := NewAdminSessionManager(time.Hour)
	if asm.timeout != time.Hour {
		t.Errorf("expected timeout 1h, got %v", asm.timeout)
	}
}

func TestAdminSessionManager_CreateAndGet(t *testing.T) {
	asm := NewAdminSessionManager(time.Hour)
	id := asm.CreateSession("admin", "127.0.0.1", "test-agent")
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}

	session := asm.GetSession(id)
	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.Username != "admin" {
		t.Errorf("expected username 'admin', got %s", session.Username)
	}
	if session.IP != "127.0.0.1" {
		t.Errorf("expected IP '127.0.0.1', got %s", session.IP)
	}
}

func TestAdminSessionManager_GetSession_NotFound(t *testing.T) {
	asm := NewAdminSessionManager(time.Hour)
	session := asm.GetSession("nonexistent")
	if session != nil {
		t.Error("expected nil for nonexistent session")
	}
}

func TestAdminSessionManager_GetSession_Expired(t *testing.T) {
	asm := NewAdminSessionManager(1 * time.Nanosecond)
	id := asm.CreateSession("admin", "127.0.0.1", "test-agent")
	time.Sleep(10 * time.Millisecond)
	session := asm.GetSession(id)
	if session != nil {
		t.Error("expected nil for expired session")
	}
}

func TestAdminSessionManager_DeleteSession(t *testing.T) {
	asm := NewAdminSessionManager(time.Hour)
	id := asm.CreateSession("admin", "127.0.0.1", "test-agent")
	asm.DeleteSession(id)

	session := asm.GetSession(id)
	if session != nil {
		t.Error("expected nil after deletion")
	}
}

func TestAdminSessionManager_ListSessions(t *testing.T) {
	asm := NewAdminSessionManager(time.Hour)
	asm.CreateSession("admin1", "127.0.0.1", "agent1")
	time.Sleep(time.Millisecond)
	asm.CreateSession("admin2", "127.0.0.1", "agent2")

	sessions := asm.ListSessions()
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestAdminSessionManager_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	asm := NewAdminSessionManager(time.Hour)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	asm.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected request without session header to pass through")
	}
}

func TestAdminSessionManager_Middleware_InvalidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	asm := NewAdminSessionManager(time.Hour)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Admin-Session", "invalid-session")

	asm.Middleware()(c)
	if !c.IsAborted() {
		t.Error("expected request with invalid session to be aborted")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAdminSessionManager_Middleware_ValidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	asm := NewAdminSessionManager(time.Hour)
	id := asm.CreateSession("admin", "127.0.0.1", "test-agent")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Admin-Session", id)

	asm.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected request with valid session to pass through")
	}
	if c.GetString("admin_user") != "admin" {
		t.Errorf("expected admin_user 'admin', got %s", c.GetString("admin_user"))
	}
}

func TestAdminSessionManager_CreateSessionHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	asm := NewAdminSessionManager(time.Hour)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/sessions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Body = nil

	// Use a valid body
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("POST", "/api/admin/sessions", nil)
	c2.Request.Header.Set("Content-Type", "application/json")

	asm.createSessionHandler(c2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing username, got %d", w2.Code)
	}
}

func TestAdminSessionManager_ListSessionsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	asm := NewAdminSessionManager(time.Hour)
	asm.CreateSession("admin", "127.0.0.1", "agent")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/sessions", nil)

	asm.listSessionsHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAdminSessionManager_DeleteSessionHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	asm := NewAdminSessionManager(time.Hour)
	id := asm.CreateSession("admin", "127.0.0.1", "agent")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("DELETE", "/api/admin/sessions/"+id, nil)
	c.Params = []gin.Param{{Key: "id", Value: id}}

	asm.deleteSessionHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAdminSessionManager_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	asm := NewAdminSessionManager(time.Hour)
	router := gin.New()
	asm.RegisterRoutes(router)
}

func TestAdminSession_Struct(t *testing.T) {
	session := AdminSession{
		ID:         "sess-1",
		Username:   "admin",
		IP:         "127.0.0.1",
		UserAgent:  "test",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
		LastActive: time.Now(),
	}
	if session.ID != "sess-1" {
		t.Error("unexpected ID")
	}
	if session.Username != "admin" {
		t.Error("unexpected Username")
	}
}
