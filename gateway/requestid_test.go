package gateway

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestIDGeneratorCustomPrefixUnix(t *testing.T) {
	generator := NewRequestIDGenerator("pm", RequestIDFormatUnix, nil)

	id := generator.Generate()

	if !strings.HasPrefix(id, "pm-") {
		t.Fatalf("expected request id to start with pm-, got %q", id)
	}
	if matched, _ := regexp.MatchString(`^pm-\d+-\d+$`, id); !matched {
		t.Fatalf("expected unix request id format, got %q", id)
	}
}

func TestRequestIDGeneratorTimestampFormat(t *testing.T) {
	generator := NewRequestIDGenerator("trace", RequestIDFormatTimestamp, nil)

	id := generator.Generate()

	if matched, _ := regexp.MatchString(`^trace-\d+-\d+$`, id); !matched {
		t.Fatalf("expected timestamp request id format, got %q", id)
	}
}

func TestRequestIDGeneratorUUIDFormat(t *testing.T) {
	generator := NewRequestIDGenerator("pm", RequestIDFormatUUID, nil)

	id := generator.Generate()

	if matched, _ := regexp.MatchString(`^pm-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, id); !matched {
		t.Fatalf("expected uuid request id format, got %q", id)
	}
}

func TestRequestIDGeneratorMiddlewarePreservesIncomingID(t *testing.T) {
	router := gin.New()
	generator := NewRequestIDGenerator("pm", RequestIDFormatUnix, nil)
	router.Use(generator.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString("request_id"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "incoming-id")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "incoming-id" {
		t.Fatalf("expected incoming request id, got %q", w.Body.String())
	}
	if got := w.Header().Get("X-Request-ID"); got != "incoming-id" {
		t.Fatalf("expected response header incoming-id, got %q", got)
	}
}

func TestRequestIDConfigAdminHandlers(t *testing.T) {
	router := gin.New()
	generator := NewRequestIDGenerator("req", RequestIDFormatUnix, nil)
	generator.RegisterRoutes(router)

	updateReq := httptest.NewRequest(http.MethodPost, "/request-id", strings.NewReader(`{"prefix":"pm","format":"uuid"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes := httptest.NewRecorder()
	router.ServeHTTP(updateRes, updateReq)

	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateRes.Code, updateRes.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	getRes := httptest.NewRecorder()
	router.ServeHTTP(getRes, getReq)

	if getRes.Code != http.StatusOK {
		t.Fatalf("expected get 200, got %d: %s", getRes.Code, getRes.Body.String())
	}
	body := getRes.Body.String()
	if !strings.Contains(body, `"prefix":"pm"`) || !strings.Contains(body, `"format":"uuid"`) {
		t.Fatalf("expected updated config in response, got %s", body)
	}
}

func TestRequestIDConfigRejectsInvalidFormat(t *testing.T) {
	router := gin.New()
	generator := NewRequestIDGenerator("req", RequestIDFormatUnix, nil)
	generator.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/request-id", strings.NewReader(`{"format":"invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
