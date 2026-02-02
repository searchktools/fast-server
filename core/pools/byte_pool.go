package pools

import "sync"

// BytePool is a multi-tiered byte slice pool for different size classes
type BytePool struct {
	pools []*sync.Pool
	sizes []int
}

// Common buffer sizes optimized for HTTP workloads
var defaultSizes = []int{
	512,   // Small requests/responses
	2048,  // Medium (most common)
	8192,  // Large
	32768, // Extra large
}

// NewBytePool creates a new byte pool with standard size tiers
func NewBytePool() *BytePool {
	return NewBytePoolWithSizes(defaultSizes)
}

// NewBytePoolWithSizes creates a byte pool with custom size tiers
func NewBytePoolWithSizes(sizes []int) *BytePool {
	bp := &BytePool{
		pools: make([]*sync.Pool, len(sizes)),
		sizes: sizes,
	}

	for i, size := range sizes {
		sz := size // Capture for closure
		bp.pools[i] = &sync.Pool{
			New: func() any {
				buf := make([]byte, sz)
				return &buf
			},
		}
	}

	return bp
}

// Get returns a byte slice of at least the requested size
func (bp *BytePool) Get(size int) []byte {
	// Find the appropriate pool
	for i, poolSize := range bp.sizes {
		if size <= poolSize {
			bufPtr := bp.pools[i].Get().(*[]byte)
			buf := *bufPtr
			return buf[:size] // Return slice with requested length
		}
	}

	// Size too large, allocate directly
	return make([]byte, size)
}

// Put returns a byte slice to the pool
func (bp *BytePool) Put(buf []byte) {
	capacity := cap(buf)

	// Find matching pool by capacity
	for i, poolSize := range bp.sizes {
		if capacity == poolSize {
			// Reset length to capacity
			buf = buf[:capacity]
			bp.pools[i].Put(&buf)
			return
		}
	}

	// Not from pool, let GC handle it
}

// GetBuffer returns a buffer pointer for zero-copy operations
func (bp *BytePool) GetBuffer(size int) *[]byte {
	for i, poolSize := range bp.sizes {
		if size <= poolSize {
			return bp.pools[i].Get().(*[]byte)
		}
	}

	buf := make([]byte, size)
	return &buf
}

// PutBuffer returns a buffer pointer to the pool
func (bp *BytePool) PutBuffer(buf *[]byte) {
	if buf == nil {
		return
	}

	capacity := cap(*buf)
	for i, poolSize := range bp.sizes {
		if capacity == poolSize {
			*buf = (*buf)[:capacity]
			bp.pools[i].Put(buf)
			return
		}
	}
}

// Stats returns pool statistics
type BytePoolStats struct {
	TotalGets  uint64
	TotalPuts  uint64
	ActiveBufs int
}

// Global byte pool instance
var globalBytePool = NewBytePool()

// GetBytes is a convenience function using the global pool
func GetBytes(size int) []byte {
	return globalBytePool.Get(size)
}

// PutBytes returns bytes to the global pool
func PutBytes(buf []byte) {
	globalBytePool.Put(buf)
}
