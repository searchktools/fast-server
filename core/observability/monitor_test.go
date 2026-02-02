package observability

import (
	"testing"
	"time"
)

func TestPerformanceMonitor(t *testing.T) {
	pm := NewPerformanceMonitor()

	// Record some requests
	pm.RecordRequest("GET /api", 10*time.Millisecond, false)
	pm.RecordRequest("GET /api", 20*time.Millisecond, false)
	pm.RecordRequest("GET /api", 30*time.Millisecond, false)

	// Check metrics
	val, ok := pm.handlers.Load("GET /api")
	if !ok {
		t.Fatal("Handler metrics not found")
	}

	metrics := val.(*HandlerMetrics)
	if count := metrics.Count.Load(); count != 3 {
		t.Errorf("Expected 3 requests, got %d", count)
	}

	avgDuration := time.Duration(metrics.TotalDuration.Load() / metrics.Count.Load())
	if avgDuration != 20*time.Millisecond {
		t.Errorf("Expected 20ms avg, got %v", avgDuration)
	}
}

func TestBottleneckDetection(t *testing.T) {
	pm := NewPerformanceMonitor()

	// Simulate slow handler
	for i := 0; i < 100; i++ {
		pm.RecordRequest("GET /slow", 150*time.Millisecond, false)
	}

	// Manually trigger detection
	bottlenecks := pm.detectBottlenecks()

	if len(bottlenecks) == 0 {
		t.Error("Expected bottleneck detection for slow handler")
	} else {
		t.Logf("✅ Detected %d bottlenecks", len(bottlenecks))
		for _, b := range bottlenecks {
			t.Logf("  - [%s] %s: %s (severity: %d)", b.Type, b.Location, b.Details, b.Severity)
		}
	}
}

func BenchmarkRecordRequest(b *testing.B) {
	pm := NewPerformanceMonitor()
	duration := 10 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.RecordRequest("GET /api", duration, false)
	}
}

func BenchmarkTraceHandler(b *testing.B) {
	pm := NewPerformanceMonitor()

	handler := func() error {
		// Simulate work
		time.Sleep(10 * time.Microsecond)
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		startTime := pm.StartTrace()
		handler()
		pm.EndTrace("GET /api", startTime, false)
	}
}
