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
	countryKey := fmt.Sprintf("nodes:%s", node.Country)
	err := r.client.ZAdd(r.ctx, countryKey, &redis.Z{
		Score:  node.Reputation,
		Member: node.ID,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to add node to sorted set: %w", err)
	}

	if node.City != "" {
		cityKey := fmt.Sprintf("nodes:%s:%s", node.Country, node.City)
		err = r.client.ZAdd(r.ctx, cityKey, &redis.Z{
			Score:  node.Reputation,
			Member: node.ID,
		}).Err()
		if err != nil {
			return fmt.Errorf("failed to add node to city sorted set: %w", err)
		}
	}

	metaKey := fmt.Sprintf("node_meta:%s", node.ID)
	err = r.client.HSet(r.ctx, metaKey, map[string]interface{}{
		"isp":      node.ISP,
		"battery":  node.Battery,
		"os":       node.OS,
		"city":     node.City,
		"country":  node.Country,
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

func (r *RedisClient) GetNodesByCity(country, city string, count int64) ([]string, error) {
	key := fmt.Sprintf("nodes:%s:%s", country, city)
	nodes, err := r.client.ZRevRange(r.ctx, key, 0, count-1).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get nodes by city: %w", err)
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
	if err := r.client.SAdd(r.ctx, key, nodeID).Err(); err != nil {
		return err
	}
	return r.client.Expire(r.ctx, key, 15*time.Minute).Err()
}

func (r *RedisClient) AddToCooldownWithTTL(target string, nodeID string, ttl time.Duration) error {
	key := fmt.Sprintf("cooldown:%s", target)
	if err := r.client.SAdd(r.ctx, key, nodeID).Err(); err != nil {
		return err
	}
	return r.client.Expire(r.ctx, key, ttl).Err()
}

func (r *RedisClient) IsInCooldown(target string, nodeID string) (bool, error) {
	key := fmt.Sprintf("cooldown:%s", target)
	return r.client.SIsMember(r.ctx, key, nodeID).Result()
}

func (r *RedisClient) RemoveFromCooldown(target string, nodeID string) error {
	key := fmt.Sprintf("cooldown:%s", target)
	return r.client.SRem(r.ctx, key, nodeID).Err()
}

func (r *RedisClient) GetCooldownEntries() (map[string][]string, error) {
	keys, err := r.client.Keys(r.ctx, "cooldown:*").Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, key := range keys {
		members, err := r.client.SMembers(r.ctx, key).Result()
		if err != nil {
			continue
		}
		target := key[len("cooldown:"):]
		ttl, _ := r.client.TTL(r.ctx, key).Result()
		label := fmt.Sprintf("%s (TTL: %s)", target, ttl)
		result[label] = members
	}
	return result, nil
}

func (r *RedisClient) CleanupExpiredCooldowns() (int, error) {
	keys, err := r.client.Keys(r.ctx, "cooldown:*").Result()
	if err != nil {
		return 0, err
	}

	cleaned := 0
	for _, key := range keys {
		ttl, err := r.client.TTL(r.ctx, key).Result()
		if err != nil {
			continue
		}
		if ttl == -1 {
			r.client.Del(r.ctx, key)
			cleaned++
		}
	}
	return cleaned, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

func (r *RedisClient) Client() *redis.Client {
	return r.client
}

func (r *RedisClient) UpdateNodeStatus(nodeID string, battery int, cpuUsage float64, isCharging bool) error {
	metaKey := fmt.Sprintf("node_meta:%s", nodeID)

	exists, err := r.client.Exists(r.ctx, metaKey).Result()
	if err != nil || exists == 0 {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	updates := map[string]interface{}{
		"lastSeen":   time.Now().Unix(),
		"battery":    battery,
		"cpuUsage":   cpuUsage,
		"isCharging": isCharging,
	}

	return r.client.HSet(r.ctx, metaKey, updates).Err()
}

func (r *RedisClient) RemoveNode(nodeID string) error {
	keys, err := r.client.Keys(r.ctx, fmt.Sprintf("*%s*", nodeID)).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.client.Del(r.ctx, keys...).Err()
	}
	return nil
}

func (r *RedisClient) GetNode(nodeID string) (*models.Node, error) {
	metaKey := fmt.Sprintf("node_meta:%s", nodeID)
	data, err := r.client.HGetAll(r.ctx, metaKey).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("node not found")
	}

	node := &models.Node{
		ID: nodeID,
	}

	if isp, ok := data["isp"]; ok {
		node.ISP = isp
	}
	if os, ok := data["os"]; ok {
		node.OS = os
	}
	if battery, ok := data["battery"]; ok {
		if b, err := strconv.Atoi(battery); err == nil {
			node.Battery = b
		}
	}

	keys, err := r.client.Keys(r.ctx, "nodes:*").Result()
	if err == nil && len(keys) > 0 {
		for _, key := range keys {
			r.client.ZScore(r.ctx, key, nodeID).Result()
		}
	}

	lastSeen := time.Now()
	if unix, ok := data["lastSeen"]; ok {
		if ts, err := strconv.ParseInt(unix, 10, 64); err == nil {
			lastSeen = time.Unix(ts, 0)
		}
	}
	node.LastSeen = lastSeen

	return node, nil
}

func (r *RedisClient) GetSessionNode(sessionID string) (string, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	nodeID, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return nodeID, nil
}

func (r *RedisClient) SetSessionNode(sessionID, nodeID string, ttlSeconds int) error {
	key := fmt.Sprintf("session:%s", sessionID)
	if ttlSeconds <= 0 {
		ttlSeconds = 3600
	}
	return r.client.SetEX(r.ctx, key, nodeID, time.Duration(ttlSeconds)*time.Second).Err()
}

func (r *RedisClient) GetNodeLoad(nodeID string) (int64, error) {
	key := fmt.Sprintf("node_load:%s", nodeID)
	load, err := r.client.Get(r.ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return load, err
}

func (r *RedisClient) IncrementNodeLoad(nodeID string) error {
	key := fmt.Sprintf("node_load:%s", nodeID)
	return r.client.Incr(r.ctx, key).Err()
}

func (r *RedisClient) DecrementNodeLoad(nodeID string) error {
	key := fmt.Sprintf("node_load:%s", nodeID)
	return r.client.Decr(r.ctx, key).Err()
}

func (r *RedisClient) RecordBandwidth(nodeID string, bytesSent, bytesReceived, durationSeconds int64) error {
	bandwidthKey := fmt.Sprintf("bandwidth:%s", nodeID)
	today := time.Now().Format("2006-01-02")

	pipe := r.client.Pipeline()
	pipe.HIncrBy(r.ctx, bandwidthKey, "bytes_sent", bytesSent)
	pipe.HIncrBy(r.ctx, bandwidthKey, "bytes_received", bytesReceived)
	pipe.HIncrBy(r.ctx, bandwidthKey, "duration_seconds", durationSeconds)
	pipe.SAdd(r.ctx, fmt.Sprintf("bandwidth_dates:%s", nodeID), today)
	pipe.Expire(r.ctx, bandwidthKey, 30*24*time.Hour)

	_, err := pipe.Exec(r.ctx)
	return err
}

func (r *RedisClient) GetBandwidth(nodeID string, period time.Time) (*models.BandwidthData, error) {
	bandwidthKey := fmt.Sprintf("bandwidth:%s", nodeID)
	data, err := r.client.HGetAll(r.ctx, bandwidthKey).Result()
	if err != nil {
		return nil, err
	}

	result := &models.BandwidthData{}
	if v, ok := data["bytes_sent"]; ok {
		result.BytesSent, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := data["bytes_received"]; ok {
		result.BytesReceived, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := data["duration_seconds"]; ok {
		result.DurationSeconds, _ = strconv.ParseInt(v, 10, 64)
	}

	return result, nil
}

func (r *RedisClient) GetBandwidthHistory(nodeID string) (map[string]models.BandwidthData, error) {
	dates, err := r.client.SMembers(r.ctx, fmt.Sprintf("bandwidth_dates:%s", nodeID)).Result()
	if err != nil {
		return nil, err
	}

	history := make(map[string]models.BandwidthData)
	for _, date := range dates {
		bandwidthKey := fmt.Sprintf("bandwidth:%s:%s", nodeID, date)
		data, err := r.client.HGetAll(r.ctx, bandwidthKey).Result()
		if err != nil {
			continue
		}

		bd := models.BandwidthData{}
		if v, ok := data["bytes_sent"]; ok {
			bd.BytesSent, _ = strconv.ParseInt(v, 10, 64)
		}
		if v, ok := data["bytes_received"]; ok {
			bd.BytesReceived, _ = strconv.ParseInt(v, 10, 64)
		}
		if v, ok := data["duration_seconds"]; ok {
			bd.DurationSeconds, _ = strconv.ParseInt(v, 10, 64)
		}
		history[date] = bd
	}

	return history, nil
}

func (r *RedisClient) GetAllNodes() ([]string, error) {
	keys, err := r.client.Keys(r.ctx, "node_meta:*").Result()
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, key := range keys {
		nodeID := key[len("node_meta:"):]
		nodes = append(nodes, nodeID)
	}

	return nodes, nil
}
