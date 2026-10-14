package gateway

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestDeprecationService_Middleware_LegacyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ds := NewDeprecationService()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/nodes", nil)

	ds.Middleware()(c)

	if c.Writer.Header().Get("Deprecation") == "" {
		t.Error("expected Deprecation header for legacy path")
	}
	if c.Writer.Header().Get("Link") == "" {
		t.Error("expected Link header for legacy path")
	}
}

func TestDeprecationService_Middleware_LegacyPathWithTrailingSlash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ds := NewDeprecationService()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/nodes/", nil)

	ds.Middleware()(c)

	if c.Writer.Header().Get("Deprecation") == "" {
		t.Error("expected Deprecation header for legacy path with trailing slash")
	}
}

func TestDeprecationService_Middleware_NonLegacyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ds := NewDeprecationService()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/nodes", nil)

	ds.Middleware()(c)

	if c.Writer.Header().Get("Deprecation") != "" {
		t.Error("expected no Deprecation header for non-legacy path")
	}
}

func TestDeprecationService_GetSunsetDate(t *testing.T) {
	ds := NewDeprecationService()
	sunset := ds.GetSunsetDate()
	if sunset.Before(time.Now()) {
		t.Error("sunset date should be in the future")
	}
}

func TestDeprecationService_AddLegacyPath(t *testing.T) {
	ds := NewDeprecationService()
	initialCount := len(ds.LegacyPaths)

	ds.AddLegacyPath("/admin/new-path")

	if len(ds.LegacyPaths) != initialCount+1 {
		t.Error("expected legacy path to be added")
	}
	if ds.LegacyPaths[len(ds.LegacyPaths)-1] != "/admin/new-path" {
		t.Error("expected new legacy path at end of list")
	}
}

func TestDeprecationService_Middleware_AddLegacyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ds := NewDeprecationService()

	ds.AddLegacyPath("/admin/custom-path")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/custom-path", nil)

	ds.Middleware()(c)

	if c.Writer.Header().Get("Deprecation") == "" {
		t.Error("expected Deprecation header for newly added legacy path")
	}
}
