package pools

import "sync"

// FastPool is a zero-overhead object pool without statistics
// Use this for hot path where every nanosecond counts
type FastPool struct {
	pool sync.Pool
}

// NewFastPool creates a fast pool without any overhead
func NewFastPool(newFunc func() any) *FastPool {
	return &FastPool{
		pool: sync.Pool{
			New: newFunc,
		},
	}
}

// Get acquires an object from the pool
func (fp *FastPool) Get() any {
	return fp.pool.Get()
}

// Put returns an object to the pool
func (fp *FastPool) Put(obj any) {
	if obj != nil {
		fp.pool.Put(obj)
	}
}

// Fast buffer pools without statistics overhead
var (
	smallBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, 0, 2048) // 2KB
			return &buf
		},
	}

	mediumBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, 0, 8192) // 8KB
			return &buf
		},
	}

	largeBufferPool = sync.Pool{
		New: func() any {
			buf := make([]byte, 0, 32768) // 32KB
			return &buf
		},
	}
)

// AcquireFastBuffer gets a buffer without statistics tracking
func AcquireFastBuffer(estimatedSize int) *[]byte {
	if estimatedSize <= 2048 {
		return smallBufferPool.Get().(*[]byte)
	} else if estimatedSize <= 8192 {
		return mediumBufferPool.Get().(*[]byte)
	} else {
		return largeBufferPool.Get().(*[]byte)
	}
}

// ReleaseFastBuffer returns a buffer without statistics tracking
func ReleaseFastBuffer(buf *[]byte) {
	if buf == nil {
		return
	}

	// Reset but keep capacity
	*buf = (*buf)[:0]

	// Return to appropriate pool
	cap := cap(*buf)
	if cap <= 2048 {
		smallBufferPool.Put(buf)
	} else if cap <= 8192 {
		mediumBufferPool.Put(buf)
	} else if cap <= 32768 {
		largeBufferPool.Put(buf)
	}
	// Oversized buffers are not pooled
}
