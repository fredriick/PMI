package gateway

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"proxymesh/internal/models"
)

type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	MaxJitter       time.Duration
	RetryableErrors []string
}

var DefaultRetryConfig = RetryConfig{
	MaxRetries:   3,
	InitialDelay: 100 * time.Millisecond,
	MaxDelay:     5 * time.Second,
	Multiplier:   2.0,
	MaxJitter:    100 * time.Millisecond,
	RetryableErrors: []string{
		"connection refused",
		"connection reset",
		"timeout",
		"i/o timeout",
		"no route to host",
		"network unreachable",
	},
}

type RetryHandler struct {
	config RetryConfig
	mu     sync.RWMutex
}

func NewRetryHandler(config RetryConfig) *RetryHandler {
	if config.MaxRetries == 0 {
		config = DefaultRetryConfig
	}
	return &RetryHandler{
		config: config,
	}
}

func (rh *RetryHandler) ExecuteWithRetry(
	ctx context.Context,
	fn func(attempt int) (*models.Node, error),
	onRetry func(attempt int, delay time.Duration, err error),
) (*models.Node, error) {
	var lastErr error
	var node *models.Node

	for attempt := 0; attempt <= rh.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		node, lastErr = fn(attempt)

		if lastErr == nil {
			return node, nil
		}

		if rh.isRetryableError(lastErr) != nil {
			return nil, lastErr
		}

		if attempt < rh.config.MaxRetries {
			delay := rh.calculateDelay(attempt)

			if onRetry != nil {
				onRetry(attempt, delay, lastErr)
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (rh *RetryHandler) calculateDelay(attempt int) time.Duration {
	delay := rh.config.InitialDelay * time.Duration(math.Pow(rh.config.Multiplier, float64(attempt)))

	if delay > rh.config.MaxDelay {
		delay = rh.config.MaxDelay
	}

	if rh.config.MaxJitter > 0 {
		jitter := time.Duration(int64(rh.config.MaxJitter))
		delay += time.Duration(int64(jitter) / int64(attempt+1))
	}

	return delay
}

func (rh *RetryHandler) isRetryableError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	for _, pattern := range rh.config.RetryableErrors {
		if containsString(errMsg, pattern) {
			return nil
		}
	}

	return err
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsAny(s, substr))))
}

func containsAny(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (rh *RetryHandler) SetMaxRetries(max int) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.config.MaxRetries = max
}

func (rh *RetryHandler) SetRetryableErrors(errors []string) {
	rh.mu.Lock()
	defer rh.mu.Unlock()
	rh.config.RetryableErrors = errors
}

type NodeRetryState struct {
	mu                  sync.Mutex
	consecutiveFailures int
	lastFailure         time.Time
	backoffUntil        time.Time
}

func (nrs *NodeRetryState) RecordFailure() {
	nrs.mu.Lock()
	defer nrs.mu.Unlock()
	nrs.consecutiveFailures++
	nrs.lastFailure = time.Now()
}

func (nrs *NodeRetryState) RecordSuccess() {
	nrs.mu.Lock()
	defer nrs.mu.Unlock()
	nrs.consecutiveFailures = 0
	nrs.backoffUntil = time.Time{}
}

func (nrs *NodeRetryState) IsBackedOff() bool {
	nrs.mu.Lock()
	defer nrs.mu.Unlock()
	return time.Now().Before(nrs.backoffUntil)
}

func (nrs *NodeRetryState) GetBackoffDuration() time.Duration {
	nrs.mu.Lock()
	defer nrs.mu.Unlock()
	base := DefaultRetryConfig.InitialDelay * time.Duration(math.Pow(DefaultRetryConfig.Multiplier, float64(nrs.consecutiveFailures)))
	if base > DefaultRetryConfig.MaxDelay {
		base = DefaultRetryConfig.MaxDelay
	}
	return base
}

func (nrs *NodeRetryState) SetBackoff(delay time.Duration) {
	nrs.mu.Lock()
	defer nrs.mu.Unlock()
	nrs.backoffUntil = time.Now().Add(delay)
}
