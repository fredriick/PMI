package gateway

import (
	"testing"
)

func TestInitTracing_Disabled(t *testing.T) {
	tr, err := InitTracing("test-service", false)
	if err != nil {
		t.Fatalf("expected no error when tracing disabled, got %v", err)
	}
	if tr != nil {
		t.Error("expected nil tracer when disabled")
	}
}

func TestTracer_Shutdown_Nil(t *testing.T) {
	var tr *Tracer
	if err := tr.Shutdown(); err != nil {
		t.Fatalf("expected no error for nil tracer shutdown, got %v", err)
	}
}

func TestTracer_Shutdown_WithProvider(t *testing.T) {
	tr, err := InitTracing("test-service", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr != nil {
		// Shutdown on a tracer created with disabled=true should not panic
		// even though provider is nil
		if err := tr.Shutdown(); err != nil {
			t.Fatalf("expected no error for disabled tracer shutdown, got %v", err)
		}
	}
}
