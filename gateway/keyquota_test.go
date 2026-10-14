package gateway

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestKeyExpirationManager_New(t *testing.T) {
	kem := NewKeyExpirationManager(nil)
	if kem == nil {
		t.Fatal("NewKeyExpirationManager returned nil")
	}
}

func TestKeyExpirationManager_CreateAndCheck(t *testing.T) {
	kem := NewKeyExpirationManager(nil)
	err := kem.CreateKey("key-1", "Test Key", 1, 30)
	if err != nil {
		t.Fatalf("CreateKey error: %v", err)
	}
	if kem.IsExpired("key-1") {
		t.Error("expected key not expired")
	}
	if kem.IsExpired("nonexistent") {
		t.Error("expected nonexistent key not expired")
	}
}

func TestKeyExpirationManager_RenewKey(t *testing.T) {
	kem := NewKeyExpirationManager(nil)
	kem.CreateKey("key-2", "Test", 1, 30)

	err := kem.RenewKey("key-2", 365)
	if err != nil {
		t.Fatalf("RenewKey error: %v", err)
	}

	expiry := kem.GetExpiry("key-2")
	if expiry.IsZero() {
		t.Error("expected non-zero expiry after renew")
	}
	expected := time.Now().AddDate(0, 0, 365)
	if expiry.Before(expected.Add(-1*time.Minute)) {
		t.Errorf("expected expiry around %v, got %v", expected, expiry)
	}
}

func TestKeyExpirationManager_RenewKey_NotFound(t *testing.T) {
	kem := NewKeyExpirationManager(nil)
	err := kem.RenewKey("missing", 30)
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestKeyExpirationManager_GetExpiry_NotFound(t *testing.T) {
	kem := NewKeyExpirationManager(nil)
	expiry := kem.GetExpiry("missing")
	if !expiry.IsZero() {
		t.Error("expected zero time for missing key")
	}
}

func TestKeyExpirationManager_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	kem := NewKeyExpirationManager(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	kem.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected request without auth to pass through")
	}
}

func TestBandwidthQuota_New(t *testing.T) {
	bq := NewBandwidthQuota(nil)
	if bq == nil {
		t.Fatal("NewBandwidthQuota returned nil")
	}
}

func TestBandwidthQuota_SetAndCheck(t *testing.T) {
	bq := NewBandwidthQuota(nil)
	err := bq.SetQuota("node-1", 100, true)
	if err != nil {
		t.Fatalf("SetQuota error: %v", err)
	}

	allowed, err := bq.CheckQuota("node-1", 10)
	if err != nil {
		t.Fatalf("CheckQuota error: %v", err)
	}
	if !allowed {
		t.Error("expected quota check allowed")
	}

	allowed, err = bq.CheckQuota("node-1", 200)
	if err == nil || allowed {
		t.Error("expected hard limit quota exceeded")
	}
}

func TestBandwidthQuota_CheckQuota_NoQuota(t *testing.T) {
	bq := NewBandwidthQuota(nil)
	allowed, err := bq.CheckQuota("unknown-node", 1000)
	if err != nil {
		t.Fatalf("CheckQuota error: %v", err)
	}
	if !allowed {
		t.Error("expected allowed when no quota set")
	}
}

func TestBandwidthQuota_RecordUsage(t *testing.T) {
	bq := NewBandwidthQuota(nil)
	bq.SetQuota("node-2", 50, true)
	err := bq.RecordUsage("node-2", 10)
	if err != nil {
		t.Fatalf("RecordUsage error: %v", err)
	}
	remaining := bq.GetRemainingQuota("node-2")
	if remaining != 40 {
		t.Errorf("expected remaining 40, got %d", remaining)
	}
}

func TestBandwidthQuota_GetRemainingQuota_NoQuota(t *testing.T) {
	bq := NewBandwidthQuota(nil)
	remaining := bq.GetRemainingQuota("unknown")
	if remaining != -1 {
		t.Errorf("expected -1 for unknown node, got %d", remaining)
	}
}

func TestBandwidthQuota_GetQuotaInfo(t *testing.T) {
	bq := NewBandwidthQuota(nil)
	bq.SetQuota("node-3", 200, false)
	info := bq.GetQuotaInfo("node-3")
	if info == nil {
		t.Fatal("expected non-nil quota info")
	}
	if info.NodeID != "node-3" {
		t.Errorf("expected node-3, got %s", info.NodeID)
	}
	if info.MonthlyGB != 200 {
		t.Errorf("expected 200GB, got %d", info.MonthlyGB)
	}
}

func TestBandwidthQuota_GetQuotaInfo_NotFound(t *testing.T) {
	bq := NewBandwidthQuota(nil)
	info := bq.GetQuotaInfo("unknown")
	if info != nil {
		t.Error("expected nil for unknown node")
	}
}

func TestBandwidthQuota_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bq := NewBandwidthQuota(nil)
	router := gin.New()
	bq.RegisterRoutes(router)
}

func TestBandwidthQuota_SetQuotaHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bq := NewBandwidthQuota(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/quotas", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	bq.setQuotaHandler(c)
}

func TestGetNextResetTime(t *testing.T) {
	reset := getNextResetTime()
	now := time.Now()
	if reset.Before(now) {
		t.Error("expected reset time in the future")
	}
	if reset.Day() != 1 {
		t.Error("expected reset on first day of month")
	}
	if reset.Hour() != 0 || reset.Minute() != 0 {
		t.Error("expected reset at midnight")
	}
}

func TestBandwidthQuota_RecordUsage_NoQuota(t *testing.T) {
	bq := NewBandwidthQuota(nil)
	err := bq.RecordUsage("unknown", 10)
	if err != nil {
		t.Errorf("RecordUsage error: %v", err)
	}
}
