package subnet

import (
	"testing"
)

func TestCalculateSubnet_Index0(t *testing.T) {
	subnet, err := calculateSubnet("2001:db8::", 48, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subnet != "2001:db8::/64" {
		t.Errorf("expected 2001:db8::/64, got %s", subnet)
	}
}

func TestCalculateSubnet_Index1(t *testing.T) {
	subnet, err := calculateSubnet("2001:db8::", 48, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subnet == "2001:db8::/64" {
		t.Error("index 1 should produce a different subnet than index 0")
	}
	if subnet != "2001:db8:0:0:1::/64" {
		t.Errorf("expected 2001:db8:0:0:1::/64, got %s", subnet)
	}
}

func TestCalculateSubnet_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		subnet, err := calculateSubnet("2001:db8::", 48, i)
		if err != nil {
			t.Fatalf("unexpected error at index %d: %v", i, err)
		}
		if seen[subnet] {
			t.Errorf("duplicate subnet at index %d: %s", i, subnet)
		}
		seen[subnet] = true
	}

	if len(seen) != 100 {
		t.Errorf("expected 100 unique subnets, got %d", len(seen))
	}
}

func TestCalculateSubnet_InvalidPrefix(t *testing.T) {
	_, err := calculateSubnet("not-an-ip", 48, 0)
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func TestCalculateSubnet_Exhausted(t *testing.T) {
	// /62 prefix gives 4 subnets (2 bits)
	_, err := calculateSubnet("2001:db8::", 62, 4)
	if err == nil {
		t.Error("expected error when exceeding max subnets")
	}
}

func TestRegisterPool_InvalidPrefixLen(t *testing.T) {
	sa := &SubnetAllocator{client: nil, ctx: nil}

	err := sa.RegisterPool("2001:db8::", 32)
	if err == nil {
		t.Error("expected error for prefix_len < 48")
	}

	err = sa.RegisterPool("2001:db8::", 65)
	if err == nil {
		t.Error("expected error for prefix_len > 64")
	}
}

func TestRegisterPool_InvalidIP(t *testing.T) {
	sa := &SubnetAllocator{client: nil, ctx: nil}

	err := sa.RegisterPool("not-an-ip", 48)
	if err == nil {
		t.Error("expected error for invalid IP")
	}
}

func TestCalculateSubnet_PrefixLen56(t *testing.T) {
	subnet, err := calculateSubnet("2001:db8::", 56, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subnet != "2001:db8::/64" {
		t.Errorf("expected 2001:db8::/64, got %s", subnet)
	}

	subnet2, err := calculateSubnet("2001:db8::", 56, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subnet2 == subnet {
		t.Error("different indices should produce different subnets")
	}
}

func TestCalculateSubnet_PrefixLen64(t *testing.T) {
	// /64 prefix means 0 host bits, only 1 subnet possible
	subnet, err := calculateSubnet("2001:db8::", 64, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subnet != "2001:db8::/64" {
		t.Errorf("expected 2001:db8::/64, got %s", subnet)
	}

	_, err = calculateSubnet("2001:db8::", 64, 1)
	if err == nil {
		t.Error("expected error for index 1 with /64 prefix (no subnets available)")
	}
}
