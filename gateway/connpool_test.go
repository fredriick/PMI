package gateway

import (
	"net"
	"testing"
	"time"
)

func TestConnPool_GetDialsNew(t *testing.T) {
	pool := NewConnPool(5, 2*time.Second)
	defer pool.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	conn, err := pool.Get(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to get connection: %v", err)
	}
	conn.Close()
}

func TestConnPool_PutAndGet(t *testing.T) {
	pool := NewConnPool(5, 2*time.Second)
	defer pool.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := listener.Addr().String()

	conn1, err := pool.Get(addr)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	pool.Put(addr, conn1)

	stats := pool.Stats()
	if stats[addr] != 1 {
		t.Errorf("expected 1 pooled connection, got %d", stats[addr])
	}
}

func TestConnPool_MaxSize(t *testing.T) {
	pool := NewConnPool(2, 2*time.Second)
	defer pool.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := listener.Addr().String()

	conn1, _ := pool.Get(addr)
	conn2, _ := pool.Get(addr)
	conn3, _ := pool.Get(addr)

	pool.Put(addr, conn1)
	pool.Put(addr, conn2)
	pool.Put(addr, conn3) // should be dropped, pool max is 2

	stats := pool.Stats()
	if stats[addr] != 2 {
		t.Errorf("expected 2 pooled connections, got %d", stats[addr])
	}
}

func TestConnPool_Close(t *testing.T) {
	pool := NewConnPool(5, 2*time.Second)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := listener.Addr().String()
	conn, _ := pool.Get(addr)
	pool.Put(addr, conn)

	pool.Close()

	stats := pool.Stats()
	if len(stats) != 0 {
		t.Error("pool should be empty after close")
	}
}

func TestConnPool_DialFailure(t *testing.T) {
	pool := NewConnPool(5, 500*time.Millisecond)
	defer pool.Close()

	_, err := pool.Get("127.0.0.1:1") // unlikely port
	if err == nil {
		t.Error("expected error for unreachable address")
	}
}
