package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type LogLevel string

const (
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelDebug LogLevel = "debug"
)

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"request_id,omitempty"`
	ClientIP  string                 `json:"client_ip,omitempty"`
	Method    string                 `json:"method,omitempty"`
	Path      string                 `json:"path,omitempty"`
	Status    int                    `json:"status,omitempty"`
	Latency   int64                  `json:"latency_ms,omitempty"`
	NodeID    string                 `json:"node_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

type Logger struct {
	output   io.Writer
	mu       sync.Mutex
	minLevel LogLevel
}

var (
	logger *Logger
	once   sync.Once
)

func InitLogger(level LogLevel, outputPath string) {
	once.Do(func() {
		var output io.Writer = os.Stdout

		if outputPath != "" {
			file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				output = file
			}
		}

		logger = &Logger{
			output:   output,
			minLevel: level,
		}

		log.SetOutput(io.Discard)
	})
}

func getLogger() *Logger {
	if logger == nil {
		logger = &Logger{
			output:   os.Stdout,
			minLevel: LevelInfo,
		}
	}
	return logger
}

func (l *Logger) log(level LogLevel, msg string, fields map[string]interface{}) {
	if level < l.minLevel {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     string(level),
		Message:   msg,
		Fields:    fields,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	jsonBytes, _ := json.Marshal(entry)
	l.output.Write(append(jsonBytes, '\n'))
}

func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.log(LevelInfo, msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.log(LevelError, msg, fields)
}

func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.log(LevelWarn, msg, fields)
}

func (l *Logger) Debug(msg string, fields map[string]interface{}) {
	l.log(LevelDebug, msg, fields)
}

func Info(msg string, fields map[string]interface{}) {
	getLogger().Info(msg, fields)
}

func Error(msg string, fields map[string]interface{}) {
	getLogger().Error(msg, fields)
}

func Warn(msg string, fields map[string]interface{}) {
	getLogger().Warn(msg, fields)
}

func Debug(msg string, fields map[string]interface{}) {
	getLogger().Debug(msg, fields)
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		requestIDValue, ok := c.Get("request_id")
		requestID, _ := requestIDValue.(string)
		if !ok || requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		c.Set("request_id", requestID)

		c.Next()

		latency := time.Since(start).Milliseconds()
		status := c.Writer.Status()

		Info("Request processed", map[string]interface{}{
			"request_id": requestID,
			"client_ip":  clientIP,
			"method":     method,
			"path":       path,
			"status":     status,
			"latency_ms": latency,
		})
	}
}
