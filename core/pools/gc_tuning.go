package pools

import (
	"runtime"
	"runtime/debug"
	"time"
)

// GCConfig holds GC tuning parameters
type GCConfig struct {
	// GOGC sets the garbage collection target percentage
	// Default is 100. Lower values = more frequent GC but less memory
	GOGC int

	// MemoryLimit sets soft memory limit in bytes
	// 0 = no limit
	MemoryLimit int64

	// MinRetainExtra minimum extra memory to retain (helps reduce GC frequency)
	MinRetainExtra int64
}

// DefaultGCConfig returns optimized GC settings for high-performance servers
func DefaultGCConfig() GCConfig {
	return GCConfig{
		GOGC:           200,      // Less frequent GC (default 100)
		MemoryLimit:    0,        // No hard limit
		MinRetainExtra: 50 << 20, // Retain 50MB extra
	}
}

// ApplyGCConfig applies GC tuning to reduce GC pressure
func ApplyGCConfig(cfg GCConfig) {
	if cfg.GOGC > 0 {
		debug.SetGCPercent(cfg.GOGC)
	}

	if cfg.MemoryLimit > 0 {
		debug.SetMemoryLimit(cfg.MemoryLimit)
	}

	// Increase initial heap size to reduce early GC
	if cfg.MinRetainExtra > 0 {
		// Force a GC then immediately allocate to set baseline
		runtime.GC()
		_ = make([]byte, cfg.MinRetainExtra)
	}
}

// GCStats holds garbage collection statistics
type GCStats struct {
	NumGC        uint32
	PauseTotal   time.Duration
	LastPause    time.Duration
	AvgPause     time.Duration
	AllocBytes   uint64
	TotalAlloc   uint64
	Sys          uint64
	NumGoroutine int
}

// GetGCStats returns current GC statistics
func GetGCStats() GCStats {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	stats := GCStats{
		NumGC:        ms.NumGC,
		AllocBytes:   ms.Alloc,
		TotalAlloc:   ms.TotalAlloc,
		Sys:          ms.Sys,
		NumGoroutine: runtime.NumGoroutine(),
	}

	if ms.NumGC > 0 {
		stats.LastPause = time.Duration(ms.PauseNs[(ms.NumGC+255)%256])

		var totalPause uint64
		numPauses := ms.NumGC
		if numPauses > 256 {
			numPauses = 256
		}

		for i := uint32(0); i < numPauses; i++ {
			totalPause += ms.PauseNs[i]
		}

		stats.PauseTotal = time.Duration(totalPause)
		if numPauses > 0 {
			stats.AvgPause = time.Duration(totalPause / uint64(numPauses))
		}
	}

	return stats
}

// OptimizeForHighThroughput applies GC settings optimized for high RPS
func OptimizeForHighThroughput() {
	ApplyGCConfig(GCConfig{
		GOGC:           300,       // Very infrequent GC
		MinRetainExtra: 100 << 20, // 100MB baseline
	})
}

// OptimizeForLowLatency applies GC settings optimized for low latency
func OptimizeForLowLatency() {
	ApplyGCConfig(GCConfig{
		GOGC:           150,      // Moderate GC frequency
		MinRetainExtra: 30 << 20, // 30MB baseline
	})
}
