package gateway

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSlowClientBuffer_New(t *testing.T) {
	scb := NewSlowClientBuffer(1024)
	if scb == nil {
		t.Fatal("NewSlowClientBuffer returned nil")
	}
	if scb.maxBodySize != 1024 {
		t.Errorf("expected maxBodySize 1024, got %d", scb.maxBodySize)
	}
}

func TestSlowClientBuffer_Middleware_NilBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	scb := NewSlowClientBuffer(1024)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	scb.Middleware()(c)
	c.Next()
}

func TestSlowClientBuffer_Middleware_UnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	scb := NewSlowClientBuffer(1024)

	body := io.NopCloser(bytes.NewReader([]byte("small body")))
	req := httptest.NewRequest("POST", "/test", body)
	req.ContentLength = 10

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	scb.Middleware()(c)
	c.Next()
}

func TestRequestBuffer_New(t *testing.T) {
	rb := NewRequestBuffer(1024)
	if rb == nil {
		t.Fatal("NewRequestBuffer returned nil")
	}
	if rb.maxBufferSize != 1024 {
		t.Errorf("expected maxBufferSize 1024, got %d", rb.maxBufferSize)
	}
}

func TestRequestBuffer_Stats(t *testing.T) {
	rb := NewRequestBuffer(1024)
	count, bytes := rb.Stats()
	if count != 0 {
		t.Errorf("expected requestCount 0, got %d", count)
	}
	if bytes != 0 {
		t.Errorf("expected bytesRead 0, got %d", bytes)
	}
}

func TestRequestBuffer_WrapHandler(t *testing.T) {
	rb := NewRequestBuffer(1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("handled"))
	})

	wrapped := rb.WrapHandler(handler)

	req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("test body")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	count, bytesRead := rb.Stats()
	if count != 1 {
		t.Errorf("expected requestCount 1, got %d", count)
	}
	if bytesRead < 9 {
		t.Errorf("expected bytesRead at least 9, got %d", bytesRead)
	}
}

func TestRequestBuffer_WrapHandler_NilBody(t *testing.T) {
	rb := NewRequestBuffer(1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	wrapped := rb.WrapHandler(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestSlowClientHandler_New(t *testing.T) {
	sch := NewSlowClientHandler(5*time.Second, 10*time.Second, 1024)
	if sch == nil {
		t.Fatal("NewSlowClientHandler returned nil")
	}
	if sch.readTimeout != 5*time.Second {
		t.Errorf("expected readTimeout 5s, got %v", sch.readTimeout)
	}
	if sch.writeTimeout != 10*time.Second {
		t.Errorf("expected writeTimeout 10s, got %v", sch.writeTimeout)
	}
	if sch.bufferSize != 1024 {
		t.Errorf("expected bufferSize 1024, got %d", sch.bufferSize)
	}
}

func TestSlowClientHandler_WrapHandler_NoTimeout(t *testing.T) {
	sch := NewSlowClientHandler(0, 0, 1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	wrapped := sch.WrapHandler(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestSlowClientHandler_WrapHandler_WithReadTimeout(t *testing.T) {
	sch := NewSlowClientHandler(5*time.Second, 0, 1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	wrapped := sch.WrapHandler(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestTimeoutWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	tw := &timeoutWriter{
		ResponseWriter: w,
		timeout:        5 * time.Second,
	}

	n, err := tw.Write([]byte("test data"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n != 9 {
		t.Errorf("expected 9 bytes written, got %d", n)
	}
}
