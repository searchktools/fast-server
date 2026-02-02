package pools

import (
	"sync"
	"sync/atomic"
)

// ConnectionPool manages Connection object pooling
type ConnectionPool struct {
	pool     sync.Pool
	gets     atomic.Uint64
	puts     atomic.Uint64
	capacity int
}

// ConnectionPoolable defines the interface for poolable connection objects
type ConnectionPoolable interface {
	Reset()
	SetFD(fd int)
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(capacity int, newFunc func() any) *ConnectionPool {
	cp := &ConnectionPool{
		capacity: capacity,
	}

	cp.pool.New = newFunc

	return cp
}

// Get retrieves a connection from the pool
func (cp *ConnectionPool) Get() any {
	cp.gets.Add(1)
	obj := cp.pool.Get()
	return obj
}

// Put returns a connection to the pool
func (cp *ConnectionPool) Put(obj any) {
	if poolable, ok := obj.(ConnectionPoolable); ok {
		poolable.Reset()
	}
	cp.puts.Add(1)
	cp.pool.Put(obj)
}

// Stats returns pool statistics
func (cp *ConnectionPool) Stats() (gets, puts uint64, hitRate float64) {
	g := cp.gets.Load()
	p := cp.puts.Load()

	if g > 0 {
		hitRate = float64(p) / float64(g)
	}

	return g, p, hitRate
}
