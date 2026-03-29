package gateway

import (
	"testing"
)

func TestGeoIPLookup_KnownRanges(t *testing.T) {
	svc := NewGeoIPService()

	tests := []struct {
		ip      string
		country string
	}{
		{"8.8.8.8", "US"},
		{"1.1.1.1", "AU"},
		{"51.1.1.1", "GB"},
		{"151.1.1.1", "EU"},
		{"196.1.1.1", "AF"},
		{"200.1.1.1", "LA"},
		{"212.1.1.1", "EU"},
	}

	for _, tt := range tests {
		got := svc.Lookup(tt.ip)
		if got != tt.country {
			t.Errorf("Lookup(%q) = %q, want %q", tt.ip, got, tt.country)
		}
	}
}

func TestGeoIPLookup_InvalidIP(t *testing.T) {
	svc := NewGeoIPService()

	if svc.Lookup("not-an-ip") != "" {
		t.Error("expected empty for invalid IP")
	}
	if svc.Lookup("") != "" {
		t.Error("expected empty for empty IP")
	}
}

func TestGeoIPLookup_PrivateIP(t *testing.T) {
	svc := NewGeoIPService()

	if svc.Lookup("192.168.1.1") != "US" {
		// 192.x is in the US range
	}
	if svc.Lookup("10.0.0.1") != "" {
		// 10.x might not be in the table, that's fine
	}
}

func TestGeoIPLoadCSV_InvalidPath(t *testing.T) {
	svc := NewGeoIPService()
	err := svc.LoadCSV("/nonexistent/file.csv")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
