package matchmaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type SessionPersistence struct {
	redis      *redis.Client
	ctx        context.Context
	mu         sync.RWMutex
	sessions   map[string]*SessionInfo
	defaultTTL time.Duration
}

type SessionInfo struct {
	SessionID    string                 `json:"session_id"`
	NodeID       string                 `json:"node_id"`
	UserID       string                 `json:"user_id"`
	CreatedAt    time.Time              `json:"created_at"`
	LastUsedAt   time.Time              `json:"last_used_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	FailoverNode string                 `json:"failover_node,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

func NewSessionPersistence(redisClient *redis.Client, defaultTTL time.Duration) *SessionPersistence {
	sp := &SessionPersistence{
		redis:      redisClient,
		ctx:        context.Background(),
		sessions:   make(map[string]*SessionInfo),
		defaultTTL: defaultTTL,
	}

	go sp.cleanupLoop()

	return sp
}

func (sp *SessionPersistence) CreateSession(sessionID, nodeID, userID string, failoverNode string) error {
	session := &SessionInfo{
		SessionID:    sessionID,
		NodeID:       nodeID,
		UserID:       userID,
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
		ExpiresAt:    time.Now().Add(sp.defaultTTL),
		FailoverNode: failoverNode,
	}

	if sp.redis != nil {
		data, _ := serializeSession(session)
		key := fmt.Sprintf("session_persist:%s", sessionID)
		sp.redis.Set(sp.redis.Context(), key, data, sp.defaultTTL)
	}

	sp.mu.Lock()
	sp.sessions[sessionID] = session
	sp.mu.Unlock()

	return nil
}

func (sp *SessionPersistence) GetNodeForSession(sessionID string) (string, bool) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if session, exists := sp.sessions[sessionID]; exists {
		if time.Now().Before(session.ExpiresAt) {
			session.LastUsedAt = time.Now()
			return session.NodeID, true
		}
	}

	if sp.redis != nil {
		key := fmt.Sprintf("session_persist:%s", sessionID)
		_, err := sp.redis.Get(sp.ctx, key).Result()
		if err == nil {
			sp.mu.RLock()
			if session, exists := sp.sessions[sessionID]; exists {
				sp.mu.RUnlock()
				if time.Now().Before(session.ExpiresAt) {
					return session.NodeID, true
				}
			}
			sp.mu.RUnlock()
		}
	}

	return "", false
}

func (sp *SessionPersistence) GetFailoverNode(sessionID string) string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if session, exists := sp.sessions[sessionID]; exists {
		return session.FailoverNode
	}
	return ""
}

func (sp *SessionPersistence) Failover(sessionID string) bool {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if session, exists := sp.sessions[sessionID]; exists {
		if session.FailoverNode != "" {
			session.NodeID = session.FailoverNode
			session.FailoverNode = ""
			return true
		}
	}
	return false
}

func (sp *SessionPersistence) DeleteSession(sessionID string) {
	sp.mu.Lock()
	delete(sp.sessions, sessionID)
	sp.mu.Unlock()

	if sp.redis != nil {
		key := fmt.Sprintf("session_persist:%s", sessionID)
		sp.redis.Del(sp.ctx, key)
	}
}

func (sp *SessionPersistence) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sp.mu.Lock()
		now := time.Now()
		for sessionID, session := range sp.sessions {
			if now.After(session.ExpiresAt) {
				delete(sp.sessions, sessionID)
				if sp.redis != nil {
					key := fmt.Sprintf("session_persist:%s", sessionID)
					sp.redis.Del(sp.ctx, key)
				}
			}
		}
		sp.mu.Unlock()
	}
}

func serializeSession(s *SessionInfo) (string, error) {
	return fmt.Sprintf("%s|%s|%s|%d|%d|%s", s.SessionID, s.NodeID, s.UserID, s.CreatedAt.Unix(), s.ExpiresAt.Unix(), s.FailoverNode), nil
}
