package gateway

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResponseCompressor_Gzip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rc := NewResponseCompressor(true, 5)
	r := gin.New()
	r.Use(rc.Middleware())
	r.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello compressed world"})
	})

	req := httptest.NewRequest("GET", "/data", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected Content-Encoding: gzip, got %q", w.Header().Get("Content-Encoding"))
	}

	gz, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("response is not valid gzip: %v", err)
	}
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}
	if string(body) == "" {
		t.Fatal("decompressed body is empty")
	}
}

func TestResponseCompressor_SkipsWhenNoAcceptEncoding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rc := NewResponseCompressor(true, 5)
	r := gin.New()
	r.Use(rc.Middleware())
	r.GET("/data", func(c *gin.Context) {
		c.String(http.StatusOK, "plain text")
	})

	req := httptest.NewRequest("GET", "/data", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "" {
		t.Fatalf("expected no Content-Encoding, got %q", w.Header().Get("Content-Encoding"))
	}
	if w.Body.String() != "plain text" {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestResponseCompressor_SkipsStreamPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rc := NewResponseCompressor(true, 5)
	r := gin.New()
	r.Use(rc.Middleware())
	r.GET("/api/peer/stream", func(c *gin.Context) {
		c.String(http.StatusOK, "event: telemetry\ndata: {}")
	})

	req := httptest.NewRequest("GET", "/api/peer/stream", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "" {
		t.Fatalf("stream path should not be compressed, got %q", w.Header().Get("Content-Encoding"))
	}
}
