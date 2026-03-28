package subnet

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
)

type SubnetAllocator struct {
	client *redis.Client
	ctx    context.Context
	mu     sync.Mutex
}

type SubnetPool struct {
	Prefix    string
	PrefixLen int
	Allocated map[string]string // nodeID -> subnet
}

func NewSubnetAllocator(client *redis.Client) *SubnetAllocator {
	return &SubnetAllocator{
		client: client,
		ctx:    context.Background(),
	}
}

func (s *SubnetAllocator) RegisterPool(prefix string, prefixLen int) error {
	if prefixLen < 48 || prefixLen > 64 {
		return fmt.Errorf("prefix length must be between 48 and 64")
	}

	ip := net.ParseIP(prefix)
	if ip == nil {
		return fmt.Errorf("invalid IPv6 prefix: %s", prefix)
	}

	key := fmt.Sprintf("subnet_pool:%s/%d", prefix, prefixLen)
	return s.client.HSet(s.ctx, key, map[string]interface{}{
		"prefix":     prefix,
		"prefix_len": prefixLen,
		"allocated":  0,
	}).Err()
}

func (s *SubnetAllocator) Allocate(nodeID string, prefix string, prefixLen int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	poolKey := fmt.Sprintf("subnet_pool:%s/%d", prefix, prefixLen)
	exists, err := s.client.Exists(s.ctx, poolKey).Result()
	if err != nil || exists == 0 {
		return "", fmt.Errorf("subnet pool not found: %s/%d", prefix, prefixLen)
	}

	allocKey := fmt.Sprintf("subnet_alloc:%s/%d", prefix, prefixLen)

	existing, err := s.client.HGet(s.ctx, allocKey, nodeID).Result()
	if err == nil && existing != "" {
		return existing, nil
	}

	count, err := s.client.HLen(s.ctx, allocKey).Result()
	if err != nil {
		return "", err
	}

	subnet, err := calculateSubnet(prefix, prefixLen, int(count))
	if err != nil {
		return "", err
	}

	if err := s.client.HSet(s.ctx, allocKey, nodeID, subnet).Err(); err != nil {
		return "", err
	}

	s.client.HIncrBy(s.ctx, poolKey, "allocated", 1)

	return subnet, nil
}

func (s *SubnetAllocator) Deallocate(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys, err := s.client.Keys(s.ctx, "subnet_alloc:*").Result()
	if err != nil {
		return err
	}

	for _, allocKey := range keys {
		exists, err := s.client.HExists(s.ctx, allocKey, nodeID).Result()
		if err != nil || !exists {
			continue
		}

		if err := s.client.HDel(s.ctx, allocKey, nodeID).Err(); err != nil {
			return err
		}

		poolKey := strings.Replace(allocKey, "subnet_alloc:", "subnet_pool:", 1)
		s.client.HIncrBy(s.ctx, poolKey, "allocated", -1)
		return nil
	}

	return nil
}

func (s *SubnetAllocator) GetNodeSubnet(nodeID string) (string, error) {
	keys, err := s.client.Keys(s.ctx, "subnet_alloc:*").Result()
	if err != nil {
		return "", err
	}

	for _, key := range keys {
		subnet, err := s.client.HGet(s.ctx, key, nodeID).Result()
		if err == nil && subnet != "" {
			return subnet, nil
		}
	}

	return "", fmt.Errorf("no subnet allocated for node: %s", nodeID)
}

func (s *SubnetAllocator) ListPools() ([]SubnetPoolInfo, error) {
	keys, err := s.client.Keys(s.ctx, "subnet_pool:*").Result()
	if err != nil {
		return nil, err
	}

	var pools []SubnetPoolInfo
	for _, key := range keys {
		data, err := s.client.HGetAll(s.ctx, key).Result()
		if err != nil {
			continue
		}

		info := SubnetPoolInfo{
			Prefix:    data["prefix"],
			PrefixLen: data["prefix_len"],
			Allocated: data["allocated"],
		}

		allocKey := strings.Replace(key, "subnet_pool:", "subnet_alloc:", 1)
		allocations, _ := s.client.HGetAll(s.ctx, allocKey).Result()
		info.Allocations = allocations

		pools = append(pools, info)
	}

	return pools, nil
}

type SubnetPoolInfo struct {
	Prefix      string            `json:"prefix"`
	PrefixLen   string            `json:"prefix_len"`
	Allocated   string            `json:"allocated"`
	Allocations map[string]string `json:"allocations,omitempty"`
}

func calculateSubnet(prefix string, prefixLen int, index int) (string, error) {
	ip := net.ParseIP(prefix)
	if ip == nil {
		return "", fmt.Errorf("invalid prefix: %s", prefix)
	}

	subnetLen := 64
	hostBits := subnetLen - prefixLen
	maxSubnets := 1 << uint(hostBits)
	if index >= maxSubnets {
		return "", fmt.Errorf("no more subnets available (max: %d)", maxSubnets)
	}

	ipBytes := make([]byte, 16)
	copy(ipBytes, ip.To16())

	byteStart := prefixLen / 8
	for i := index; i > 0 && byteStart < 16; i >>= 8 {
		ipBytes[15-byteStart] |= byte(i & 0xFF)
		byteStart++
	}

	resultIP := net.IP(ipBytes)
	return fmt.Sprintf("%s/%d", resultIP.String(), subnetLen), nil
}
