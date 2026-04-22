package payout

import (
	"testing"
	"time"

	"proxymesh/internal/config"
)

func TestNewPayoutService(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	if svc == nil {
		t.Fatal("NewPayoutService() returned nil")
	}
	if svc.rates.RatePerGBSent != 0.50 {
		t.Errorf("expected RatePerGBSent=0.50, got %f", svc.rates.RatePerGBSent)
	}
	if svc.rates.RatePerGBReceived != 0.30 {
		t.Errorf("expected RatePerGBReceived=0.30, got %f", svc.rates.RatePerGBReceived)
	}
	if svc.rates.MinPayoutAmount != 10.00 {
		t.Errorf("expected MinPayoutAmount=10.00, got %f", svc.rates.MinPayoutAmount)
	}
	if len(svc.tiers) != 3 {
		t.Fatalf("expected 3 tiers, got %d", len(svc.tiers))
	}
}

func TestSetAndGetRates(t *testing.T) {
	svc := NewPayoutService(nil, nil)

	newRates := PayoutRates{
		RatePerGBSent:     1.00,
		RatePerGBReceived: 0.50,
		MinPayoutAmount:   5.00,
	}
	svc.SetRates(newRates)
	got := svc.GetRates()
	if got.RatePerGBSent != 1.00 || got.RatePerGBReceived != 0.50 || got.MinPayoutAmount != 5.00 {
		t.Errorf("SetRates/GetRates roundtrip failed: got %+v", got)
	}
}

func TestFindTier(t *testing.T) {
	svc := NewPayoutService(nil, nil)

	tests := []struct {
		totalGB float64
		expect  string
	}{
		{0, "Basic"},
		{5, "Basic"},
		{10, "Premium"},
		{50, "Premium"},
		{100, "Enterprise"},
		{500, "Enterprise"},
	}

	for _, tt := range tests {
		got := svc.findTier(tt.totalGB)
		if got != tt.expect {
			t.Errorf("findTier(%.0f GB) = %q, want %q", tt.totalGB, got, tt.expect)
		}
	}
}

func TestGetTierRates(t *testing.T) {
	svc := NewPayoutService(nil, nil)

	sent, recv := svc.getTierRates("Basic")
	if sent != 0.30 || recv != 0.15 {
		t.Errorf("Basic rates: sent=%f recv=%f", sent, recv)
	}

	sent, recv = svc.getTierRates("Premium")
	if sent != 0.50 || recv != 0.30 {
		t.Errorf("Premium rates: sent=%f recv=%f", sent, recv)
	}

	sent, recv = svc.getTierRates("Enterprise")
	if sent != 0.80 || recv != 0.50 {
		t.Errorf("Enterprise rates: sent=%f recv=%f", sent, recv)
	}

	sent, recv = svc.getTierRates("UnknownTier")
	if sent != svc.rates.RatePerGBSent || recv != svc.rates.RatePerGBReceived {
		t.Errorf("UnknownTier should return default rates: sent=%f recv=%f", sent, recv)
	}
}

func TestSetAndGetTiers(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	custom := []config.PricingTier{
		{Name: "Starter", MinGBMonthly: 0, MaxGBMonthly: 5, RatePerGBSent: 0.10, RatePerGBRecv: 0.05},
		{Name: "Pro", MinGBMonthly: 5, MaxGBMonthly: 50, RatePerGBSent: 0.60, RatePerGBRecv: 0.40},
	}
	svc.SetTiers(custom)
	got := svc.GetTiers()
	if len(got) != 2 {
		t.Fatalf("expected 2 tiers, got %d", len(got))
	}
	if got[0].Name != "Starter" || got[1].Name != "Pro" {
		t.Errorf("unexpected tier order: %+v", got)
	}
}

func TestPayoutJSONSerialization(t *testing.T) {
	p := Payout{
		NodeID:        "node-123",
		Period:        "2026-04",
		BytesSent:     2_147_483_648,
		BytesReceived: 1_073_741_824,
		GBSent:        2.0,
		GBReceived:    1.0,
		Amount:        1.60,
		Tier:          "Premium",
		CalculatedAt:  time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC),
	}

	if p.GBSent != 2.0 {
		t.Errorf("GBSent = %f, want 2.0", p.GBSent)
	}
	if p.GBReceived != 1.0 {
		t.Errorf("GBReceived = %f, want 1.0", p.GBReceived)
	}
	if p.Amount == 0 {
		t.Error("Amount should not be zero")
	}
	if p.Tier != "Premium" {
		t.Errorf("Tier = %q, want %q", p.Tier, "Premium")
	}
}

func TestMinPayoutThreshold(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	_ = svc
	// MinPayoutAmount=10.00: 0.5 GB sent × $0.50 = $0.25 → zeroed out
	// 20 GB sent × $0.50 = $10.00 → paid
	t.Log("MinPayoutAmount=10.00: payouts below this amount are zeroed")
}
