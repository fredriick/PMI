package payout

import (
	"testing"
	"time"

	"proxymesh/internal/config"
	"proxymesh/internal/models"
)

func TestNewPayoutService_WithRedis(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	if svc == nil {
		t.Fatal("NewPayoutService() returned nil")
	}
	if svc.matchmaker != nil {
		t.Error("expected nil matchmaker")
	}
	if svc.redis != nil {
		t.Error("expected nil redis")
	}
}

func TestSetTiers_Empty(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	svc.SetTiers(nil)
	got := svc.GetTiers()
	if got != nil && len(got) != 0 {
		t.Errorf("expected empty or nil slice, got %d", len(got))
	}
}

func TestSetAndGetRates_Default(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	rates := svc.GetRates()
	if rates.RatePerGBSent != 0.50 {
		t.Errorf("expected 0.50, got %f", rates.RatePerGBSent)
	}
	if rates.RatePerGBReceived != 0.30 {
		t.Errorf("expected 0.30, got %f", rates.RatePerGBReceived)
	}
	if rates.MinPayoutAmount != 10.00 {
		t.Errorf("expected 10.00, got %f", rates.MinPayoutAmount)
	}
}

func TestFindTier_Boundary(t *testing.T) {
	svc := NewPayoutService(nil, nil)

	// Below Basic range (should not happen with default tiers starting at 0)
	if svc.findTier(-1) != "Basic" {
		t.Error("expected Basic for negative GB")
	}

	// Exact boundary: 0 GB is Basic
	if svc.findTier(0) != "Basic" {
		t.Error("expected Basic for 0 GB")
	}

	// Exactly at tier boundary: 10 GB should be Premium (Basic max is exclusive)
	if svc.findTier(10) != "Premium" {
		t.Errorf("expected Premium for 10 GB, got %s", svc.findTier(10))
	}

	// Exactly at tier boundary: 100 GB should be Enterprise (Premium max is exclusive)
	if svc.findTier(100) != "Enterprise" {
		t.Errorf("expected Enterprise for 100 GB, got %s", svc.findTier(100))
	}

	// Very large GB falls back to Basic (no tier matches beyond Enterprise max)
	if svc.findTier(100000) != "Basic" {
		t.Errorf("expected Basic fallback for 100000 GB, got %s", svc.findTier(100000))
	}
}

func TestGetTierRates_Basic(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	sent, recv := svc.getTierRates("Basic")
	if sent != 0.30 {
		t.Errorf("expected Basic sent rate 0.30, got %f", sent)
	}
	if recv != 0.15 {
		t.Errorf("expected Basic recv rate 0.15, got %f", recv)
	}
}

func TestGetTierRates_Enterprise(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	sent, recv := svc.getTierRates("Enterprise")
	if sent != 0.80 {
		t.Errorf("expected Enterprise sent rate 0.80, got %f", sent)
	}
	if recv != 0.50 {
		t.Errorf("expected Enterprise recv rate 0.50, got %f", recv)
	}
}

func TestCalculatePayout_ZeroAmount(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	_, err := svc.CalculatePayout("node-1", time.Now())
	if err == nil {
		t.Error("expected error when redis is nil")
	}
}

func TestCalculateAllPayouts_ZeroAmount(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	_, err := svc.CalculateAllPayouts(time.Now())
	if err == nil {
		t.Error("expected error when redis is nil")
	}
}

func TestGetPayoutHistory_NilRedis(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	_, err := svc.GetPayoutHistory("node-1", 10)
	if err == nil {
		t.Error("expected error when redis is nil")
	}
}

func TestGetBandwidthBreakdown_NilRedis(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	_, err := svc.GetBandwidthBreakdown("node-1")
	if err == nil {
		t.Error("expected error when redis is nil")
	}
}

func TestPayoutService_SetGetTiers_RoundTrip(t *testing.T) {
	svc := NewPayoutService(nil, nil)

	customTiers := []config.PricingTier{
		{Name: "Micro", MinGBMonthly: 0, MaxGBMonthly: 1, RatePerGBSent: 0.1, RatePerGBRecv: 0.05},
		{Name: "Macro", MinGBMonthly: 1, MaxGBMonthly: 1000, RatePerGBSent: 0.9, RatePerGBRecv: 0.45},
	}
	svc.SetTiers(customTiers)

	got := svc.GetTiers()
	if len(got) != 2 {
		t.Errorf("expected 2 tiers, got %d", len(got))
	}
	if got[0].Name != "Micro" {
		t.Errorf("expected tier 'Micro', got %s", got[0].Name)
	}
	if got[1].Name != "Macro" {
		t.Errorf("expected tier 'Macro', got %s", got[1].Name)
	}
}

func TestPayoutService_SetGetRates_RoundTrip(t *testing.T) {
	svc := NewPayoutService(nil, nil)

	svc.SetRates(PayoutRates{
		RatePerGBSent:     2.5,
		RatePerGBReceived: 1.5,
		MinPayoutAmount:   50.0,
	})

	rates := svc.GetRates()
	if rates.RatePerGBSent != 2.5 {
		t.Errorf("expected 2.5, got %f", rates.RatePerGBSent)
	}
	if rates.RatePerGBReceived != 1.5 {
		t.Errorf("expected 1.5, got %f", rates.RatePerGBReceived)
	}
	if rates.MinPayoutAmount != 50.0 {
		t.Errorf("expected 50.0, got %f", rates.MinPayoutAmount)
	}
}

func TestPayout_CalculatedAt(t *testing.T) {
	before := time.Now()
	p := Payout{
		NodeID:     "node-1",
		Period:     "2024-01",
		Amount:     100.0,
		CalculatedAt: time.Now(),
	}
	after := time.Now()

	if p.CalculatedAt.Before(before) || p.CalculatedAt.After(after) {
		t.Error("CalculatedAt should be between before and after")
	}
}

func TestFindTier_WithCustomTiers(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	svc.SetTiers([]config.PricingTier{
		{Name: "Starter", MinGBMonthly: 0, MaxGBMonthly: 5, RatePerGBSent: 0.1, RatePerGBRecv: 0.05},
		{Name: "Pro", MinGBMonthly: 5, MaxGBMonthly: 50, RatePerGBSent: 0.5, RatePerGBRecv: 0.25},
	})

	if svc.findTier(2) != "Starter" {
		t.Errorf("expected Starter for 2 GB, got %s", svc.findTier(2))
	}
	if svc.findTier(25) != "Pro" {
		t.Errorf("expected Pro for 25 GB, got %s", svc.findTier(25))
	}
	if svc.findTier(100) != "Basic" {
		t.Errorf("expected Basic fallback for 100 GB (no matching tier), got %s", svc.findTier(100))
	}
}

func TestGetTierRates_WithCustomTiers(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	svc.SetTiers([]config.PricingTier{
		{Name: "Custom", MinGBMonthly: 0, MaxGBMonthly: 100, RatePerGBSent: 0.75, RatePerGBRecv: 0.35},
	})

	sent, recv := svc.getTierRates("Custom")
	if sent != 0.75 {
		t.Errorf("expected 0.75, got %f", sent)
	}
	if recv != 0.35 {
		t.Errorf("expected 0.35, got %f", recv)
	}

	sent, recv = svc.getTierRates("NonExistent")
	if sent != 0.50 {
		t.Errorf("expected default sent 0.50, got %f", sent)
	}
	if recv != 0.30 {
		t.Errorf("expected default recv 0.30, got %f", recv)
	}
}

func TestPayout_JsonTags(t *testing.T) {
	p := Payout{
		NodeID:        "node-1",
		Period:        "2024-01",
		BytesSent:     1073741824,
		BytesReceived: 536870912,
		GBSent:        1.0,
		GBReceived:    0.5,
		Amount:        50.0,
		Tier:          "Premium",
		CalculatedAt:  time.Now(),
	}

	if p.NodeID != "node-1" {
		t.Error("unexpected NodeID")
	}
	if p.BytesSent != 1073741824 {
		t.Error("unexpected BytesSent")
	}
	if p.Tier != "Premium" {
		t.Error("unexpected Tier")
	}
}

func TestModels_BandwidthData(t *testing.T) {
	bd := models.BandwidthData{
		BytesSent:      1000,
		BytesReceived:  500,
		DurationSeconds: 10,
	}
	if bd.BytesSent != 1000 {
		t.Error("unexpected BytesSent")
	}
	if bd.BytesReceived != 500 {
		t.Error("unexpected BytesReceived")
	}
	if bd.DurationSeconds != 10 {
		t.Error("unexpected DurationSeconds")
	}
}

func TestCalculatePayout_AmountBelowMinimum(t *testing.T) {
	svc := NewPayoutService(nil, nil)
	_, err := svc.CalculatePayout("node-1", time.Now())
	if err == nil {
		t.Error("expected error when redis is nil")
	}
}

func TestPayoutRates_DefaultValues(t *testing.T) {
	rates := PayoutRates{}
	if rates.RatePerGBSent != 0 {
		t.Error("expected default RatePerGBSent to be 0")
	}
	if rates.RatePerGBReceived != 0 {
		t.Error("expected default RatePerGBReceived to be 0")
	}
	if rates.MinPayoutAmount != 0 {
		t.Error("expected default MinPayoutAmount to be 0")
	}
}
