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
	if len(svc.GetTiers()) != 3 {
		t.Errorf("expected 3 tiers, got %d", len(svc.GetTiers()))
	}
}

func TestSetAndGetTiers(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	tiers := []config.PricingTier{
		{Name: "Test", MinGBMonthly: 0, MaxGBMonthly: 100, RatePerGBSent: 1.0, RatePerGBRecv: 0.5},
	}
	svc.SetTiers(tiers)
	got := svc.GetTiers()
	if len(got) != 1 {
		t.Errorf("expected 1 tier, got %d", len(got))
	}
	if got[0].Name != "Test" {
		t.Errorf("expected tier 'Test', got %s", got[0].Name)
	}
}

func TestSetAndGetRates(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	svc.SetRates(PayoutRates{
		RatePerGBSent:     1.0,
		RatePerGBReceived: 0.5,
		MinPayoutAmount:   5.0,
	})
	rates := svc.GetRates()
	if rates.RatePerGBSent != 1.0 {
		t.Errorf("expected 1.0, got %f", rates.RatePerGBSent)
	}
	if rates.RatePerGBReceived != 0.5 {
		t.Errorf("expected 0.5, got %f", rates.RatePerGBReceived)
	}
	if rates.MinPayoutAmount != 5.0 {
		t.Errorf("expected 5.0, got %f", rates.MinPayoutAmount)
	}
}

func TestFindTier(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	tests := []struct {
		totalGB  float64
		expected string
	}{
		{0, "Basic"},
		{5, "Basic"},
		{15, "Premium"},
		{50, "Premium"},
		{150, "Enterprise"},
		{5000, "Enterprise"},
	}
	for _, tt := range tests {
		got := svc.findTier(tt.totalGB)
		if got != tt.expected {
			t.Errorf("findTier(%f) = %s, want %s", tt.totalGB, got, tt.expected)
		}
	}
}

func TestGetTierRates(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	sent, recv := svc.getTierRates("Premium")
	if sent != 0.50 {
		t.Errorf("expected Premium sent rate 0.50, got %f", sent)
	}
	if recv != 0.30 {
		t.Errorf("expected Premium recv rate 0.30, got %f", recv)
	}

	sent, recv = svc.getTierRates("NonExistent")
	if sent != 0.50 {
		t.Errorf("expected default sent rate 0.50, got %f", sent)
	}
	if recv != 0.30 {
		t.Errorf("expected default recv rate 0.30, got %f", recv)
	}
}

func TestCalculatePayout_NilRedis(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	_, err := svc.CalculatePayout("node-1", time.Now())
	if err == nil {
		t.Error("expected error when redis is nil")
	}
}

func TestCalculateAllPayouts_NilRedis(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	_, err := svc.CalculateAllPayouts(time.Now())
	if err == nil {
		t.Error("expected error when redis is nil")
	}
}

func TestPayout_Struct(t *testing.T) {
	p := Payout{
		NodeID:        "node-1",
		Period:        "2024-01",
		BytesSent:     1000,
		BytesReceived: 500,
		GBSent:        0.001,
		GBReceived:    0.0005,
		Amount:        0.50,
		Tier:          "Basic",
		CalculatedAt:  time.Now(),
	}
	if p.NodeID != "node-1" {
		t.Error("unexpected NodeID")
	}
	if p.Amount != 0.50 {
		t.Error("unexpected Amount")
	}
}

func TestPayoutRates_Struct(t *testing.T) {
	r := PayoutRates{
		RatePerGBSent:     0.5,
		RatePerGBReceived: 0.3,
		MinPayoutAmount:   10.0,
	}
	if r.RatePerGBSent != 0.5 {
		t.Error("unexpected RatePerGBSent")
	}
}
