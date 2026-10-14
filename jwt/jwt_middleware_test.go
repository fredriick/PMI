package jwt

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestJWTAuthMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	JWTAuthMiddleware(svc)(c)
	if !c.IsAborted() {
		t.Error("expected request to be aborted without token")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestJWTAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService()
	token, _ := svc.GenerateToken("user-1", []string{"admin"}, nil, "node-1")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	JWTAuthMiddleware(svc)(c)
	if c.IsAborted() {
		t.Error("expected request to pass with valid token")
	}
	if GetUserID(c) != "user-1" {
		t.Errorf("expected user_id 'user-1', got %s", GetUserID(c))
	}
	if GetNodeID(c) != "node-1" {
		t.Errorf("expected node_id 'node-1', got %s", GetNodeID(c))
	}
	if !IsAdmin(c) {
		t.Error("expected IsAdmin to be true")
	}
}

func TestJWTAuthMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewJWTService("secret", -time.Hour)
	token, _ := svc.GenerateToken("user-1", []string{"user"}, nil, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	JWTAuthMiddleware(svc)(c)
	if !c.IsAborted() {
		t.Error("expected request to be aborted with expired token")
	}
}

func TestJWTAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid-token")

	JWTAuthMiddleware(svc)(c)
	if !c.IsAborted() {
		t.Error("expected request to be aborted with invalid token")
	}
}

func TestJWTAuthMiddleware_XPeerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService()
	token, _ := svc.GeneratePeerToken("node-1")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Peer-Token", token)

	JWTAuthMiddleware(svc)(c)
	if c.IsAborted() {
		t.Error("expected request to pass with X-Peer-Token")
	}
	if GetNodeID(c) != "node-1" {
		t.Errorf("expected node_id 'node-1', got %s", GetNodeID(c))
	}
}

func TestOptionalJWTAuth_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	OptionalJWTAuth(svc)(c)
	if c.IsAborted() {
		t.Error("expected request to pass without token")
	}
}

func TestOptionalJWTAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService()
	token, _ := svc.GenerateToken("user-1", []string{"admin"}, nil, "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	OptionalJWTAuth(svc)(c)
	if c.IsAborted() {
		t.Error("expected request to pass with valid token")
	}
	if GetUserID(c) != "user-1" {
		t.Errorf("expected user_id 'user-1', got %s", GetUserID(c))
	}
}

func TestOptionalJWTAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid-token")

	OptionalJWTAuth(svc)(c)
	if c.IsAborted() {
		t.Error("expected request to pass (optional auth)")
	}
}

func TestRequireRole_NoRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	RequireRole(nil, "admin")(c)
	if !c.IsAborted() {
		t.Error("expected request to be aborted without roles")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestRequireRole_InsufficientPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("roles", []string{"user"})

	RequireRole(nil, "admin")(c)
	if !c.IsAborted() {
		t.Error("expected request to be aborted for insufficient permissions")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestRequireRole_SufficientPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("roles", []string{"admin", "user"})

	RequireRole(nil, "admin")(c)
	if c.IsAborted() {
		t.Error("expected request to pass with sufficient permissions")
	}
}

func TestRequireRole_InvalidRolesType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("roles", "not-a-slice")

	RequireRole(nil, "admin")(c)
	if !c.IsAborted() {
		t.Error("expected request to be aborted with invalid roles type")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestGetUserID_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	if GetUserID(c) != "" {
		t.Error("expected empty string for missing user_id")
	}
}

func TestGetRoles_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	if GetRoles(c) != nil {
		t.Error("expected nil for missing roles")
	}
}

func TestGetRoles_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("roles", []string{"admin", "user"})

	roles := GetRoles(c)
	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}
}

func TestGetNodeID_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	if GetNodeID(c) != "" {
		t.Error("expected empty string for missing node_id")
	}
}

func TestIsAdmin_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	if IsAdmin(c) {
		t.Error("expected false for missing is_admin")
	}
}
