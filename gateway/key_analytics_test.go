package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestKeyAnalytics_New(t *testing.T) {
	ka := NewKeyAnalytics(nil)
	if ka == nil {
		t.Fatal("NewKeyAnalytics returned nil")
	}
}

func TestKeyAnalytics_RecordAndGet(t *testing.T) {
	ka := NewKeyAnalytics(nil)
	ka.RecordRequest("key-1", 100, 50, false)
	ka.RecordRequest("key-1", 200, 75, true)
	ka.RecordRequest("key-2", 50, 25, false)

	stat := ka.GetStats("key-1")
	if stat == nil {
		t.Fatal("expected stats for key-1")
	}
	if stat.Requests != 2 {
		t.Errorf("expected 2 requests, got %d", stat.Requests)
	}
	if stat.BytesSent != 300 {
		t.Errorf("expected 300 bytes sent, got %d", stat.BytesSent)
	}
	if stat.Errors != 1 {
		t.Errorf("expected 1 error, got %d", stat.Errors)
	}

	stat2 := ka.GetStats("key-2")
	if stat2.Requests != 1 {
		t.Errorf("expected 1 request for key-2, got %d", stat2.Requests)
	}
}

func TestKeyAnalytics_GetStats_Nonexistent(t *testing.T) {
	ka := NewKeyAnalytics(nil)
	stat := ka.GetStats("missing")
	if stat != nil {
		t.Error("expected nil for missing key")
	}
}

func TestKeyAnalytics_GetTopKeys(t *testing.T) {
	ka := NewKeyAnalytics(nil)
	ka.RecordRequest("a", 0, 0, false)
	ka.RecordRequest("a", 0, 0, false)
	ka.RecordRequest("b", 0, 0, false)
	ka.RecordRequest("c", 0, 0, false)
	ka.RecordRequest("c", 0, 0, false)
	ka.RecordRequest("c", 0, 0, false)

	top := ka.GetTopKeys(2)
	if len(top) != 2 {
		t.Errorf("expected 2 top keys, got %d", len(top))
	}
	if top[0].Key != "c" {
		t.Errorf("expected top key 'c', got %q", top[0].Key)
	}
	if top[0].Requests != 3 {
		t.Errorf("expected 3 requests for top key, got %d", top[0].Requests)
	}
}

func TestKeyAnalytics_GetTopKeys_LimitExceeds(t *testing.T) {
	ka := NewKeyAnalytics(nil)
	ka.RecordRequest("x", 0, 0, false)

	top := ka.GetTopKeys(100)
	if len(top) != 1 {
		t.Errorf("expected 1 key, got %d", len(top))
	}
}

func TestKeyAnalytics_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ka := NewKeyAnalytics(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	ka.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected request to pass through")
	}
}

func TestKeyAnalytics_ListKeysHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ka := NewKeyAnalytics(nil)
	ka.RecordRequest("key-a", 10, 5, false)
	ka.RecordRequest("key-b", 20, 10, false)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/analytics/keys?limit=5", nil)

	ka.listKeysHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestKeyAnalytics_GetKeyHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ka := NewKeyAnalytics(nil)
	ka.RecordRequest("key-c", 5, 3, false)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/analytics/keys/key-c", nil)
	c.Params = []gin.Param{{Key: "key", Value: "key-c"}}

	ka.getKeyHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/api/admin/analytics/keys/missing", nil)
	c2.Params = []gin.Param{{Key: "key", Value: "missing"}}
	ka.getKeyHandler(c2)
	if w2.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w2.Code)
	}
}
