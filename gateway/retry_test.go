package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"proxymesh/internal/models"
)

func TestRetryHandler_New_DefaultConfig(t *testing.T) {
	rh := NewRetryHandler(RetryConfig{})
	if rh == nil {
		t.Fatal("NewRetryHandler returned nil")
	}
	if rh.config.MaxRetries != DefaultRetryConfig.MaxRetries {
		t.Errorf("expected default MaxRetries %d, got %d", DefaultRetryConfig.MaxRetries, rh.config.MaxRetries)
	}
}

func TestRetryHandler_New_CustomConfig(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   5,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Multiplier:   3.0,
	}
	rh := NewRetryHandler(config)
	if rh.config.MaxRetries != 5 {
		t.Errorf("expected MaxRetries 5, got %d", rh.config.MaxRetries)
	}
}

func TestRetryHandler_ExecuteWithRetry_Success(t *testing.T) {
	rh := NewRetryHandler(RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	attempts := 0
	node, err := rh.ExecuteWithRetry(context.Background(), func(attempt int) (*models.Node, error) {
		attempts++
		return &models.Node{ID: "node-1"}, nil
	}, nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if node == nil || node.ID != "node-1" {
		t.Error("expected node-1")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryHandler_ExecuteWithRetry_RetryableError(t *testing.T) {
	rh := NewRetryHandler(RetryConfig{
		MaxRetries:   2,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		RetryableErrors: []string{
			"connection refused",
		},
	})

	attempts := 0
	_, err := rh.ExecuteWithRetry(context.Background(), func(attempt int) (*models.Node, error) {
		attempts++
		return nil, errors.New("connection refused")
	}, nil)

	if err == nil {
		t.Error("expected error after max retries")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts (initial + 2 retries), got %d", attempts)
	}
}

func TestRetryHandler_ExecuteWithRetry_NonRetryableError(t *testing.T) {
	rh := NewRetryHandler(RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	attempts := 0
	_, err := rh.ExecuteWithRetry(context.Background(), func(attempt int) (*models.Node, error) {
		attempts++
		return nil, errors.New("authentication failed")
	}, nil)

	if err == nil {
		t.Error("expected error for non-retryable error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetryHandler_ExecuteWithRetry_EventualSuccess(t *testing.T) {
	rh := NewRetryHandler(RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		RetryableErrors: []string{
			"connection refused",
		},
	})

	attempts := 0
	node, err := rh.ExecuteWithRetry(context.Background(), func(attempt int) (*models.Node, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("connection refused")
		}
		return &models.Node{ID: "node-1"}, nil
	}, nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if node.ID != "node-1" {
		t.Errorf("expected node-1, got %q", node.ID)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryHandler_ExecuteWithRetry_ContextCancelled(t *testing.T) {
	rh := NewRetryHandler(RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := rh.ExecuteWithRetry(ctx, func(attempt int) (*models.Node, error) {
		return nil, errors.New("connection refused")
	}, nil)

	if err == nil {
		t.Error("expected context cancelled error")
	}
}

func TestRetryHandler_calculateDelay(t *testing.T) {
	rh := NewRetryHandler(RetryConfig{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		MaxJitter:    0,
	})

	delay0 := rh.calculateDelay(0)
	delay1 := rh.calculateDelay(1)
	delay2 := rh.calculateDelay(2)

	if delay0 <= 0 {
		t.Errorf("expected positive delay for attempt 0, got %v", delay0)
	}
	if delay1 <= delay0 {
		t.Errorf("expected delay to increase with attempt, got %v <= %v", delay1, delay0)
	}
	if delay2 <= delay1 {
		t.Errorf("expected delay to increase with attempt, got %v <= %v", delay2, delay1)
	}
}

func TestRetryHandler_calculateDelay_Capped(t *testing.T) {
	rh := NewRetryHandler(RetryConfig{
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   10.0,
		MaxJitter:    0,
	})

	delay := rh.calculateDelay(10)
	if delay <= 0 {
		t.Error("expected positive delay")
	}
}

func TestRetryHandler_isRetryableError(t *testing.T) {
	rh := NewRetryHandler(DefaultRetryConfig)

	if rh.isRetryableError(nil) != nil {
		t.Error("expected nil for nil error")
	}

	if rh.isRetryableError(errors.New("connection refused")) != nil {
		t.Error("expected nil (retryable) for connection refused")
	}

	if rh.isRetryableError(errors.New("authentication failed")) == nil {
		t.Error("expected non-nil (non-retryable) for authentication failed")
	}
}

func TestRetryHandler_SetMaxRetries(t *testing.T) {
	rh := NewRetryHandler(RetryConfig{MaxRetries: 3})

	rh.SetMaxRetries(5)
	if rh.config.MaxRetries != 5 {
		t.Errorf("expected MaxRetries 5, got %d", rh.config.MaxRetries)
	}
}

func TestRetryHandler_SetRetryableErrors(t *testing.T) {
	rh := NewRetryHandler(RetryConfig{
		RetryableErrors: []string{"original"},
	})

	rh.SetRetryableErrors([]string{"custom error"})
	if rh.isRetryableError(errors.New("custom error")) != nil {
		t.Error("expected nil (retryable) for custom error")
	}
	if rh.isRetryableError(errors.New("original")) == nil {
		t.Error("expected non-nil (non-retryable) after replacing retryable errors")
	}
}

func TestContainsString(t *testing.T) {
	if !containsString("connection refused", "refused") {
		t.Error("expected containsString to find substring")
	}
	if containsString("hello world", "xyz") {
		t.Error("expected containsString to not find substring")
	}
	if !containsString("abc", "abc") {
		t.Error("expected containsString to match exact string")
	}
	if !containsString("", "") {
		t.Error("expected containsString to match empty strings")
	}
}

func TestContainsAny(t *testing.T) {
	if !containsAny("hello world", "world") {
		t.Error("expected containsAny to find substring")
	}
	if containsAny("hello", "xyz") {
		t.Error("expected containsAny to not find substring")
	}
}

func TestNodeRetryState_RecordFailure_Success(t *testing.T) {
	nrs := &NodeRetryState{}

	nrs.RecordFailure()
	if nrs.consecutiveFailures != 1 {
		t.Errorf("expected consecutiveFailures 1, got %d", nrs.consecutiveFailures)
	}

	nrs.RecordSuccess()
	if nrs.consecutiveFailures != 0 {
		t.Errorf("expected consecutiveFailures 0 after success, got %d", nrs.consecutiveFailures)
	}
}

func TestNodeRetryState_IsBackedOff(t *testing.T) {
	nrs := &NodeRetryState{}

	if nrs.IsBackedOff() {
		t.Error("expected not backed off initially")
	}

	nrs.SetBackoff(1 * time.Hour)
	if !nrs.IsBackedOff() {
		t.Error("expected backed off after SetBackoff")
	}
}

func TestNodeRetryState_GetBackoffDuration(t *testing.T) {
	nrs := &NodeRetryState{}

	duration := nrs.GetBackoffDuration()
	if duration <= 0 {
		t.Error("expected positive backoff duration")
	}

	nrs.consecutiveFailures = 5
	duration = nrs.GetBackoffDuration()
	if duration <= 0 {
		t.Error("expected positive backoff duration with failures")
	}
}
