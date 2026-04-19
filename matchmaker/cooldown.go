package matchmaker

import (
	"context"
	"log"
	"sync"
	"time"
)

type CooldownService struct {
	redis *RedisClient
	mu    sync.RWMutex
	ttl   time.Duration
	stop  chan bool
}

func NewCooldownService(redis *RedisClient, ttl time.Duration) *CooldownService {
	return &CooldownService{
		redis: redis,
		ttl:   ttl,
		stop:  make(chan bool),
	}
}

func (c *CooldownService) Start() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stop:
			return
		}
	}
}

func (c *CooldownService) Stop() {
	close(c.stop)
}

func (c *CooldownService) AddCooldown(target string, nodeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.redis.client.SAdd(context.Background(), "cooldown:"+target, nodeID).Err(); err != nil {
		return err
	}
	return c.redis.client.Expire(context.Background(), "cooldown:"+target, c.ttl).Err()
}

func (c *CooldownService) IsInCooldown(target string, nodeID string) (bool, error) {
	return c.redis.IsInCooldown(target, nodeID)
}

func (c *CooldownService) cleanupExpired() {
	ctx := context.Background()
	keys, err := c.redis.client.Keys(ctx, "cooldown:*").Result()
	if err != nil {
		log.Printf("Error getting cooldown keys: %v", err)
		return
	}

	for _, key := range keys {
		ttl, err := c.redis.client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}

		if ttl <= 0 {
			err = c.redis.client.Del(ctx, key).Err()
			if err != nil {
				log.Printf("Error deleting expired cooldown key %s: %v", key, err)
			}
		}
	}
}

func (c *CooldownService) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ttl = ttl
}

func (c *CooldownService) GetTTL() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ttl
}