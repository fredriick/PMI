package subnet

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
)

func redisClient(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 14})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	_ = rdb.FlushDB(context.Background()).Err()
	return rdb
}

func TestAllocate_NewNode(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	sa := &SubnetAllocator{client: rdb, ctx: context.Background()}

	if err := sa.RegisterPool("2001:db8::", 48); err != nil {
		t.Fatalf("RegisterPool failed: %v", err)
	}

	subnet, err := sa.Allocate("node1", "2001:db8::", 48)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if subnet == "" {
		t.Error("expected non-empty subnet")
	}
}

func TestAllocate_PoolNotFound(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	sa := &SubnetAllocator{client: rdb, ctx: context.Background()}

	_, err := sa.Allocate("node1", "nonexistent::", 48)
	if err == nil {
		t.Error("expected error for nonexistent pool")
	}
}

func TestAllocate_Idempotent(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	sa := &SubnetAllocator{client: rdb, ctx: context.Background()}

	sa.RegisterPool("2001:db8::", 48)

	subnet1, err := sa.Allocate("node1", "2001:db8::", 48)
	if err != nil {
		t.Fatalf("first Allocate failed: %v", err)
	}

	subnet2, err := sa.Allocate("node1", "2001:db8::", 48)
	if err != nil {
		t.Fatalf("second Allocate failed: %v", err)
	}

	if subnet1 != subnet2 {
		t.Errorf("expected same subnet for repeated allocation, got %s != %s", subnet1, subnet2)
	}
}

func TestAllocate_DifferentNodes(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	sa := &SubnetAllocator{client: rdb, ctx: context.Background()}

	sa.RegisterPool("2001:db8::", 48)

	subnet1, _ := sa.Allocate("node1", "2001:db8::", 48)
	subnet2, _ := sa.Allocate("node2", "2001:db8::", 48)

	if subnet1 == subnet2 {
		t.Error("expected different subnets for different nodes")
	}
}

func TestDeallocate(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	sa := &SubnetAllocator{client: rdb, ctx: context.Background()}

	sa.RegisterPool("2001:db8::", 48)

	sa.Allocate("node1", "2001:db8::", 48)

	if err := sa.Deallocate("node1"); err != nil {
		t.Fatalf("Deallocate failed: %v", err)
	}
}

func TestDeallocate_NonExistentNode(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	sa := &SubnetAllocator{client: rdb, ctx: context.Background()}

	if err := sa.Deallocate("nonexistent"); err != nil {
		t.Errorf("Deallocate for nonexistent node should not error: %v", err)
	}
}

func TestGetNodeSubnet(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	sa := &SubnetAllocator{client: rdb, ctx: context.Background()}

	sa.RegisterPool("2001:db8::", 48)

	expected, _ := sa.Allocate("node1", "2001:db8::", 48)

	got, err := sa.GetNodeSubnet("node1")
	if err != nil {
		t.Fatalf("GetNodeSubnet failed: %v", err)
	}
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}

	_, err = sa.GetNodeSubnet("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent node")
	}
}

func TestListPools(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	sa := &SubnetAllocator{client: rdb, ctx: context.Background()}

	sa.RegisterPool("2001:db8::", 48)
	sa.RegisterPool("2001:db9::", 56)

	pools, err := sa.ListPools()
	if err != nil {
		t.Fatalf("ListPools failed: %v", err)
	}
	if len(pools) != 2 {
		t.Errorf("expected 2 pools, got %d", len(pools))
	}
}

func TestListPools_Empty(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	sa := &SubnetAllocator{client: rdb, ctx: context.Background()}

	pools, err := sa.ListPools()
	if err != nil {
		t.Fatalf("ListPools failed: %v", err)
	}
	if len(pools) != 0 {
		t.Errorf("expected 0 pools for empty Redis, got %d", len(pools))
	}
}

func TestRegisterPool_Success(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	sa := &SubnetAllocator{client: rdb, ctx: context.Background()}

	if err := sa.RegisterPool("2001:db8::", 48); err != nil {
		t.Errorf("RegisterPool failed: %v", err)
	}

	if err := sa.RegisterPool("2001:db9::", 64); err != nil {
		t.Errorf("RegisterPool with /64 failed: %v", err)
	}
}

func TestGetNodeSubnet_NilClient(t *testing.T) {
	sa := &SubnetAllocator{client: nil, ctx: context.Background()}

	_, err := sa.GetNodeSubnet("node1")
	if err == nil {
		t.Error("expected error for nil Redis client")
	}
}
