package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type AuditEntry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	UserIP    string `json:"user_ip"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Body      string `json:"body,omitempty"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

type AuditLogger struct {
	file  *os.File
	mu    sync.Mutex
	redis *redis.Client
	ctx   context.Context
}

func NewAuditLogger(logPath string, redisClient *redis.Client) (*AuditLogger, error) {
	al := &AuditLogger{
		redis: redisClient,
		ctx:   context.Background(),
	}

	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log: %w", err)
		}
		al.file = f
	}

	return al, nil
}

func (al *AuditLogger) Log(entry AuditEntry) {
	entry.Timestamp = time.Now().UTC().Format(time.RFC3339)

	al.mu.Lock()
	defer al.mu.Unlock()

	if al.file != nil {
		data, _ := json.Marshal(entry)
		al.file.Write(append(data, '\n'))
	}

	if al.redis != nil {
		key := fmt.Sprintf("audit:%s:%s", time.Now().Format("2006-01-02"), entry.Action)
		data, _ := json.Marshal(entry)
		al.redis.RPush(al.ctx, key, string(data))
		al.redis.Expire(al.ctx, key, 90*24*time.Hour)
	}
}

func (al *AuditLogger) Close() {
	if al.file != nil {
		al.file.Close()
	}
}

func (al *AuditLogger) GetEntries(date string, action string, limit int64) ([]AuditEntry, error) {
	if al.redis == nil {
		return nil, fmt.Errorf("Redis not configured for audit log queries")
	}

	var key string
	if action != "" {
		key = fmt.Sprintf("audit:%s:%s", date, action)
	} else {
		keys, err := al.redis.Keys(al.ctx, fmt.Sprintf("audit:%s:*", date)).Result()
		if err != nil {
			return nil, err
		}
		var allEntries []AuditEntry
		for _, k := range keys {
			entries, _ := al.getEntriesFromKey(k, limit)
			allEntries = append(allEntries, entries...)
		}
		return allEntries, nil
	}

	return al.getEntriesFromKey(key, limit)
}

func (al *AuditLogger) getEntriesFromKey(key string, limit int64) ([]AuditEntry, error) {
	var start int64 = 0
	if limit > 0 {
		length, _ := al.redis.LLen(al.ctx, key).Result()
		if length > limit {
			start = length - limit
		}
	}

	data, err := al.redis.LRange(al.ctx, key, start, -1).Result()
	if err != nil {
		return nil, err
	}

	var entries []AuditEntry
	for _, d := range data {
		var entry AuditEntry
		if err := json.Unmarshal([]byte(d), &entry); err == nil {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// AuditMiddleware creates a gin middleware that logs admin actions.
func (al *AuditLogger) AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Request.Method == "GET" {
			return
		}

		entry := AuditEntry{
			Action:   fmt.Sprintf("%s_%s", c.Request.Method, c.FullPath()),
			Resource: c.FullPath(),
			UserIP:   c.ClientIP(),
			Method:   c.Request.Method,
			Path:     c.Request.URL.Path,
			Success:  c.Writer.Status() < 400,
		}

		if !entry.Success {
			entry.Error = c.Errors.String()
		}

		al.Log(entry)
	}
}
