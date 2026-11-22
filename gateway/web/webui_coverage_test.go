package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"proxymesh/matchmaker"
)

func TestWebUI_NodesAPIHandler_NilMatchmaker(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/nodes", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for nil matchmaker, got %d", rr.Code)
	}
}

func TestWebUI_CooldownsAPIHandler_NilMatchmaker(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/cooldowns", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for nil matchmaker, got %d", rr.Code)
	}
}

func TestWebUI_HealthAPIHandler_NilMatchmaker(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestWebUI_NodeHealthAPIHandler_NilMatchmaker(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/health/node-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nil matchmaker, got %d", rr.Code)
	}
}

func TestWebUI_NodeHealthHistoryAPIHandler_NilMatchmaker(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/health/node-1/history", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestWebUI_ResetCircuitBreakerHandler_ValidID(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes/node-1/reset", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for nil matchmaker, got %d", rr.Code)
	}
}

func TestWebUI_EjectNodeHandler_ValidID(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes/node-1/eject", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for nil matchmaker, got %d", rr.Code)
	}
}

func TestNewWebUI_WithMatchmaker(t *testing.T) {
	mm := &matchmaker.Matchmaker{}
	w := NewWebUI(mm)

	if w == nil {
		t.Fatal("NewWebUI() returned nil")
	}
	if w.mm != mm {
		t.Error("expected mm to match")
	}
}

func TestDashboardResponse_ContentType(t *testing.T) {
	relPath := filepath.Join("web", "dashboard", "index.html")
	dir := filepath.Dir(relPath)
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	err := os.WriteFile(relPath, []byte("<html><body>test</body></html>"), 0644)
	if err != nil {
		t.Fatalf("failed to write temp dashboard: %v", err)
	}

	w := NewWebUI(nil)
	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/dashboard", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "<html><body>test</body></html>" {
		t.Errorf("unexpected body: %s", rr.Body.String())
	}
}

func TestRegisterRoutes_AllRoutes(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	routes := r.Routes()
	found := map[string]bool{}
	for _, route := range routes {
		found[route.Method+" "+route.Path] = true
	}

	expected := map[string]bool{
		"GET /web/dashboard":                  true,
		"GET /web/api/nodes":                  true,
		"GET /web/api/cooldowns":              true,
		"GET /web/api/health":                 true,
		"GET /web/api/health/:nodeID":         true,
		"GET /web/api/health/:nodeID/history": true,
		"POST /web/api/nodes/:id/reset":       true,
		"POST /web/api/nodes/:id/eject":       true,
	}
	for path := range expected {
		if !found[path] {
			t.Errorf("missing route: %s", path)
		}
	}
}
