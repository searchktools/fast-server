package observability

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceMonitor provides zero-overhead performance monitoring
type PerformanceMonitor struct {
	enabled  atomic.Bool
	handlers sync.Map
	global   struct {
		totalRequests   atomic.Uint64
		totalDuration   atomic.Uint64
		totalCPUTime    atomic.Uint64
		totalAllocBytes atomic.Uint64
	}
	bottlenecks  []Bottleneck
	bottleneckMu sync.RWMutex
}

// HandlerMetrics stores per-handler metrics
type HandlerMetrics struct {
	Name           string
	Count          atomic.Uint64
	Errors         atomic.Uint64
	TotalDuration  atomic.Uint64
	MinDuration    atomic.Uint64
	MaxDuration    atomic.Uint64
	CPUTime        atomic.Uint64
	AllocBytes     atomic.Uint64
	latencyBuckets [10]atomic.Uint64
}

// Bottleneck represents a performance issue
type Bottleneck struct {
	Type       string
	Location   string
	Severity   int
	Impact     float64
	DetectedAt time.Time
	Details    string
}

// NewPerformanceMonitor creates a monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	pm := &PerformanceMonitor{}
	pm.enabled.Store(true)
	go pm.analyzeBottlenecks()
	return pm
}

// RecordRequest records a request
func (pm *PerformanceMonitor) RecordRequest(handler string, duration time.Duration, isError bool) {
	if !pm.enabled.Load() {
		return
	}

	val, _ := pm.handlers.LoadOrStore(handler, &HandlerMetrics{Name: handler})
	metrics := val.(*HandlerMetrics)

	metrics.Count.Add(1)
	if isError {
		metrics.Errors.Add(1)
	}

	durationNs := uint64(duration.Nanoseconds())
	metrics.TotalDuration.Add(durationNs)
	pm.updateMinMax(metrics, durationNs)
	pm.updateLatencyBucket(metrics, durationNs)

	pm.global.totalRequests.Add(1)
	pm.global.totalDuration.Add(durationNs)
}

func (pm *PerformanceMonitor) updateMinMax(m *HandlerMetrics, d uint64) {
	for {
		min := m.MinDuration.Load()
		if min == 0 || d < min {
			if m.MinDuration.CompareAndSwap(min, d) {
				break
			}
		} else {
			break
		}
	}
	for {
		max := m.MaxDuration.Load()
		if d > max {
			if m.MaxDuration.CompareAndSwap(max, d) {
				break
			}
		} else {
			break
		}
	}
}

func (pm *PerformanceMonitor) updateLatencyBucket(m *HandlerMetrics, durationNs uint64) {
	ms := durationNs / 1_000_000
	idx := 0
	switch {
	case ms < 1:
		idx = 0
	case ms < 5:
		idx = 1
	case ms < 10:
		idx = 2
	case ms < 50:
		idx = 3
	case ms < 100:
		idx = 4
	case ms < 500:
		idx = 5
	case ms < 1000:
		idx = 6
	case ms < 5000:
		idx = 7
	case ms < 10000:
		idx = 8
	default:
		idx = 9
	}
	m.latencyBuckets[idx].Add(1)
}

func (pm *PerformanceMonitor) analyzeBottlenecks() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !pm.enabled.Load() {
			continue
		}
		bottlenecks := pm.detectBottlenecks()
		pm.bottleneckMu.Lock()
		pm.bottlenecks = bottlenecks
		pm.bottleneckMu.Unlock()
	}
}

func (pm *PerformanceMonitor) detectBottlenecks() []Bottleneck {
	bottlenecks := make([]Bottleneck, 0)

	pm.handlers.Range(func(key, value interface{}) bool {
		m := value.(*HandlerMetrics)
		count := m.Count.Load()
		if count == 0 {
			return true
		}

		avgDuration := time.Duration(m.TotalDuration.Load() / count)

		// High latency
		if avgDuration > 100*time.Millisecond {
			bottlenecks = append(bottlenecks, Bottleneck{
				Type:       "latency",
				Location:   m.Name,
				Severity:   8,
				Impact:     100.0,
				DetectedAt: time.Now(),
				Details:    fmt.Sprintf("High latency (%v avg)", avgDuration),
			})
		}

		// High error rate
		errors := m.Errors.Load()
		if errors > 0 && float64(errors)/float64(count) > 0.05 {
			bottlenecks = append(bottlenecks, Bottleneck{
				Type:       "errors",
				Location:   m.Name,
				Severity:   10,
				Impact:     float64(errors) / float64(count) * 100,
				DetectedAt: time.Now(),
				Details:    fmt.Sprintf("%.1f%% error rate", float64(errors)/float64(count)*100),
			})
		}

		return true
	})

	return bottlenecks
}

// GetBottlenecks returns detected bottlenecks
func (pm *PerformanceMonitor) GetBottlenecks() []Bottleneck {
	pm.bottleneckMu.RLock()
	defer pm.bottleneckMu.RUnlock()
	return append([]Bottleneck{}, pm.bottlenecks...)
}

// StartTrace starts timing
func (pm *PerformanceMonitor) StartTrace() int64 {
	if !pm.enabled.Load() {
		return 0
	}
	return time.Now().UnixNano()
}

// EndTrace ends timing and records
func (pm *PerformanceMonitor) EndTrace(handler string, startTime int64, isError bool) {
	if startTime == 0 {
		return
	}
	duration := time.Duration(time.Now().UnixNano() - startTime)
	pm.RecordRequest(handler, duration, isError)
}
