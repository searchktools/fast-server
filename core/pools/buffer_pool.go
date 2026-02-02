package pools

import (
	"sync"
	"sync/atomic"
)

// Buffer pool sizes
const (
	SmallBufferSize  = 2 * 1024  // 2KB for simple responses
	MediumBufferSize = 8 * 1024  // 8KB for typical JSON
	LargeBufferSize  = 32 * 1024 // 32KB for complex responses
)

// BufferPool manages response buffers with three size tiers
type BufferPool struct {
	small  sync.Pool
	medium sync.Pool
	large  sync.Pool

	// Statistics
	smallHits  atomic.Uint64
	mediumHits atomic.Uint64
	largeHits  atomic.Uint64
	totalGets  atomic.Uint64
}

// NewBufferPool creates a new buffer pool
func NewBufferPool() *BufferPool {
	return &BufferPool{
		small: sync.Pool{
			New: func() any {
				buf := make([]byte, 0, SmallBufferSize)
				return &buf
			},
		},
		medium: sync.Pool{
			New: func() any {
				buf := make([]byte, 0, MediumBufferSize)
				return &buf
			},
		},
		large: sync.Pool{
			New: func() any {
				buf := make([]byte, 0, LargeBufferSize)
				return &buf
			},
		},
	}
}

// Get acquires a buffer of appropriate size
func (bp *BufferPool) Get(estimatedSize int) *[]byte {
	bp.totalGets.Add(1)

	if estimatedSize <= SmallBufferSize {
		bp.smallHits.Add(1)
		return bp.small.Get().(*[]byte)
	} else if estimatedSize <= MediumBufferSize {
		bp.mediumHits.Add(1)
		return bp.medium.Get().(*[]byte)
	} else {
		bp.largeHits.Add(1)
		return bp.large.Get().(*[]byte)
	}
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf *[]byte) {
	if buf == nil {
		return
	}

	// Reset buffer but keep capacity
	*buf = (*buf)[:0]

	// Return to appropriate pool based on capacity
	cap := cap(*buf)
	if cap <= SmallBufferSize {
		bp.small.Put(buf)
	} else if cap <= MediumBufferSize {
		bp.medium.Put(buf)
	} else if cap <= LargeBufferSize {
		bp.large.Put(buf)
	}
	// Oversized buffers are not pooled (let GC collect them)
}

// Stats returns buffer pool statistics
func (bp *BufferPool) Stats() BufferStats {
	total := bp.totalGets.Load()
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(bp.smallHits.Load()+bp.mediumHits.Load()+bp.largeHits.Load()) / float64(total)
	}
	return BufferStats{
		SmallHits:  bp.smallHits.Load(),
		MediumHits: bp.mediumHits.Load(),
		LargeHits:  bp.largeHits.Load(),
		TotalGets:  total,
		HitRate:    hitRate,
	}
}

// BufferStats contains buffer pool statistics
type BufferStats struct {
	SmallHits  uint64
	MediumHits uint64
	LargeHits  uint64
	TotalGets  uint64
	HitRate    float64
}

// Global buffer pool
var globalBufferPool = NewBufferPool()

// AcquireBuffer gets a buffer from the global pool
func AcquireBuffer(estimatedSize int) *[]byte {
	return globalBufferPool.Get(estimatedSize)
}

// ReleaseBuffer returns a buffer to the global pool
func ReleaseBuffer(buf *[]byte) {
	globalBufferPool.Put(buf)
}

// GetBufferStats returns statistics for the global buffer pool
func GetBufferStats() BufferStats {
	return globalBufferPool.Stats()
}
