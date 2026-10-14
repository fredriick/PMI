package gateway

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogger_InitLogger_Default(t *testing.T) {
	InitLogger(LevelInfo, "")
	if logger == nil {
		t.Error("expected logger to be initialized")
	}
	if logger.minLevel != LevelInfo {
		t.Errorf("expected min level info, got %s", logger.minLevel)
	}
}

func TestLogger_GetLogger_Default(t *testing.T) {
	// Reset global logger to test getLogger fallback
	prevLogger := logger
	logger = nil
	defer func() { logger = prevLogger }()

	l := getLogger()
	if l == nil {
		t.Fatal("expected non-nil logger from getLogger")
	}
	if l.minLevel != LevelInfo {
		t.Errorf("expected default min level info, got %s", l.minLevel)
	}
}

func TestLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		output:   &buf,
		minLevel: LevelDebug,
	}

	l.Debug("debug message", nil)
	if buf.Len() == 0 {
		t.Error("expected output for debug level")
	}
	buf.Reset()

	l.Info("info message", nil)
	if buf.Len() == 0 {
		t.Error("expected output for info level")
	}
	buf.Reset()

	l.Warn("warn message", nil)
	if buf.Len() == 0 {
		t.Error("expected output for warn level")
	}
	buf.Reset()

	l.Error("error message", nil)
	if buf.Len() == 0 {
		t.Error("expected output for error level")
	}
}

func TestLogger_Info_Error_Warn_Debug(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		output:   &buf,
		minLevel: LevelDebug,
	}

	l.Info("info test", map[string]interface{}{"key": "value"})
	l.Error("error test", map[string]interface{}{"err": "test error"})
	l.Warn("warn test", nil)
	l.Debug("debug test", nil)

	content := buf.String()
	if !strings.Contains(content, "info test") {
		t.Error("expected info message in log output")
	}
	if !strings.Contains(content, "error test") {
		t.Error("expected error message in log output")
	}
	if !strings.Contains(content, "warn test") {
		t.Error("expected warn message in log output")
	}
	if !strings.Contains(content, "debug test") {
		t.Error("expected debug message in log output")
	}
}

func TestLogEntry_JSONMarshal(t *testing.T) {
	entry := LogEntry{
		Timestamp: "2026-01-01T00:00:00Z",
		Level:     "info",
		Message:   "test message",
		RequestID: "req-123",
		ClientIP:  "127.0.0.1",
		Method:    "GET",
		Path:      "/test",
		Status:    200,
		Latency:   42,
		NodeID:    "node-1",
		Fields:    map[string]interface{}{"custom": "value"},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal log entry: %v", err)
	}

	var decoded LogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if decoded.Level != "info" {
		t.Errorf("expected level info, got %s", decoded.Level)
	}
	if decoded.Message != "test message" {
		t.Errorf("expected message test message, got %s", decoded.Message)
	}
}

func TestLogger_PackageFunctions(t *testing.T) {
	var buf bytes.Buffer
	prevLogger := logger
	logger = &Logger{
		output:   &buf,
		minLevel: LevelDebug,
	}
	defer func() { logger = prevLogger }()

	Info("package info", nil)
	Error("package error", nil)
	Warn("package warn", nil)
	Debug("package debug", nil)

	content := buf.String()
	if !strings.Contains(content, "package info") {
		t.Error("expected package info message")
	}
	if !strings.Contains(content, "package error") {
		t.Error("expected package error message")
	}
}
