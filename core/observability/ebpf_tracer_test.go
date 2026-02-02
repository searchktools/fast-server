package observability

import (
	"testing"
	"time"
)

func TestEBPFTracer(t *testing.T) {
	tracer := NewEBPFTracer()

	// Trace syscalls
	tracer.TraceSystemCall("read", 10*time.Microsecond, nil)
	tracer.TraceSystemCall("write", 20*time.Microsecond, nil)
	tracer.TraceSystemCall("read", 15*time.Microsecond, nil)

	stats := tracer.GetSyscallStats()

	if len(stats) != 2 {
		t.Errorf("Expected 2 syscalls, got %d", len(stats))
	}

	readStats, ok := stats["read"]
	if !ok {
		t.Fatal("read syscall not found")
	}

	if readStats.Count != 2 {
		t.Errorf("Expected 2 read calls, got %d", readStats.Count)
	}
}

func TestNetworkTracing(t *testing.T) {
	tracer := NewEBPFTracer()

	tracer.TraceNetwork("tcp", 1024, 2048, true)
	tracer.TraceNetwork("tcp", 512, 1024, false)

	stats := tracer.GetNetworkStats()

	tcpStats, ok := stats["tcp"]
	if !ok {
		t.Fatal("TCP stats not found")
	}

	if tcpStats.BytesSent != 1536 {
		t.Errorf("Expected 1536 bytes sent, got %d", tcpStats.BytesSent)
	}

	if tcpStats.Connections != 1 {
		t.Errorf("Expected 1 connection, got %d", tcpStats.Connections)
	}
}

func BenchmarkTraceSyscall(b *testing.B) {
	tracer := NewEBPFTracer()
	duration := 10 * time.Microsecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracer.TraceSystemCall("read", duration, nil)
	}
}
