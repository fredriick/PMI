package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRequestReplay_New(t *testing.T) {
	rr := NewRequestReplay(nil, time.Hour)
	if rr == nil {
		t.Fatal("NewRequestReplay returned nil")
	}
}

func TestRequestReplay_RecordAndGet(t *testing.T) {
	rr := NewRequestReplay(nil, time.Hour)

	req := httptest.NewRequest("POST", "http://example.com/api", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")

	id := rr.RecordRequest(req, []byte(`{"key":"value"}`), "10.0.0.1")
	if id == "" {
		t.Fatal("expected non-empty request ID")
	}

	replayReq := rr.GetRequest(id)
	if replayReq == nil {
		t.Fatal("expected to find recorded request")
	}
	if replayReq.Method != "POST" {
		t.Errorf("expected method POST, got %s", replayReq.Method)
	}
	if replayReq.ClientIP != "10.0.0.1" {
		t.Errorf("expected client IP 10.0.0.1, got %s", replayReq.ClientIP)
	}
}

func TestRequestReplay_GetRequest_NotFound(t *testing.T) {
	rr := NewRequestReplay(nil, time.Hour)
	req := rr.GetRequest("missing-id")
	if req != nil {
		t.Error("expected nil for missing request")
	}
}

func TestRequestReplay_ListRequests(t *testing.T) {
	rr := NewRequestReplay(nil, time.Hour)

	req1 := httptest.NewRequest("GET", "http://a.com", nil)
	req2 := httptest.NewRequest("POST", "http://b.com", nil)
	req3 := httptest.NewRequest("PUT", "http://c.com", nil)

	id1 := rr.RecordRequest(req1, nil, "ip-1")
	time.Sleep(time.Millisecond)
	id2 := rr.RecordRequest(req2, nil, "ip-2")
	time.Sleep(time.Millisecond)
	id3 := rr.RecordRequest(req3, nil, "ip-3")

	if id1 == id2 || id2 == id3 || id1 == id3 {
		t.Error("expected unique request IDs")
	}

	list := rr.ListRequests(2)
	if len(list) != 2 {
		t.Errorf("expected 2 requests, got %d", len(list))
	}
}

func TestRequestReplay_ListRequests_LimitZero(t *testing.T) {
	rr := NewRequestReplay(nil, time.Hour)
	req := httptest.NewRequest("GET", "http://a.com", nil)
	rr.RecordRequest(req, nil, "ip")

	list := rr.ListRequests(0)
	if len(list) != 1 {
		t.Errorf("expected 1 request with limit 0 (current behavior), got %d", len(list))
	}
}

func TestRequestReplay_ReplayRequest_NotFound(t *testing.T) {
	rr := NewRequestReplay(nil, time.Hour)
	err := rr.ReplayRequest("missing", "http://target.com")
	if err == nil {
		t.Error("expected error for missing request")
	}
}

func TestRequestReplay_CreateReplayHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rr := NewRequestReplay(nil, time.Hour)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/replay", strings.NewReader(`{"method":"GET","url":"http://a.com"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	rr.createReplayHandler(c)
	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestRequestReplay_ListReplayHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rr := NewRequestReplay(nil, time.Hour)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/replay", nil)

	rr.listReplayHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRequestReplay_ReplayHandler_NoTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rr := NewRequestReplay(nil, time.Hour)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/replay/some-id", nil)
	c.Params = []gin.Param{{Key: "id", Value: "some-id"}}

	rr.replayHandler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestEndpointSizeLimits_New(t *testing.T) {
	esl := NewEndpointSizeLimits()
	if esl == nil {
		t.Fatal("NewEndpointSizeLimits returned nil")
	}
}

func TestEndpointSizeLimits_SetAndGetLimit(t *testing.T) {
	esl := NewEndpointSizeLimits()
	esl.SetLimit("/api/test", "POST", 1024)

	limit := esl.GetLimit("/api/test", "POST")
	if limit != 1024 {
		t.Errorf("expected limit 1024, got %d", limit)
	}

	def := esl.GetLimit("/unknown", "GET")
	if def != 10*1024*1024 {
		t.Errorf("expected default 10MB, got %d", def)
	}
}

func TestEndpointSizeLimits_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	esl := NewEndpointSizeLimits()
	esl.SetLimit("/api/upload", "POST", 100)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/upload", nil)
	c.Request.ContentLength = 200

	esl.Middleware()(c)
	if !c.IsAborted() {
		t.Error("expected request aborted due to size limit")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", w.Code)
	}
}

func TestEndpointSizeLimits_Middleware_UnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	esl := NewEndpointSizeLimits()
	esl.SetLimit("/api/upload", "POST", 1000)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/upload", nil)
	c.Request.ContentLength = 100

	esl.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected request under limit to pass")
	}
}

func TestEndpointSizeLimits_SetLimitHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	esl := NewEndpointSizeLimits()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/limits", strings.NewReader(`{"endpoint":"/x","method":"GET","max_bytes":500}`))
	c.Request.Header.Set("Content-Type", "application/json")

	esl.setLimitHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestEndpointSizeLimits_ListLimitsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	esl := NewEndpointSizeLimits()
	esl.SetLimit("/a", "GET", 100)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/limits", nil)

	esl.listLimitsHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestReplayRequest_Struct(t *testing.T) {
	rr := &ReplayRequest{
		ID:        "req-1",
		Method:    "GET",
		URL:       "http://example.com",
		Headers:   map[string]string{"X-Test": "value"},
		Body:      "body",
		ClientIP:  "1.2.3.4",
		CreatedAt: time.Now(),
	}
	if rr.ID != "req-1" {
		t.Error("unexpected ID")
	}
	if rr.Method != "GET" {
		t.Error("unexpected method")
	}
	_ = io.Discard
}
