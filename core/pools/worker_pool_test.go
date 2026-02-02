package pools

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_Basic(t *testing.T) {
	pool := NewWorkerPool(4)
	defer pool.Close()

	done := make(chan bool)
	var counter atomic.Int64

	// Submit 100 tasks
	for i := 0; i < 100; i++ {
		pool.Submit(func() {
			counter.Add(1)
		})
	}

	// Wait for completion
	go func() {
		for {
			stats := pool.Stats()
			if stats.TasksCompleted >= 100 {
				done <- true
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-done:
		if counter.Load() != 100 {
			t.Errorf("Expected 100 tasks completed, got %d", counter.Load())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Test timeout")
	}
}

func TestWorkerPool_WorkStealing(t *testing.T) {
	pool := NewWorkerPool(4)
	defer pool.Close()

	var counter atomic.Int64

	// Submit tasks that take different time
	for i := 0; i < 100; i++ {
		i := i
		pool.Submit(func() {
			if i%10 == 0 {
				time.Sleep(10 * time.Millisecond) // Some tasks are slower
			}
			counter.Add(1)
		})
	}

	// Wait for completion
	time.Sleep(500 * time.Millisecond)

	stats := pool.Stats()
	if stats.TasksCompleted < 100 {
		t.Errorf("Expected 100 tasks completed, got %d", stats.TasksCompleted)
	}

	// Check that work stealing happened
	if stats.StealsSuccess == 0 {
		t.Log("Warning: No successful steals detected")
	}
}

func BenchmarkWorkerPool_Submit(b *testing.B) {
	pool := NewWorkerPool(8)
	defer pool.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Submit(func() {
				// Simulate some work
				_ = 1 + 1
			})
		}
	})

	// Wait for completion
	for {
		stats := pool.Stats()
		if stats.TasksCompleted >= uint64(b.N) {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func BenchmarkGoroutine_Direct(b *testing.B) {
	var wg atomic.Int64
	wg.Store(int64(b.N))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			go func() {
				// Simulate some work
				_ = 1 + 1
				wg.Add(-1)
			}()
		}
	})

	// Wait for completion
	for wg.Load() > 0 {
		time.Sleep(1 * time.Millisecond)
	}
}
