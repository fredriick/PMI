package gateway

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type RequestLogEntry struct {
	Timestamp    int64  `json:"timestamp"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	Status       int    `json:"status"`
	Duration     int64  `json:"duration_ms"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent"`
	RequestSize  int64  `json:"request_size"`
	ResponseSize int64  `json:"response_size"`
	Error        string `json:"error,omitempty"`
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

type HTTPRequestLogger struct {
	redisClient *redis.Client
	enabled     bool
	maxBodySize int
}

func NewRequestLogger(redisClient *redis.Client, enabled bool) *HTTPRequestLogger {
	return &HTTPRequestLogger{
		redisClient: redisClient,
		enabled:     enabled,
		maxBodySize: 4096,
	}
}

func (rl *HTTPRequestLogger) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.enabled {
			c.Next()
			return
		}

		start := time.Now()

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		duration := time.Since(start).Milliseconds()

		logEntry := RequestLogEntry{
			Timestamp:    start.Unix(),
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			Status:       c.Writer.Status(),
			Duration:     duration,
			ClientIP:     c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			RequestSize:  c.Request.ContentLength,
			ResponseSize: int64(blw.body.Len()),
			Error:        c.Errors.ByType(gin.ErrorTypePrivate).String(),
		}

		go rl.storeLog(logEntry)
	}
}

func (rl *HTTPRequestLogger) storeLog(entry RequestLogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	key := "reqlog:" + time.Now().Format("20060102")
	rl.redisClient.RPush(rl.redisClient.Context(), key, data)
	rl.redisClient.Expire(rl.redisClient.Context(), key, 7*24*time.Hour)
}

func (rl *HTTPRequestLogger) GetLogs(date string, limit int64) ([]RequestLogEntry, error) {
	key := "reqlog:" + date
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	data, err := rl.redisClient.LRange(rl.redisClient.Context(), key, -int64(limit), -1).Result()
	if err != nil {
		return nil, err
	}

	var logs []RequestLogEntry
	for _, d := range data {
		var log RequestLogEntry
		if err := json.Unmarshal([]byte(d), &log); err == nil {
			logs = append(logs, log)
		}
	}

	return logs, nil
}
