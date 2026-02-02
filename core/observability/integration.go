package observability

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
)

// Observatory is the central observability hub
type Observatory struct {
	Monitor *PerformanceMonitor
	Tracer  *EBPFTracer
	enabled bool
}

// NewObservatory creates a new observatory
func NewObservatory() *Observatory {
	return &Observatory{
		Monitor: NewPerformanceMonitor(),
		Tracer:  NewEBPFTracer(),
		enabled: true,
	}
}

// TraceHandler wraps a handler with full observability
func (o *Observatory) TraceHandler(name string, fn func() error) error {
	if !o.enabled {
		return fn()
	}

	// Start tracing
	startTime := o.Monitor.StartTrace()
	startMem := getMemStats()

	// Execute handler
	err := fn()

	// End tracing
	endMem := getMemStats()
	o.Monitor.EndTrace(name, startTime, err != nil)

	// Record memory allocation
	if endMem > startMem {
		allocBytes := endMem - startMem
		o.Tracer.TraceSystemCall("malloc", time.Since(time.Unix(0, startTime)), nil)
		_ = allocBytes // Would be recorded in production
	}

	return err
}

// TraceSyscall traces a syscall with timing
func (o *Observatory) TraceSyscall(name string, fn func() error) error {
	if !o.enabled {
		return fn()
	}

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	o.Tracer.TraceSystemCall(name, duration, err)
	return err
}

// TraceNetworkIO traces network I/O
func (o *Observatory) TraceNetworkIO(protocol string, fd int, op string) func(n int, err error) {
	if !o.enabled {
		return func(n int, err error) {}
	}

	start := time.Now()

	return func(n int, err error) {
		duration := time.Since(start)

		// Record network stats
		if op == "write" && n > 0 {
			o.Tracer.TraceNetwork(protocol, uint64(n), 0, false)
		} else if op == "read" && n > 0 {
			o.Tracer.TraceNetwork(protocol, 0, uint64(n), false)
		}

		// Record syscall
		syscallName := fmt.Sprintf("syscall.%s", op)
		o.Tracer.TraceSystemCall(syscallName, duration, err)
	}
}

// GetFullReport generates a comprehensive report
func (o *Observatory) GetFullReport() string {
	report := "╔═══════════════════════════════════════════════════╗\n"
	report += "║     High-Performance Server Observatory          ║\n"
	report += "╚═══════════════════════════════════════════════════╝\n\n"

	// Handler performance
	report += "📊 Handler Performance:\n"
	bottlenecks := o.Monitor.GetBottlenecks()
	if len(bottlenecks) == 0 {
		report += "  ✅ No bottlenecks detected\n"
	} else {
		report += fmt.Sprintf("  ⚠️  %d bottlenecks detected:\n", len(bottlenecks))
		for i, b := range bottlenecks {
			report += fmt.Sprintf("    %d. [%s] %s - %s (severity: %d/10)\n",
				i+1, b.Type, b.Location, b.Details, b.Severity)
		}
	}
	report += "\n"

	// eBPF trace data
	report += o.Tracer.Report()

	// System metrics
	report += "\n💻 System Metrics:\n"
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	report += fmt.Sprintf("  • Heap Alloc: %d MB\n", m.HeapAlloc/(1024*1024))
	report += fmt.Sprintf("  • Heap Objects: %d\n", m.HeapObjects)
	report += fmt.Sprintf("  • GC Runs: %d\n", m.NumGC)
	report += fmt.Sprintf("  • Goroutines: %d\n", runtime.NumGoroutine())

	return report
}

// Enable enables all observability
func (o *Observatory) Enable() {
	o.enabled = true
	o.Monitor.enabled.Store(true)
	o.Tracer.Enable()
}

// Disable disables all observability
func (o *Observatory) Disable() {
	o.enabled = false
	o.Monitor.enabled.Store(false)
	o.Tracer.Disable()
}

// Helper functions

func getMemStats() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.TotalAlloc
}

// WrapSyscallWrite wraps syscall.Write with tracing
func (o *Observatory) WrapSyscallWrite(fd int, p []byte) (n int, err error) {
	if !o.enabled {
		return syscall.Write(fd, p)
	}

	callback := o.TraceNetworkIO("tcp", fd, "write")
	n, err = syscall.Write(fd, p)
	callback(n, err)
	return
}

// WrapSyscallRead wraps syscall.Read with tracing
func (o *Observatory) WrapSyscallRead(fd int, p []byte) (n int, err error) {
	if !o.enabled {
		return syscall.Read(fd, p)
	}

	callback := o.TraceNetworkIO("tcp", fd, "read")
	n, err = syscall.Read(fd, p)
	callback(n, err)
	return
}
