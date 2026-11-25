package payout

import (
	"testing"
	"time"

	"proxymesh/internal/config"
	"proxymesh/internal/models"
)

func TestCalculatePayout_WithMockRedis(t *testing.T) {
	mockRedis := newMockRedisClient()
	// 20 GB sent + 10 GB received = 30 GB total -> Premium tier (10-100 GB)
	// But wait, tier is based on total GB = 30, which is Premium tier
	// Premium rates: 0.50 sent, 0.30 recv
	// Amount = 20*0.50 + 10*0.30 = 10 + 3 = 13 > 10 minimum
	mockRedis.SetBandwidth("node-1", models.BandwidthData{
		BytesSent:     21474836480,  // 20 GB
		BytesReceived: 10737418240,  // 10 GB
	})

	svc := NewPayoutService(nil, mockRedis)
	payout, err := svc.CalculatePayout("node-1", time.Now())
	if err != nil {
		t.Fatalf("CalculatePayout error: %v", err)
	}

	if payout == nil {
		t.Fatal("expected payout, got nil")
	}

	expectedGBSent := 20.0
	expectedGBReceived := 10.0
	expectedAmount := expectedGBSent*0.50 + expectedGBReceived*0.30 // Premium tier rates

	if payout.GBSent != expectedGBSent {
		t.Errorf("expected GBSent %f, got %f", expectedGBSent, payout.GBSent)
	}
	if payout.GBReceived != expectedGBReceived {
		t.Errorf("expected GBReceived %f, got %f", expectedGBReceived, payout.GBReceived)
	}
	if payout.Amount != expectedAmount {
		t.Errorf("expected Amount %f, got %f", expectedAmount, payout.Amount)
	}
	if payout.Tier != "Premium" {
		t.Errorf("expected Tier Premium, got %s", payout.Tier)
	}
	if payout.NodeID != "node-1" {
		t.Errorf("expected NodeID node-1, got %s", payout.NodeID)
	}
}

func TestCalculatePayout_PremiumTier(t *testing.T) {
	mockRedis := newMockRedisClient()
	// 50 GB sent + 20 GB received = 70 GB total -> Premium tier (10-100 GB)
	mockRedis.SetBandwidth("node-1", models.BandwidthData{
		BytesSent:     53687091200,  // 50 GB
		BytesReceived: 21474836480,  // 20 GB
	})

	svc := NewPayoutService(nil, mockRedis)
	payout, err := svc.CalculatePayout("node-1", time.Now())
	if err != nil {
		t.Fatalf("CalculatePayout error: %v", err)
	}

	if payout == nil {
		t.Fatal("expected payout, got nil")
	}

	expectedGBSent := 50.0
	expectedGBReceived := 20.0
	expectedAmount := expectedGBSent*0.50 + expectedGBReceived*0.30 // Premium tier rates

	if payout.GBSent != expectedGBSent {
		t.Errorf("expected GBSent %f, got %f", expectedGBSent, payout.GBSent)
	}
	if payout.GBReceived != expectedGBReceived {
		t.Errorf("expected GBReceived %f, got %f", expectedGBReceived, payout.GBReceived)
	}
	if payout.Amount != expectedAmount {
		t.Errorf("expected Amount %f, got %f", expectedAmount, payout.Amount)
	}
	if payout.Tier != "Premium" {
		t.Errorf("expected Tier Premium, got %s", payout.Tier)
	}
}

func TestCalculatePayout_EnterpriseTier(t *testing.T) {
	mockRedis := newMockRedisClient()
	mockRedis.SetBandwidth("node-1", models.BandwidthData{
		BytesSent:     161061273600,  // 150 GB
		BytesReceived: 53687091200,   // 50 GB
	})

	svc := NewPayoutService(nil, mockRedis)
	payout, err := svc.CalculatePayout("node-1", time.Now())
	if err != nil {
		t.Fatalf("CalculatePayout error: %v", err)
	}

	if payout == nil {
		t.Fatal("expected payout, got nil")
	}

	expectedGBSent := 150.0
	expectedGBReceived := 50.0
	expectedAmount := expectedGBSent*0.80 + expectedGBReceived*0.50 // Enterprise tier rates

	if payout.GBSent != expectedGBSent {
		t.Errorf("expected GBSent %f, got %f", expectedGBSent, payout.GBSent)
	}
	if payout.GBReceived != expectedGBReceived {
		t.Errorf("expected GBReceived %f, got %f", expectedGBReceived, payout.GBReceived)
	}
	if payout.Amount != expectedAmount {
		t.Errorf("expected Amount %f, got %f", expectedAmount, payout.Amount)
	}
	if payout.Tier != "Enterprise" {
		t.Errorf("expected Tier Enterprise, got %s", payout.Tier)
	}
}

func TestCalculatePayout_BelowMinimum(t *testing.T) {
	mockRedis := newMockRedisClient()
	mockRedis.SetBandwidth("node-1", models.BandwidthData{
		BytesSent:     107374182,  // 0.1 GB
		BytesReceived: 53687091,   // 0.05 GB
	})

	svc := NewPayoutService(nil, mockRedis)
	payout, err := svc.CalculatePayout("node-1", time.Now())
	if err != nil {
		t.Fatalf("CalculatePayout error: %v", err)
	}

	if payout == nil {
		t.Fatal("expected payout, got nil")
	}

	// Amount should be 0 because it's below minimum
	if payout.Amount != 0 {
		t.Errorf("expected Amount 0 (below minimum), got %f", payout.Amount)
	}
}

func TestCalculatePayout_CustomTiers(t *testing.T) {
	mockRedis := newMockRedisClient()
	mockRedis.SetBandwidth("node-1", models.BandwidthData{
		BytesSent:     53687091200,  // 50 GB
		BytesReceived: 21474836480,  // 20 GB
	})

	svc := NewPayoutService(nil, mockRedis)
	svc.SetTiers([]config.PricingTier{
		{Name: "Starter", MinGBMonthly: 0, MaxGBMonthly: 100, RatePerGBSent: 0.25, RatePerGBRecv: 0.10},
		{Name: "Pro", MinGBMonthly: 100, MaxGBMonthly: 1000, RatePerGBSent: 0.60, RatePerGBRecv: 0.30},
	})

	payout, err := svc.CalculatePayout("node-1", time.Now())
	if err != nil {
		t.Fatalf("CalculatePayout error: %v", err)
	}

	if payout == nil {
		t.Fatal("expected payout, got nil")
	}

	expectedGBSent := 50.0
	expectedGBReceived := 20.0
	expectedAmount := expectedGBSent*0.25 + expectedGBReceived*0.10 // Starter tier

	if payout.GBSent != expectedGBSent {
		t.Errorf("expected GBSent %f, got %f", expectedGBSent, payout.GBSent)
	}
	if payout.GBReceived != expectedGBReceived {
		t.Errorf("expected GBReceived %f, got %f", expectedGBReceived, payout.GBReceived)
	}
	if payout.Amount != expectedAmount {
		t.Errorf("expected Amount %f, got %f", expectedAmount, payout.Amount)
	}
	if payout.Tier != "Starter" {
		t.Errorf("expected Tier Starter, got %s", payout.Tier)
	}
}

func TestCalculatePayout_RecordsPayout(t *testing.T) {
	mockRedis := newMockRedisClient()
	// 20 GB sent + 10 GB received = 30 GB -> Premium tier, above minimum
	mockRedis.SetBandwidth("node-1", models.BandwidthData{
		BytesSent:     21474836480,
		BytesReceived: 10737418240,
	})

	svc := NewPayoutService(nil, mockRedis)
	payout, err := svc.CalculatePayout("node-1", time.Now())
	if err != nil {
		t.Fatalf("CalculatePayout error: %v", err)
	}

	if payout == nil {
		t.Fatal("expected payout, got nil")
	}

	// Verify payout was recorded (amount > 0)
	if len(mockRedis.payouts) != 1 {
		t.Errorf("expected 1 recorded payout, got %d", len(mockRedis.payouts))
	}
	if mockRedis.payouts[0]["node_id"] != "node-1" {
		t.Errorf("expected recorded node_id node-1, got %v", mockRedis.payouts[0]["node_id"])
	}
}

func TestCalculateAllPayouts_WithMockRedis(t *testing.T) {
	mockRedis := newMockRedisClient()
	// All above minimum payout (10 USD)
	mockRedis.SetBandwidth("node-1", models.BandwidthData{BytesSent: 53687091200, BytesReceived: 21474836480})  // 70 GB -> Premium
	mockRedis.SetBandwidth("node-2", models.BandwidthData{BytesSent: 53687091200, BytesReceived: 21474836480})  // 70 GB -> Premium
	mockRedis.SetBandwidth("node-3", models.BandwidthData{BytesSent: 53687091200, BytesReceived: 21474836480})  // 70 GB -> Premium

	svc := NewPayoutService(nil, mockRedis)
	payouts, err := svc.CalculateAllPayouts(time.Now())
	if err != nil {
		t.Fatalf("CalculateAllPayouts error: %v", err)
	}

	if len(payouts) != 3 {
		t.Errorf("expected 3 payouts, got %d", len(payouts))
	}

	for _, p := range payouts {
		if p.Amount <= 0 {
			t.Errorf("expected positive amount for node %s, got %f", p.NodeID, p.Amount)
		}
		if p.NodeID == "" {
			t.Error("expected NodeID to be set")
		}
	}
}

func TestCalculateAllPayouts_EmptyNodes(t *testing.T) {
	mockRedis := newMockRedisClient()
	mockRedis.SetNodes([]string{})

	svc := NewPayoutService(nil, mockRedis)
	payouts, err := svc.CalculateAllPayouts(time.Now())
	if err != nil {
		t.Fatalf("CalculateAllPayouts error: %v", err)
	}

	if len(payouts) != 0 {
		t.Errorf("expected 0 payouts, got %d", len(payouts))
	}
}

func TestCalculateAllPayouts_SkipsZeroAmount(t *testing.T) {
	mockRedis := newMockRedisClient()
	// node-1: below minimum (0.1 GB sent + 0.05 GB received)
	mockRedis.SetBandwidth("node-1", models.BandwidthData{BytesSent: 107374182, BytesReceived: 53687091})
	// node-2: above minimum (50 GB sent + 20 GB received = 70 GB -> Premium)
	mockRedis.SetBandwidth("node-2", models.BandwidthData{BytesSent: 53687091200, BytesReceived: 21474836480})

	svc := NewPayoutService(nil, mockRedis)
	payouts, err := svc.CalculateAllPayouts(time.Now())
	if err != nil {
		t.Fatalf("CalculateAllPayouts error: %v", err)
	}

	// Should only return node-2 (node-1 is below minimum)
	if len(payouts) != 1 {
		t.Errorf("expected 1 payout (node-2), got %d", len(payouts))
	}
	if len(payouts) > 0 && payouts[0].NodeID != "node-2" {
		t.Errorf("expected node-2, got %s", payouts[0].NodeID)
	}
}

func TestGetPayoutHistory_WithMockRedis(t *testing.T) {
	mockRedis := newMockRedisClient()

	svc := NewPayoutService(nil, mockRedis)
	history, err := svc.GetPayoutHistory("node-1", 10)
	if err != nil {
		t.Fatalf("GetPayoutHistory error: %v", err)
	}

	// Should return empty history initially
	if history == nil {
		t.Fatal("expected history slice, got nil")
	}
}

func TestGetPayoutHistory_ReturnsRecorded(t *testing.T) {
	mockRedis := newMockRedisClient()
	mockRedis.SetBandwidth("node-1", models.BandwidthData{BytesSent: 21474836480, BytesReceived: 10737418240})

	svc := NewPayoutService(nil, mockRedis)
	_, _ = svc.CalculatePayout("node-1", time.Now())

	history, err := svc.GetPayoutHistory("node-1", 10)
	if err != nil {
		t.Fatalf("GetPayoutHistory error: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(history))
	}
}

func TestGetBandwidthBreakdown_WithMockRedis(t *testing.T) {
	mockRedis := newMockRedisClient()
	mockRedis.SetBandwidthHistory("node-1", map[string]models.BandwidthData{
		"2024-01": {BytesSent: 1073741824, BytesReceived: 536870912},
		"2024-02": {BytesSent: 2147483648, BytesReceived: 1073741824},
	})

	svc := NewPayoutService(nil, mockRedis)
	breakdown, err := svc.GetBandwidthBreakdown("node-1")
	if err != nil {
		t.Fatalf("GetBandwidthBreakdown error: %v", err)
	}

	if breakdown == nil {
		t.Fatal("expected breakdown map, got nil")
	}

	if len(breakdown) != 2 {
		t.Errorf("expected 2 months of data, got %d", len(breakdown))
	}

	if breakdown["2024-01"].BytesSent != 1073741824 {
		t.Errorf("expected 2024-01 BytesSent 1073741824, got %d", breakdown["2024-01"].BytesSent)
	}
	if breakdown["2024-02"].BytesSent != 2147483648 {
		t.Errorf("expected 2024-02 BytesSent 2147483648, got %d", breakdown["2024-02"].BytesSent)
	}
}

func TestGetBandwidthBreakdown_Empty(t *testing.T) {
	mockRedis := newMockRedisClient()

	svc := NewPayoutService(nil, mockRedis)
	breakdown, err := svc.GetBandwidthBreakdown("node-1")
	if err != nil {
		t.Fatalf("GetBandwidthBreakdown error: %v", err)
	}

	if breakdown == nil {
		t.Fatal("expected breakdown map, got nil")
	}
	if len(breakdown) != 0 {
		t.Errorf("expected empty breakdown, got %d entries", len(breakdown))
	}
}

func TestCalculatePayoutInternal_NilRedis(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	_, err := svc.calculatePayoutInternal("node-1", time.Now())
	if err == nil {
		t.Error("expected error when redis is nil")
	}
}

func TestCalculatePayoutInternal_WithMockRedis(t *testing.T) {
	mockRedis := newMockRedisClient()
	mockRedis.SetBandwidth("node-1", models.BandwidthData{
		BytesSent:     1073741824,
		BytesReceived: 536870912,
	})

	svc := NewPayoutService(nil, mockRedis)
	payout, err := svc.calculatePayoutInternal("node-1", time.Now())
	if err != nil {
		t.Fatalf("calculatePayoutInternal error: %v", err)
	}

	if payout == nil {
		t.Fatal("expected payout, got nil")
	}

	expectedGBSent := 1.0
	expectedGBReceived := 0.5

	if payout.GBSent != expectedGBSent {
		t.Errorf("expected GBSent %f, got %f", expectedGBSent, payout.GBSent)
	}
	if payout.GBReceived != expectedGBReceived {
		t.Errorf("expected GBReceived %f, got %f", expectedGBReceived, payout.GBReceived)
	}
	if payout.NodeID != "node-1" {
		t.Errorf("expected NodeID node-1, got %s", payout.NodeID)
	}
}

func TestCalculatePayoutInternal_RecordsPayout(t *testing.T) {
	mockRedis := newMockRedisClient()
	mockRedis.SetBandwidth("node-1", models.BandwidthData{
		BytesSent:     1073741824,
		BytesReceived: 536870912,
	})

	svc := NewPayoutService(nil, mockRedis)
	payout, err := svc.calculatePayoutInternal("node-1", time.Now())
	if err != nil {
		t.Fatalf("calculatePayoutInternal error: %v", err)
	}

	if payout == nil {
		t.Fatal("expected payout, got nil")
	}

	// calculatePayoutInternal does NOT record the payout - only CalculatePayout does
	// This is expected behavior
	if len(mockRedis.payouts) != 0 {
		t.Errorf("expected 0 recorded payouts from calculatePayoutInternal, got %d", len(mockRedis.payouts))
	}
}

func TestFindTier_ExactBoundaries(t *testing.T) {
	svc := NewPayoutService(nil, nil)

	// Test exact boundary values
	tests := []struct {
		totalGB  float64
		expected string
	}{
		{0, "Basic"},
		{9.999, "Basic"},
		{10, "Premium"},
		{99.999, "Premium"},
		{100, "Enterprise"},
		{9999, "Enterprise"},
		{10001, "Basic"}, // Beyond Enterprise max
	}

	for _, tt := range tests {
		got := svc.findTier(tt.totalGB)
		if got != tt.expected {
			t.Errorf("findTier(%f) = %s, want %s", tt.totalGB, got, tt.expected)
		}
	}
}

func TestFindTier_NoTiersConfigured(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	svc.SetTiers([]config.PricingTier{}) // Empty tiers

	result := svc.findTier(50.0)
	if result != "Basic" {
		t.Errorf("expected Basic fallback when no tiers, got %s", result)
	}
}

func TestGetTierRates_CaseSensitive(t *testing.T) {
	svc := NewPayoutService(nil, nil)

	sent, recv := svc.getTierRates("basic") // lowercase
	if sent != 0.50 || recv != 0.30 {
		t.Errorf("expected default rates for lowercase 'basic', got %f, %f", sent, recv)
	}

	sent, recv = svc.getTierRates("BASIC") // uppercase
	if sent != 0.50 || recv != 0.30 {
		t.Errorf("expected default rates for uppercase 'BASIC', got %f, %f", sent, recv)
	}

	sent, recv = svc.getTierRates("Basic") // correct case
	if sent != 0.30 || recv != 0.15 {
		t.Errorf("expected Basic rates, got %f, %f", sent, recv)
	}
}

func TestPayoutService_ZeroValues(t *testing.T) {
	mockRedis := newMockRedisClient()
	mockRedis.SetBandwidth("node-1", models.BandwidthData{BytesSent: 0, BytesReceived: 0})

	svc := NewPayoutService(nil, mockRedis)
	payout, err := svc.CalculatePayout("node-1", time.Now())
	if err != nil {
		t.Fatalf("CalculatePayout error: %v", err)
	}

	if payout.GBSent != 0 || payout.GBReceived != 0 {
		t.Errorf("expected 0 GB for zero bytes, got %f, %f", payout.GBSent, payout.GBReceived)
	}
	if payout.Amount != 0 {
		t.Errorf("expected 0 amount for zero bytes, got %f", payout.Amount)
	}
}

func TestPayoutService_NegativeBytes(t *testing.T) {
	mockRedis := newMockRedisClient()
	mockRedis.SetBandwidth("node-1", models.BandwidthData{BytesSent: -1000, BytesReceived: -500})

	svc := NewPayoutService(nil, mockRedis)
	payout, err := svc.CalculatePayout("node-1", time.Now())
	if err != nil {
		t.Fatalf("CalculatePayout error: %v", err)
	}

	// Should handle negative bytes gracefully (result in negative GB, then 0 amount due to min)
	if payout.GBSent >= 0 {
		t.Errorf("expected negative GBSent for negative bytes, got %f", payout.GBSent)
	}
}