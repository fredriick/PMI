package gateway

import (
	"context"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type ConnectionDrainer struct {
	mu           sync.RWMutex
	activeConns  int64
	draining     atomic.Bool
	drainTimeout time.Duration
	listener     interface {
		Close() error
		Addr() string
	}
}

func NewConnectionDrainer(drainTimeout time.Duration) *ConnectionDrainer {
	return &ConnectionDrainer{
		drainTimeout: drainTimeout,
	}
}

func (cd *ConnectionDrainer) StartDrain() bool {
	if cd.draining.CompareAndSwap(false, true) {
		log.Printf("Starting connection drain (timeout: %v)", cd.drainTimeout)
		return true
	}
	return false
}

func (cd *ConnectionDrainer) IsDraining() bool {
	return cd.draining.Load()
}

func (cd *ConnectionDrainer) WaitForDrain() {
	if !cd.draining.Load() {
		return
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(cd.drainTimeout)

	for {
		select {
		case <-ticker.C:
			conns := cd.ActiveConnections()
			if conns == 0 {
				log.Printf("All connections drained")
				return
			}
		case <-timeout:
			log.Printf("Drain timeout reached, forcing shutdown")
			return
		}
	}
}

func (cd *ConnectionDrainer) ActiveConnections() int64 {
	return atomic.LoadInt64(&cd.activeConns)
}

func (cd *ConnectionDrainer) IncrementConnections() {
	atomic.AddInt64(&cd.activeConns, 1)
}

func (cd *ConnectionDrainer) DecrementConnections() {
	atomic.AddInt64(&cd.activeConns, -1)
}

func (cd *ConnectionDrainer) WrapServer(server *http.Server) *http.Server {
	originalHandler := server.Handler

	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cd.IsDraining() {
			w.Header().Set("X-Server-Draining", "true")
		}

		cd.IncrementConnections()
		defer cd.DecrementConnections()

		originalHandler.ServeHTTP(w, r)
	})

	return server
}

type GracefulShutdown struct {
	servers      []*http.Server
	drainTimeout time.Duration
	wg           sync.WaitGroup
	mu           sync.Mutex
	shutdownCh   chan struct{}
}

func NewGracefulShutdown(drainTimeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		drainTimeout: drainTimeout,
		shutdownCh:   make(chan struct{}),
	}
}

func (gs *GracefulShutdown) AddServer(server *http.Server) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.servers = append(gs.servers, server)
}

func (gs *GracefulShutdown) Start() error {
	<-gs.shutdownCh
	log.Println("Graceful shutdown initiated")

	gs.mu.Lock()
	servers := make([]*http.Server, len(gs.servers))
	copy(servers, gs.servers)
	gs.mu.Unlock()

	var wg sync.WaitGroup

	for _, server := range servers {
		wg.Add(1)
		go func(s *http.Server) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), gs.drainTimeout)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				log.Printf("Server shutdown error: %v", err)
			}
		}(server)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All servers shutdown complete")
	case <-time.After(gs.drainTimeout + time.Second):
		log.Println("Shutdown timeout, forcing exit")
	}

	return nil
}

func (gs *GracefulShutdown) Trigger() {
	close(gs.shutdownCh)
}
