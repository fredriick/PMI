package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewWebUI(t *testing.T) {
	w := NewWebUI(nil)
	if w == nil {
		t.Fatal("NewWebUI() returned nil")
	}
	if w.mm != nil {
		t.Error("expected nil mm")
	}
}

func TestWebUI_RegisterRoutes(t *testing.T) {
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
	for path := range found {
		if !expected[path] {
			t.Errorf("unexpected route: %s", path)
		}
	}
}

func TestMin(t *testing.T) {
	tcs := []struct{ a, b, want int }{
		{3, 5, 3},
		{7, 2, 2},
		{4, 4, 4},
		{0, 1, 0},
	}
	for _, tc := range tcs {
		got := min(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("min(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestEjectNodeHandler_MissingID(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes//eject", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing node id, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestResetCircuitBreakerHandler_MissingID(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes//reset", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing node id, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDashboardResponse_ServesStaticFile(t *testing.T) {
	relPath := filepath.Join("web", "dashboard", "index.html")
	dir := filepath.Dir(relPath)
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	err := os.WriteFile(relPath, []byte("<html><body>ok</body></html>"), 0644)
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
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "<html><body>ok</body></html>" {
		t.Errorf("unexpected body: %s", rr.Body.String())
	}
}

func TestDashboardResponse_Returns404WhenMissing(t *testing.T) {
	w := NewWebUI(nil)
	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/dashboard", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when dashboard file is missing, got %d: %s", rr.Code, rr.Body.String())
	}
}
