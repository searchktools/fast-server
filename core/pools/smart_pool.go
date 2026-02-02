package pools

import (
	"sync"
	"sync/atomic"
	"time"
)

// SmartPool is a dynamically-sized object pool with warmup and statistics
type SmartPool struct {
	pool      sync.Pool
	newFunc   func() any
	resetFunc func(any)

	// Statistics
	gets      atomic.Uint64
	puts      atomic.Uint64
	news      atomic.Uint64
	startTime time.Time

	// Configuration
	warmupSize    int
	maxIdleSize   int
	targetHitRate float64
}

// SmartPoolConfig configures a smart pool
type SmartPoolConfig struct {
	New           func() any
	Reset         func(any)
	WarmupSize    int     // Number of objects to pre-allocate
	MaxIdleSize   int     // Maximum idle objects to keep
	TargetHitRate float64 // Target cache hit rate (0.0-1.0)
}

// NewSmartPool creates a new smart pool with configuration
func NewSmartPool(config SmartPoolConfig) *SmartPool {
	if config.WarmupSize == 0 {
		config.WarmupSize = 100
	}
	if config.MaxIdleSize == 0 {
		config.MaxIdleSize = 1000
	}
	if config.TargetHitRate == 0 {
		config.TargetHitRate = 0.90
	}

	sp := &SmartPool{
		newFunc:       config.New,
		resetFunc:     config.Reset,
		warmupSize:    config.WarmupSize,
		maxIdleSize:   config.MaxIdleSize,
		targetHitRate: config.TargetHitRate,
		startTime:     time.Now(),
	}

	sp.pool.New = func() any {
		sp.news.Add(1)
		return config.New()
	}

	// Warmup: pre-allocate objects
	sp.Warmup()

	return sp
}

// Get acquires an object from the pool
func (sp *SmartPool) Get() any {
	sp.gets.Add(1)
	return sp.pool.Get()
}

// Put returns an object to the pool
func (sp *SmartPool) Put(obj any) {
	if obj == nil {
		return
	}

	sp.puts.Add(1)

	// Reset object state
	if sp.resetFunc != nil {
		sp.resetFunc(obj)
	}

	sp.pool.Put(obj)
}

// Warmup pre-allocates objects in the pool
func (sp *SmartPool) Warmup() {
	for i := 0; i < sp.warmupSize; i++ {
		obj := sp.newFunc()
		sp.pool.Put(obj)
	}
}

// Stats returns pool statistics
func (sp *SmartPool) Stats() SmartPoolStats {
	gets := sp.gets.Load()
	puts := sp.puts.Load()
	news := sp.news.Load()

	hitRate := 0.0
	if gets > 0 {
		// Hit rate = (gets - news) / gets
		// Objects served from pool vs newly created
		hits := gets - news
		if hits > 0 {
			hitRate = float64(hits) / float64(gets)
		}
	}

	return SmartPoolStats{
		Gets:      gets,
		Puts:      puts,
		News:      news,
		HitRate:   hitRate,
		Uptime:    time.Since(sp.startTime),
		ReuseRate: float64(puts) / float64(gets+1), // Avoid division by zero
	}
}

// SmartPoolStats contains smart pool statistics
type SmartPoolStats struct {
	Gets      uint64
	Puts      uint64
	News      uint64
	HitRate   float64
	Uptime    time.Duration
	ReuseRate float64
}

// Optimize adjusts pool behavior based on statistics
func (sp *SmartPool) Optimize() {
	stats := sp.Stats()

	// If hit rate is below target, consider warming up more
	if stats.HitRate < sp.targetHitRate && stats.Gets > 1000 {
		// Warmup additional objects
		additionalWarmup := sp.warmupSize / 10 // 10% increase
		for i := 0; i < additionalWarmup; i++ {
			obj := sp.newFunc()
			sp.pool.Put(obj)
		}
	}
}

// StartAutoOptimize starts automatic optimization in background
func (sp *SmartPool) StartAutoOptimize(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			sp.Optimize()
		}
	}()
}
