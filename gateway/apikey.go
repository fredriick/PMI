package gateway

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type APIKeyService struct {
	client *redis.Client
	ctx    context.Context
}

func NewAPIKeyService(client *redis.Client) *APIKeyService {
	return &APIKeyService{
		client: client,
		ctx:    context.Background(),
	}
}

type APIKey struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Scope     string    `json:"scope"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	LastUsed  time.Time `json:"last_used,omitempty"`
}

type APIKeyScope string

const (
	ScopeRead  APIKeyScope = "read"
	ScopeWrite APIKeyScope = "write"
	ScopeAdmin APIKeyScope = "admin"
	ScopeAll   APIKeyScope = "*"
)

var validScopes = map[APIKeyScope]bool{
	ScopeRead:  true,
	ScopeWrite: true,
	ScopeAdmin: true,
	ScopeAll:   true,
}

func (s *APIKeyService) CreateKey(name string, ttlDays int, scope string) (*APIKey, error) {
	if !validScopes[APIKeyScope(scope)] {
		scope = string(ScopeRead)
	}

	rawKey := generateKey(32)
	hash := hashKey(rawKey)

	key := &APIKey{
		Key:       rawKey,
		Name:      name,
		Scope:     scope,
		CreatedAt: time.Now(),
	}

	redisKey := fmt.Sprintf("apikey:%s", hash)
	data := map[string]interface{}{
		"name":       name,
		"scope":      scope,
		"created_at": key.CreatedAt.Unix(),
	}

	if ttlDays > 0 {
		key.ExpiresAt = time.Now().AddDate(0, 0, ttlDays)
		data["expires_at"] = key.ExpiresAt.Unix()
		ttl := time.Until(key.ExpiresAt)
		if err := s.client.HSet(s.ctx, redisKey, data).Err(); err != nil {
			return nil, err
		}
		s.client.Expire(s.ctx, redisKey, ttl)
	} else {
		if err := s.client.HSet(s.ctx, redisKey, data).Err(); err != nil {
			return nil, err
		}
	}

	return key, nil
}

func (s *APIKeyService) ValidateKey(rawKey string) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("api key service not initialized")
	}
	hash := hashKey(rawKey)
	redisKey := fmt.Sprintf("apikey:%s", hash)

	data, err := s.client.HGetAll(s.ctx, redisKey).Result()
	if err != nil {
		return "", err
	}

	if len(data) == 0 {
		return "", nil
	}

	if expiresStr, ok := data["expires_at"]; ok {
		var expiresAt int64
		fmt.Sscanf(expiresStr, "%d", &expiresAt)
		if time.Now().Unix() > expiresAt {
			s.client.Del(s.ctx, redisKey)
			return "", nil
		}
	}

	s.client.HSet(s.ctx, redisKey+":meta", map[string]interface{}{
		"last_used": time.Now().Unix(),
	})

	return data["scope"], nil
}

func (s *APIKeyService) RevokeKey(rawKey string) error {
	hash := hashKey(rawKey)
	redisKey := fmt.Sprintf("apikey:%s", hash)
	return s.client.Del(s.ctx, redisKey).Err()
}

func (s *APIKeyService) RevokeKeyByHash(hash string) error {
	redisKey := fmt.Sprintf("apikey:%s", hash)
	return s.client.Del(s.ctx, redisKey).Err()
}

func (s *APIKeyService) ListKeys() ([]map[string]string, error) {
	keys, err := s.client.Keys(s.ctx, "apikey:*").Result()
	if err != nil {
		return nil, err
	}

	var result []map[string]string
	for _, redisKey := range keys {
		if strings.HasSuffix(redisKey, ":meta") {
			continue
		}
		data, err := s.client.HGetAll(s.ctx, redisKey).Result()
		if err != nil {
			continue
		}
		data["hash"] = redisKey[len("apikey:"):]
		result = append(result, data)
	}

	return result, nil
}

func (s *APIKeyService) HasScope(rawKey string, requiredScope string) bool {
	if s == nil || s.client == nil {
		return false
	}
	scope, err := s.ValidateKey(rawKey)
	if err != nil || scope == "" {
		return false
	}

	if scope == "*" {
		return true
	}

	return scope == requiredScope
}

// SetKeyRateLimit sets a per-key rate limit (requests per window).
func (s *APIKeyService) SetKeyRateLimit(rawKey string, requests int, windowSeconds int) error {
	hash := hashKey(rawKey)
	rlKey := fmt.Sprintf("apikey_rl:%s", hash)
	return s.client.HSet(s.ctx, rlKey, map[string]interface{}{
		"requests": requests,
		"window":   windowSeconds,
	}).Err()
}

// GetKeyRateLimit returns the per-key rate limit. Returns 0,0 if not set.
func (s *APIKeyService) GetKeyRateLimit(rawKey string) (int, int, error) {
	hash := hashKey(rawKey)
	rlKey := fmt.Sprintf("apikey_rl:%s", hash)
	data, err := s.client.HGetAll(s.ctx, rlKey).Result()
	if err != nil || len(data) == 0 {
		return 0, 0, err
	}

	var requests, window int
	fmt.Sscanf(data["requests"], "%d", &requests)
	fmt.Sscanf(data["window"], "%d", &window)
	return requests, window, nil
}

// GetKeyHash returns the hash of a raw API key.
func (s *APIKeyService) GetKeyHash(rawKey string) string {
	return hashKey(rawKey)
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func generateKey(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}
