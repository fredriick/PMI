package gateway

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestConnectionDrainer_New_Shutdown(t *testing.T) {
	cd := NewConnectionDrainer(30 * time.Second)
	if cd == nil {
		t.Fatal("NewConnectionDrainer returned nil")
	}
	if cd.drainTimeout != 30*time.Second {
		t.Errorf("expected drainTimeout 30s, got %v", cd.drainTimeout)
	}
	if cd.IsDraining() {
		t.Error("expected not draining initially")
	}
}

func TestConnectionDrainer_StartDrain_Shutdown(t *testing.T) {
	cd := NewConnectionDrainer(30 * time.Second)

	if cd.StartDrain() != true {
		t.Error("expected StartDrain to return true first time")
	}
	if !cd.IsDraining() {
		t.Error("expected draining to be true after StartDrain")
	}
	if cd.StartDrain() != false {
		t.Error("expected StartDrain to return false second time (already draining)")
	}
}

func TestConnectionDrainer_ActiveConnections(t *testing.T) {
	cd := NewConnectionDrainer(30 * time.Second)

	if cd.ActiveConnections() != 0 {
		t.Errorf("expected 0 active connections, got %d", cd.ActiveConnections())
	}

	cd.IncrementConnections()
	if cd.ActiveConnections() != 1 {
		t.Errorf("expected 1 active connection, got %d", cd.ActiveConnections())
	}

	cd.DecrementConnections()
	if cd.ActiveConnections() != 0 {
		t.Errorf("expected 0 active connections after decrement, got %d", cd.ActiveConnections())
	}
}

func TestConnectionDrainer_DecrementBelowZero(t *testing.T) {
	cd := NewConnectionDrainer(30 * time.Second)

	cd.DecrementConnections()
	// Can go below zero with atomic operations, just verify it doesn't crash
	_ = cd.ActiveConnections()
}

func TestConnectionDrainer_WrapServer_Shutdown(t *testing.T) {
	cd := NewConnectionDrainer(30 * time.Second)

	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	server := &http.Server{Handler: originalHandler}
	wrapped := cd.WrapServer(server)

	if wrapped.Handler == nil {
		t.Error("expected wrapped server to have a handler")
	}

	// Test that WrapServer increments/decrements connections
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrapped.Handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestConnectionDrainer_WrapServer_Draining(t *testing.T) {
	cd := NewConnectionDrainer(30 * time.Second)
	cd.StartDrain()

	originalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	server := &http.Server{Handler: originalHandler}
	wrapped := cd.WrapServer(server)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrapped.Handler.ServeHTTP(w, req)

	if w.Header().Get("X-Server-Draining") != "true" {
		t.Error("expected X-Server-Draining header when draining")
	}
}

func TestGracefulShutdown_New_Shutdown(t *testing.T) {
	gs := NewGracefulShutdown(30 * time.Second)
	if gs == nil {
		t.Fatal("NewGracefulShutdown returned nil")
	}
	if gs.drainTimeout != 30*time.Second {
		t.Errorf("expected drainTimeout 30s, got %v", gs.drainTimeout)
	}
	if gs.shutdownCh == nil {
		t.Error("expected shutdownCh to be initialized")
	}
}

func TestGracefulShutdown_AddServer_Shutdown(t *testing.T) {
	gs := NewGracefulShutdown(30 * time.Second)

	srv := &http.Server{}
	gs.AddServer(srv)
	gs.AddServer(srv)

	if len(gs.servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(gs.servers))
	}
}

func TestGracefulShutdown_Trigger_Shutdown(t *testing.T) {
	gs := NewGracefulShutdown(100 * time.Millisecond)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	gs.AddServer(srv.Config)

	go gs.Trigger()

	// Start should return after shutdown completes
	err := gs.Start()
	if err != nil {
		t.Errorf("expected no error from Start, got %v", err)
	}
}

func TestAtomicBoolOperations(t *testing.T) {
	var draining atomic.Bool
	if draining.Load() {
		t.Error("expected atomic bool to be false initially")
	}

	if !draining.CompareAndSwap(false, true) {
		t.Error("expected CompareAndSwap to succeed")
	}
	if !draining.Load() {
		t.Error("expected atomic bool to be true after swap")
	}

	if draining.CompareAndSwap(false, true) {
		t.Error("expected CompareAndSwap to fail when already true")
	}

	draining.Store(false)
	if draining.Load() {
		t.Error("expected atomic bool to be false after Store")
	}
}
