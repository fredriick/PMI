package gateway

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type ConnPool struct {
	pools   map[string]*nodePool
	mu      sync.RWMutex
	maxSize int
	timeout time.Duration
}

type nodePool struct {
	conns  chan net.Conn
	mu     sync.Mutex
	addr   string
	active int
}

func NewConnPool(maxPerNode int, dialTimeout time.Duration) *ConnPool {
	return &ConnPool{
		pools:   make(map[string]*nodePool),
		maxSize: maxPerNode,
		timeout: dialTimeout,
	}
}

func (p *ConnPool) getPool(addr string) *nodePool {
	p.mu.RLock()
	pool, exists := p.pools[addr]
	p.mu.RUnlock()

	if exists {
		return pool
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	pool, exists = p.pools[addr]
	if exists {
		return pool
	}

	pool = &nodePool{
		conns: make(chan net.Conn, p.maxSize),
		addr:  addr,
	}
	p.pools[addr] = pool
	return pool
}

func (p *ConnPool) Get(addr string) (net.Conn, error) {
	pool := p.getPool(addr)

	select {
	case conn := <-pool.conns:
		if conn != nil {
			pool.mu.Lock()
			pool.active--
			pool.mu.Unlock()
			return conn, nil
		}
	default:
	}

	conn, err := net.DialTimeout("tcp", addr, p.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	return conn, nil
}

func (p *ConnPool) Put(addr string, conn net.Conn) {
	if conn == nil {
		return
	}

	pool := p.getPool(addr)

	pool.mu.Lock()
	if pool.active >= p.maxSize {
		pool.mu.Unlock()
		conn.Close()
		return
	}
	pool.active++
	pool.mu.Unlock()

	select {
	case pool.conns <- conn:
	default:
		pool.mu.Lock()
		pool.active--
		pool.mu.Unlock()
		conn.Close()
	}
}

func (p *ConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pool := range p.pools {
		close(pool.conns)
		for conn := range pool.conns {
			conn.Close()
		}
	}
	p.pools = make(map[string]*nodePool)
}

func (p *ConnPool) Stats() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]int)
	for addr, pool := range p.pools {
		pool.mu.Lock()
		stats[addr] = pool.active
		pool.mu.Unlock()
	}
	return stats
}
