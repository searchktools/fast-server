package observability

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// EBPFTracer simulates eBPF-style kernel-level tracing
// In production, this would use cilium/ebpf or libbpf-go
type EBPFTracer struct {
	enabled   atomic.Bool
	events    chan *TraceEvent
	eventPool sync.Pool

	// Aggregated metrics
	syscalls     sync.Map // map[string]*SyscallStats
	networkStats sync.Map // map[string]*NetworkStats
	lockStats    sync.Map // map[string]*LockStats

	// Sampling
	sampleRate atomic.Uint32 // 0-100 percentage
}

// TraceEvent represents a kernel-level event
type TraceEvent struct {
	Timestamp  int64  // nanoseconds
	EventType  string // "syscall", "network", "lock", "alloc"
	PID        int
	TID        int
	Handler    string
	Duration   int64 // nanoseconds
	BytesCount uint64
	Details    string
}

// SyscallStats tracks syscall performance
type SyscallStats struct {
	Name      string
	Count     atomic.Uint64
	TotalTime atomic.Uint64 // nanoseconds
	MinTime   atomic.Uint64
	MaxTime   atomic.Uint64
	Errors    atomic.Uint64
}

// NetworkStats tracks network I/O
type NetworkStats struct {
	Protocol    string // "tcp", "udp"
	BytesSent   atomic.Uint64
	BytesRecv   atomic.Uint64
	Connections atomic.Uint64
	Retransmits atomic.Uint64
	Errors      atomic.Uint64
}

// LockStats tracks lock contention
type LockStats struct {
	Location  string
	Waits     atomic.Uint64
	WaitTime  atomic.Uint64 // total wait time in nanoseconds
	HoldTime  atomic.Uint64 // total hold time in nanoseconds
	Conflicts atomic.Uint64
}

// NewEBPFTracer creates a new tracer
func NewEBPFTracer() *EBPFTracer {
	tracer := &EBPFTracer{
		events: make(chan *TraceEvent, 10000),
		eventPool: sync.Pool{
			New: func() interface{} {
				return &TraceEvent{}
			},
		},
	}

	tracer.enabled.Store(true)
	tracer.sampleRate.Store(100) // 100% sampling by default

	// Start event processor
	go tracer.processEvents()

	return tracer
}

// TraceSystemCall records a syscall event
func (t *EBPFTracer) TraceSystemCall(name string, duration time.Duration, err error) {
	if !t.enabled.Load() {
		return
	}

	// Fast path aggregation (no event allocation)
	val, _ := t.syscalls.LoadOrStore(name, &SyscallStats{Name: name})
	stats := val.(*SyscallStats)

	stats.Count.Add(1)
	durationNs := uint64(duration.Nanoseconds())
	stats.TotalTime.Add(durationNs)

	if err != nil {
		stats.Errors.Add(1)
	}

	// Update min/max
	t.updateSyscallMinMax(stats, durationNs)
}

func (t *EBPFTracer) updateSyscallMinMax(stats *SyscallStats, duration uint64) {
	// Update min
	for {
		min := stats.MinTime.Load()
		if min == 0 || duration < min {
			if stats.MinTime.CompareAndSwap(min, duration) {
				break
			}
		} else {
			break
		}
	}

	// Update max
	for {
		max := stats.MaxTime.Load()
		if duration > max {
			if stats.MaxTime.CompareAndSwap(max, duration) {
				break
			}
		} else {
			break
		}
	}
}

// TraceNetwork records network I/O
func (t *EBPFTracer) TraceNetwork(protocol string, bytesSent, bytesRecv uint64, isNewConn bool) {
	if !t.enabled.Load() {
		return
	}

	val, _ := t.networkStats.LoadOrStore(protocol, &NetworkStats{Protocol: protocol})
	stats := val.(*NetworkStats)

	stats.BytesSent.Add(bytesSent)
	stats.BytesRecv.Add(bytesRecv)

	if isNewConn {
		stats.Connections.Add(1)
	}
}

// TraceLock records lock contention
func (t *EBPFTracer) TraceLock(location string, waitTime, holdTime time.Duration, wasContended bool) {
	if !t.enabled.Load() {
		return
	}

	val, _ := t.lockStats.LoadOrStore(location, &LockStats{Location: location})
	stats := val.(*LockStats)

	stats.Waits.Add(1)
	stats.WaitTime.Add(uint64(waitTime.Nanoseconds()))
	stats.HoldTime.Add(uint64(holdTime.Nanoseconds()))

	if wasContended {
		stats.Conflicts.Add(1)
	}
}

// processEvents processes trace events in background
func (t *EBPFTracer) processEvents() {
	for event := range t.events {
		// Process event based on type
		switch event.EventType {
		case "syscall":
			// Already handled in TraceSystemCall
		case "network":
			// Already handled in TraceNetwork
		case "lock":
			// Already handled in TraceLock
		}

		// Return event to pool
		t.eventPool.Put(event)
	}
}

// GetSyscallStats returns syscall statistics
func (t *EBPFTracer) GetSyscallStats() map[string]SyscallSnapshot {
	result := make(map[string]SyscallSnapshot)

	t.syscalls.Range(func(key, value interface{}) bool {
		name := key.(string)
		stats := value.(*SyscallStats)
		count := stats.Count.Load()

		if count > 0 {
			result[name] = SyscallSnapshot{
				Name:    name,
				Count:   count,
				AvgTime: time.Duration(stats.TotalTime.Load() / count),
				MinTime: time.Duration(stats.MinTime.Load()),
				MaxTime: time.Duration(stats.MaxTime.Load()),
				Errors:  stats.Errors.Load(),
			}
		}
		return true
	})

	return result
}

// GetNetworkStats returns network statistics
func (t *EBPFTracer) GetNetworkStats() map[string]NetworkSnapshot {
	result := make(map[string]NetworkSnapshot)

	t.networkStats.Range(func(key, value interface{}) bool {
		protocol := key.(string)
		stats := value.(*NetworkStats)

		result[protocol] = NetworkSnapshot{
			Protocol:    protocol,
			BytesSent:   stats.BytesSent.Load(),
			BytesRecv:   stats.BytesRecv.Load(),
			Connections: stats.Connections.Load(),
			Retransmits: stats.Retransmits.Load(),
			Errors:      stats.Errors.Load(),
		}
		return true
	})

	return result
}

// GetLockStats returns lock contention statistics
func (t *EBPFTracer) GetLockStats() map[string]LockSnapshot {
	result := make(map[string]LockSnapshot)

	t.lockStats.Range(func(key, value interface{}) bool {
		location := key.(string)
		stats := value.(*LockStats)
		waits := stats.Waits.Load()

		if waits > 0 {
			result[location] = LockSnapshot{
				Location:       location,
				Waits:          waits,
				AvgWaitTime:    time.Duration(stats.WaitTime.Load() / waits),
				AvgHoldTime:    time.Duration(stats.HoldTime.Load() / waits),
				Conflicts:      stats.Conflicts.Load(),
				ContentionRate: float64(stats.Conflicts.Load()) / float64(waits) * 100,
			}
		}
		return true
	})

	return result
}

// Enable enables tracing
func (t *EBPFTracer) Enable() {
	t.enabled.Store(true)
}

// Disable disables tracing
func (t *EBPFTracer) Disable() {
	t.enabled.Store(false)
}

// SetSampleRate sets sampling rate (0-100%)
func (t *EBPFTracer) SetSampleRate(rate uint32) {
	if rate > 100 {
		rate = 100
	}
	t.sampleRate.Store(rate)
}

// Snapshot types
type SyscallSnapshot struct {
	Name    string
	Count   uint64
	AvgTime time.Duration
	MinTime time.Duration
	MaxTime time.Duration
	Errors  uint64
}

type NetworkSnapshot struct {
	Protocol    string
	BytesSent   uint64
	BytesRecv   uint64
	Connections uint64
	Retransmits uint64
	Errors      uint64
}

type LockSnapshot struct {
	Location       string
	Waits          uint64
	AvgWaitTime    time.Duration
	AvgHoldTime    time.Duration
	Conflicts      uint64
	ContentionRate float64
}

// Report generates a human-readable report
func (t *EBPFTracer) Report() string {
	report := "═══════════════════════════════════════\n"
	report += "eBPF Performance Trace Report\n"
	report += "═══════════════════════════════════════\n\n"

	// Syscall stats
	report += "📊 System Calls:\n"
	syscalls := t.GetSyscallStats()
	if len(syscalls) == 0 {
		report += "  No data\n"
	}
	for name, stats := range syscalls {
		report += fmt.Sprintf("  • %s: %d calls, avg=%v, min=%v, max=%v, errors=%d\n",
			name, stats.Count, stats.AvgTime, stats.MinTime, stats.MaxTime, stats.Errors)
	}

	// Network stats
	report += "\n🌐 Network I/O:\n"
	network := t.GetNetworkStats()
	if len(network) == 0 {
		report += "  No data\n"
	}
	for protocol, stats := range network {
		report += fmt.Sprintf("  • %s: %d conns, sent=%d MB, recv=%d MB, errors=%d\n",
			protocol, stats.Connections,
			stats.BytesSent/(1024*1024), stats.BytesRecv/(1024*1024),
			stats.Errors)
	}

	// Lock stats
	report += "\n🔒 Lock Contention:\n"
	locks := t.GetLockStats()
	if len(locks) == 0 {
		report += "  No data\n"
	}
	for location, stats := range locks {
		report += fmt.Sprintf("  • %s: %d waits, avg_wait=%v, conflicts=%d (%.1f%%)\n",
			location, stats.Waits, stats.AvgWaitTime,
			stats.Conflicts, stats.ContentionRate)
	}

	report += "\n═══════════════════════════════════════\n"
	return report
}
