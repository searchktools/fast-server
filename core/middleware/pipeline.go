package middleware

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/searchktools/fast-server/core/http"
)

// HandlerFunc is the signature for middleware handlers
// Uses FDContext for zero-allocation performance
type HandlerFunc func(*http.FDContext)

// Pipeline is a zero-allocation middleware pipeline
type Pipeline struct {
	handlers []HandlerFunc
	length   int
}

// NewPipeline creates a new middleware pipeline
func NewPipeline() *Pipeline {
	return &Pipeline{
		handlers: make([]HandlerFunc, 0, 16), // Pre-allocate for 16 middlewares
	}
}

// Use adds a middleware to the pipeline
func (p *Pipeline) Use(handler HandlerFunc) *Pipeline {
	p.handlers = append(p.handlers, handler)
	p.length = len(p.handlers)
	return p
}

// Execute runs the middleware pipeline
// Zero-allocation: no interface{}, no closures, direct function calls
func (p *Pipeline) Execute(ctx *http.FDContext, finalHandler HandlerFunc) {
	// Fast path: no middlewares
	if p.length == 0 {
		finalHandler(ctx)
		return
	}

	// Execute middlewares in order
	for i := 0; i < p.length; i++ {
		p.handlers[i](ctx)

		// Check if aborted (fast path)
		if ctx.IsAborted() {
			return // Skip remaining middlewares
		}
	}

	// Execute final handler if not aborted
	if !ctx.IsAborted() {
		finalHandler(ctx)
	}
}

// Compile pre-compiles the pipeline for better performance
func (p *Pipeline) Compile() *Pipeline {
	if p.length <= 1 {
		return p
	}

	// Create new slice with exact size
	compiled := make([]HandlerFunc, p.length)
	copy(compiled, p.handlers)
	p.handlers = compiled

	return p
}

// AsyncPipeline provides async middleware execution
type AsyncPipeline struct {
	sync     *Pipeline
	async    []AsyncHandlerFunc
	pool     *sync.Pool
	workerCh chan asyncTask
}

// AsyncHandlerFunc is a middleware that runs asynchronously
type AsyncHandlerFunc func(*http.FDContext)

type asyncTask struct {
	handler AsyncHandlerFunc
	ctx     *http.FDContext
}

// NewAsyncPipeline creates a pipeline with async support
func NewAsyncPipeline(workers int) *AsyncPipeline {
	if workers <= 0 {
		workers = 4
	}

	p := &AsyncPipeline{
		sync:     NewPipeline(),
		async:    make([]AsyncHandlerFunc, 0, 8),
		workerCh: make(chan asyncTask, 256),
		pool: &sync.Pool{
			New: func() interface{} {
				return &asyncTask{}
			},
		},
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		go p.worker()
	}

	return p
}

// worker processes async tasks
func (p *AsyncPipeline) worker() {
	for task := range p.workerCh {
		task.handler(task.ctx)
		p.pool.Put(&task)
	}
}

// UseSync adds a synchronous middleware
func (p *AsyncPipeline) UseSync(handler HandlerFunc) *AsyncPipeline {
	p.sync.Use(handler)
	return p
}

// UseAsync adds an asynchronous middleware
func (p *AsyncPipeline) UseAsync(handler AsyncHandlerFunc) *AsyncPipeline {
	p.async = append(p.async, handler)
	return p
}

// Execute runs both sync and async middlewares
func (p *AsyncPipeline) Execute(ctx *http.FDContext, finalHandler HandlerFunc) {
	// Execute sync middlewares (blocking)
	p.sync.Execute(ctx, finalHandler)

	// Execute async middlewares (non-blocking)
	if !ctx.IsAborted() {
		for _, handler := range p.async {
			task := p.pool.Get().(*asyncTask)
			task.handler = handler
			task.ctx = ctx

			select {
			case p.workerCh <- *task:
			// Submitted
			default:
				// Channel full, execute inline
				handler(ctx)
				p.pool.Put(task)
			}
		}
	}
}

// Common middleware implementations

// Recovery recovers from panics
func Recovery() HandlerFunc {
	return func(ctx *http.FDContext) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				ctx.Abort()
				ctx.JSON(500, map[string]interface{}{
					"error": "Internal Server Error",
				})
			}
		}()
	}
}

// Logger logs requests (async)
func Logger() AsyncHandlerFunc {
	return func(ctx *http.FDContext) {
		method := ctx.Method()
		path := ctx.Path()
		log.Printf("[%s] %s", method, path)
	}
}

// CORS adds CORS headers
func CORS() HandlerFunc {
	return func(ctx *http.FDContext) {
		ctx.SetHeader("Access-Control-Allow-Origin", "*")
		ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if ctx.Method() == "OPTIONS" {
			ctx.Abort()
			ctx.Status(204)
		}
	}
}

// RateLimiter implements rate limiting
func RateLimiter(requestsPerSecond int) HandlerFunc {
	var (
		tokens     int
		lastRefill time.Time
		mu         sync.Mutex
	)

	tokens = requestsPerSecond
	lastRefill = time.Now()

	return func(ctx *http.FDContext) {
		mu.Lock()

		now := time.Now()
		elapsed := now.Sub(lastRefill)
		if elapsed > time.Second {
			tokens = requestsPerSecond
			lastRefill = now
		}

		if tokens > 0 {
			tokens--
			mu.Unlock()
			return
		}

		mu.Unlock()

		ctx.Abort()
		ctx.Status(429)
		ctx.JSON(429, map[string]interface{}{
			"error": "Too Many Requests",
		})
	}
}

// RequestID adds a unique request ID
func RequestID() HandlerFunc {
	var counter uint64

	return func(ctx *http.FDContext) {
		id := atomic.AddUint64(&counter, 1)
		ctx.SetHeader("X-Request-ID", fmt.Sprintf("%d", id))
	}
}

// Metrics collects request metrics (async)
func Metrics() AsyncHandlerFunc {
	return func(ctx *http.FDContext) {
		// Collect metrics without blocking
		method := ctx.Method()
		path := ctx.Path()
		_ = method
		_ = path
	}
}
