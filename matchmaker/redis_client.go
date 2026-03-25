package matchmaker

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"proxymesh/internal/config"
	"proxymesh/internal/models"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisClient(cfg *config.RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

func (r *RedisClient) AddNode(node *models.Node) error {
	key := fmt.Sprintf("nodes:%s", node.Country)
	err := r.client.ZAdd(r.ctx, key, &redis.Z{
		Score:  node.Reputation,
		Member: node.ID,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to add node to sorted set: %w", err)
	}

	metaKey := fmt.Sprintf("node_meta:%s", node.ID)
	err = r.client.HSet(r.ctx, metaKey, map[string]interface{}{
		"isp":      node.ISP,
		"battery":  node.Battery,
		"os":       node.OS,
		"lastSeen": node.LastSeen.Unix(),
	}).Err()

	return err
}

func (r *RedisClient) GetTopNodes(country string, count int64) ([]string, error) {
	key := fmt.Sprintf("nodes:%s", country)
	nodes, err := r.client.ZRevRange(r.ctx, key, 0, count-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get top nodes: %w", err)
	}
	return nodes, nil
}

func (r *RedisClient) GetNodeMeta(nodeID string) (*models.NodeMeta, error) {
	metaKey := fmt.Sprintf("node_meta:%s", nodeID)
	data, err := r.client.HGetAll(r.ctx, metaKey).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("node meta not found")
	}

	lastSeen := time.Now()
	if unix, ok := data["lastSeen"]; ok {
		if ts, err := strconv.ParseInt(unix, 10, 64); err == nil {
			lastSeen = time.Unix(ts, 0)
		}
	}

	battery := 0
	if b, ok := data["battery"]; ok {
		if parsed, err := strconv.Atoi(b); err == nil {
			battery = parsed
		}
	}

	return &models.NodeMeta{
		NodeID:   nodeID,
		ISP:      data["isp"],
		Battery:  battery,
		OS:       data["os"],
		LastSeen: lastSeen,
	}, nil
}

func (r *RedisClient) AddToCooldown(target string, nodeID string) error {
	key := fmt.Sprintf("cooldown:%s", target)
	return r.client.SAdd(r.ctx, key, nodeID).Err()
}

func (r *RedisClient) IsInCooldown(target string, nodeID string) (bool, error) {
	key := fmt.Sprintf("cooldown:%s", target)
	return r.client.SIsMember(r.ctx, key, nodeID).Result()
}

func (r *RedisClient) RemoveFromCooldown(target string, nodeID string) error {
	key := fmt.Sprintf("cooldown:%s", target)
	return r.client.SRem(r.ctx, key, nodeID).Err()
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
