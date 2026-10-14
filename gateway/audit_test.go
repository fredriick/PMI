package gateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAuditLogger_NoFile(t *testing.T) {
	al, err := NewAuditLogger("", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if al == nil {
		t.Fatal("expected non-nil AuditLogger")
	}
	if al.file != nil {
		t.Error("expected nil file when logPath is empty")
	}
}

func TestNewAuditLogger_WithFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")

	al, err := NewAuditLogger(logPath, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if al == nil {
		t.Fatal("expected non-nil AuditLogger")
	}
	if al.file == nil {
		t.Error("expected non-nil file when logPath is provided")
	}
	defer al.Close()
}

func TestAuditLogger_Log_NoFile(t *testing.T) {
	al := &AuditLogger{
		redis: nil,
		ctx:   nil,
	}

	entry := AuditEntry{
		Action:   "test_action",
		Resource: "/test",
		UserIP:   "127.0.0.1",
		Method:   "GET",
		Path:     "/test",
		Success:  true,
	}

	al.Log(entry)
}

func TestAuditLogger_Close(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")

	al, err := NewAuditLogger(logPath, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	al.Close()

	// Verify file was created
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("expected audit log file to exist, got %v", err)
	}
	if len(data) != 0 {
		t.Error("expected empty audit log file when no entries logged")
	}
}

func TestAuditLogger_Log_WithFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")

	al, err := NewAuditLogger(logPath, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer al.Close()

	entry := AuditEntry{
		Action:   "test_action",
		Resource: "/test",
		UserIP:   "127.0.0.1",
		Method:   "GET",
		Path:     "/test",
		Success:  true,
	}

	al.Log(entry)
	al.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("expected audit log file to exist, got %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "test_action") {
		t.Error("expected audit log to contain action")
	}
	if !strings.Contains(content, "/test") {
		t.Error("expected audit log to contain path")
	}
}

func TestAuditLogger_Log_SuccessFalse(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")

	al, err := NewAuditLogger(logPath, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer al.Close()

	entry := AuditEntry{
		Action:   "test_action",
		Resource: "/test",
		UserIP:   "127.0.0.1",
		Method:   "POST",
		Path:     "/test",
		Success:  false,
		Error:    "something went wrong",
	}

	al.Log(entry)
	al.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("expected audit log file to exist, got %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "something went wrong") {
		t.Error("expected audit log to contain error message")
	}
}

func TestAuditLogger_GetEntries_NilRedis_Audit(t *testing.T) {
	al := &AuditLogger{
		redis: nil,
		ctx:   nil,
	}

	_, err := al.GetEntries("2026-01-01", "", 10)
	if err == nil {
		t.Error("expected error when Redis is nil")
	}
}
