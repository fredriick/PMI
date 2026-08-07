package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type RateLimitTier struct {
	Name      string
	Requests  int
	WindowSec int
}

type RateLimitTiers struct {
	Tiers map[string]RateLimitTier
	redis *redis.Client
}

type TieredRateLimiter struct {
	limiter RateLimiter
	tiers   *RateLimitTiers
}

func NewRateLimitTiers(redisClient *redis.Client) *RateLimitTiers {
	return &RateLimitTiers{
		Tiers: map[string]RateLimitTier{
			"free":       {Name: "free", Requests: 10, WindowSec: 60},
			"basic":      {Name: "basic", Requests: 100, WindowSec: 60},
			"premium":    {Name: "premium", Requests: 1000, WindowSec: 60},
			"enterprise": {Name: "enterprise", Requests: 10000, WindowSec: 60},
		},
		redis: redisClient,
	}
}

func (r *RateLimitTiers) GetTier(tierName string) RateLimitTier {
	if tier, ok := r.Tiers[tierName]; ok {
		return tier
	}
	return r.Tiers["free"]
}

func (r *RateLimitTiers) GetTierForKey(apiKey string) string {
	if r.redis == nil || apiKey == "" {
		return "basic"
	}
	tier, err := r.redis.Get(context.Background(), "key:tier:"+apiKey).Result()
	if err != nil || tier == "" {
		return "basic"
	}
	return tier
}

func (r *RateLimitTiers) SetKeyTier(apiKey, tier string) error {
	if r.redis == nil {
		return nil
	}
	return r.redis.Set(context.Background(), "key:tier:"+apiKey, tier, 0).Err()
}

func (r *RateLimitTiers) GetKeyTier(apiKey string) string {
	return r.GetTierForKey(apiKey)
}

func (r *RateLimitTiers) ListTiers() []RateLimitTier {
	var tiers []RateLimitTier
	for _, tier := range r.Tiers {
		tiers = append(tiers, tier)
	}
	return tiers
}

func NewTieredRateLimiter(limiter RateLimiter, tiers *RateLimitTiers) *TieredRateLimiter {
	return &TieredRateLimiter{
		limiter: limiter,
		tiers:   tiers,
	}
}

func (t *TieredRateLimiter) Allow(clientID, apiKey string) (bool, error) {
	tierName := t.tiers.GetTierForKey(apiKey)
	tier := t.tiers.GetTier(tierName)
	
	originalLimit := t.limiter.GetLimit()
	t.limiter.SetLimit(tier.Requests)
	
	allowed, err := t.limiter.Allow(clientID)
	
	t.limiter.SetLimit(originalLimit)
	
	return allowed, err
}

func (t *TieredRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.Request.Header.Get("Proxy-Authorization")
		}

		clientID := c.ClientIP()
		if authHeader != "" {
			clientID = clientID + ":" + authHeader
		}

		allowed, err := t.Allow(clientID, authHeader)
		if err != nil {
			c.Next()
			return
		}

		if !allowed {
			metrics.IncRateLimited()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

func GenerateReferralCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[n.Int64()]
	}
	return "REF-" + hex.EncodeToString(b)[:8]
}
