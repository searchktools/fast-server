package pools

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Task represents a unit of work
type Task func()

// WorkerPool implements a work-stealing goroutine pool
type WorkerPool struct {
	numWorkers int
	queues     []*workerQueue
	workers    []*worker
	closed     atomic.Bool

	// Statistics
	stats struct {
		tasksSubmitted atomic.Uint64
		tasksCompleted atomic.Uint64
		stealsSuccess  atomic.Uint64
		stealsFailed   atomic.Uint64
	}
}

// workerQueue is a lock-free queue for a single worker
type workerQueue struct {
	tasks chan Task
	id    int
}

// worker represents a goroutine that processes tasks
type worker struct {
	id       int
	pool     *WorkerPool
	queue    *workerQueue
	stopping atomic.Bool
}

// NewWorkerPool creates a new work-stealing worker pool
func NewWorkerPool(numWorkers int) *WorkerPool {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	pool := &WorkerPool{
		numWorkers: numWorkers,
		queues:     make([]*workerQueue, numWorkers),
		workers:    make([]*worker, numWorkers),
	}

	// Create worker queues
	for i := 0; i < numWorkers; i++ {
		pool.queues[i] = &workerQueue{
			tasks: make(chan Task, 256), // Buffered channel for each worker
			id:    i,
		}
	}

	// Create and start workers
	for i := 0; i < numWorkers; i++ {
		w := &worker{
			id:    i,
			pool:  pool,
			queue: pool.queues[i],
		}
		pool.workers[i] = w
		go w.run()
	}

	return pool
}

// Submit submits a task to the pool using round-robin
func (p *WorkerPool) Submit(task Task) bool {
	if p.closed.Load() {
		return false
	}

	p.stats.tasksSubmitted.Add(1)

	// Round-robin distribution based on task count
	idx := int(p.stats.tasksSubmitted.Load()) % p.numWorkers

	select {
	case p.queues[idx].tasks <- task:
		return true
	default:
		// Queue full, try next worker
		idx = (idx + 1) % p.numWorkers
		select {
		case p.queues[idx].tasks <- task:
			return true
		default:
			// All queues full, execute inline
			task()
			p.stats.tasksCompleted.Add(1)
			return true
		}
	}
}

// worker.run is the main loop for a worker goroutine
func (w *worker) run() {
	// Pin this goroutine to a specific P if possible
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		// Try to get task from own queue first
		select {
		case task := <-w.queue.tasks:
			if task == nil {
				return // Shutdown signal
			}
			task()
			w.pool.stats.tasksCompleted.Add(1)
			continue
		default:
		}

		// Own queue is empty, try to steal from other workers
		if w.trySteal() {
			continue
		}

		// No work available, block on own queue
		task, ok := <-w.queue.tasks
		if !ok || task == nil {
			return // Channel closed or shutdown
		}

		task()
		w.pool.stats.tasksCompleted.Add(1)
	}
}

// trySteal attempts to steal work from another worker
func (w *worker) trySteal() bool {
	numWorkers := w.pool.numWorkers

	// Try to steal from (numWorkers - 1) other workers
	// Start from a pseudo-random position to avoid contention
	start := (w.id + 1) % numWorkers

	for i := 0; i < numWorkers-1; i++ {
		victimIdx := (start + i) % numWorkers
		victim := w.pool.queues[victimIdx]

		select {
		case task := <-victim.tasks:
			if task != nil {
				// Successfully stole a task
				w.pool.stats.stealsSuccess.Add(1)
				task()
				w.pool.stats.tasksCompleted.Add(1)
				return true
			}
		default:
			// Victim queue is empty, try next
		}
	}

	w.pool.stats.stealsFailed.Add(1)
	return false
}

// Close gracefully shuts down the worker pool
func (p *WorkerPool) Close() {
	if !p.closed.CompareAndSwap(false, true) {
		return // Already closed
	}

	// Send shutdown signal to all workers
	for _, q := range p.queues {
		close(q.tasks)
	}
}

// Stats returns pool statistics
func (p *WorkerPool) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		NumWorkers:     p.numWorkers,
		TasksSubmitted: p.stats.tasksSubmitted.Load(),
		TasksCompleted: p.stats.tasksCompleted.Load(),
		TasksPending:   p.stats.tasksSubmitted.Load() - p.stats.tasksCompleted.Load(),
		StealsSuccess:  p.stats.stealsSuccess.Load(),
		StealsFailed:   p.stats.stealsFailed.Load(),
	}
}

// WorkerPoolStats contains pool statistics
type WorkerPoolStats struct {
	NumWorkers     int
	TasksSubmitted uint64
	TasksCompleted uint64
	TasksPending   uint64
	StealsSuccess  uint64
	StealsFailed   uint64
}

// Global worker pool instance
var (
	globalPool     *WorkerPool
	globalPoolOnce sync.Once
)

// GetGlobalPool returns the global worker pool
func GetGlobalPool() *WorkerPool {
	globalPoolOnce.Do(func() {
		globalPool = NewWorkerPool(runtime.NumCPU())
	})
	return globalPool
}

// SubmitTask submits a task to the global worker pool
func SubmitTask(task Task) bool {
	return GetGlobalPool().Submit(task)
}
