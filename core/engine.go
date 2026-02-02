package core

import (
	"log"
	"net"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/searchktools/fast-server/core/http"
	"github.com/searchktools/fast-server/core/poller"
	"github.com/searchktools/fast-server/core/pools"
	"github.com/searchktools/fast-server/core/router"
)

// HandlerFunc defines the handler function type (accepts http.Context interface)
type HandlerFunc func(ctx http.Context)

// Connection states
const (
	StateReading = iota
	StateProcessing
	StateWriting
	StateKeepalive
)

// Connection represents an active connection
type Connection struct {
	fd         int
	state      int
	readBuf    []byte
	readOffset int
	request    *http.Request
	context    *http.FDContext
	lastActive time.Time
	keepAlive  bool
	closeAfter bool
}

// Reset implements ConnectionPoolable interface
func (c *Connection) Reset() {
	c.fd = -1
	c.state = StateReading
	c.readBuf = nil
	c.readOffset = 0
	c.request = nil
	c.context = nil
	c.lastActive = time.Time{}
	c.keepAlive = false
	c.closeAfter = false
}

// SetFD implements ConnectionPoolable interface
func (c *Connection) SetFD(fd int) {
	c.fd = fd
	c.lastActive = time.Now()
}

// Engine is a high-performance zero-allocation HTTP engine with epoll/kqueue
type Engine struct {
	router      *router.RadixRouter
	poller      poller.Poller
	connections map[int]*Connection
	connMu      sync.RWMutex

	maxConnections int
	readTimeout    time.Duration
	writeTimeout   time.Duration
	idleTimeout    time.Duration

	// Fine-grained memory pools
	contextPool    *pools.SmartPool
	requestPool    *pools.SmartPool
	bytePool       *pools.BytePool
	connectionPool *pools.ConnectionPool
	workerPool     *pools.WorkerPool // Work-stealing goroutine pool
}

// NewEngine creates a new engine instance
func NewEngine() *Engine {
	e := &Engine{
		router:         router.NewRadixRouter(),
		connections:    make(map[int]*Connection, 10000),
		maxConnections: 100000,
		readTimeout:    10 * time.Second,
		writeTimeout:   10 * time.Second,
		idleTimeout:    5 * time.Second, // Short idle timeout for aggressive cleanup
	}

	// Apply GC optimizations for high throughput
	pools.OptimizeForHighThroughput()

	// Initialize fine-grained pools
	e.bytePool = pools.NewBytePool()

	// Connection pool
	e.connectionPool = pools.NewConnectionPool(10000, func() any {
		return &Connection{
			fd:    -1,
			state: StateReading,
		}
	})

	e.contextPool = pools.NewSmartPool(pools.SmartPoolConfig{
		New: func() any {
			return &http.FDContext{}
		},
		Reset: func(obj any) {
			if ctx, ok := obj.(*http.FDContext); ok {
				ctx.Reset(0, nil)
			}
		},
		WarmupSize:    500,  // Increased from 300
		TargetHitRate: 0.95, // Target 95% hit rate
	})

	e.requestPool = pools.NewSmartPool(pools.SmartPoolConfig{
		New: func() any {
			return &http.Request{}
		},
		Reset: func(obj any) {
			if req, ok := obj.(*http.Request); ok {
				// Simple reset without calling external function
				req.Method = ""
				req.Path = ""
				req.Proto = ""
				req.Body = req.Body[:0]
			}
		},
		WarmupSize:    500,
		TargetHitRate: 0.95,
	})

	// Start auto-optimization
	e.contextPool.StartAutoOptimize(30 * time.Second)
	e.requestPool.StartAutoOptimize(30 * time.Second)

	// Initialize work-stealing worker pool
	numWorkers := runtime.NumCPU()
	e.workerPool = pools.NewWorkerPool(numWorkers)

	log.Printf("📊 Fine-grained pools initialized:")
	log.Printf("   - Connection pool: 10000 capacity")
	log.Printf("   - Context pool: 500 warmup, 95%% target")
	log.Printf("   - Request pool: 500 warmup, 95%% target")
	log.Printf("   - Byte pool: 4-tier (512/2K/8K/32K)")
	log.Printf("   - Worker pool: %d workers (work-stealing)", numWorkers)
	log.Printf("   - GC: Optimized for high throughput (GOGC=300)")

	return e
}

// GET registers a GET route
func (e *Engine) GET(path string, handler HandlerFunc) {
	e.router.Add("GET", path, func(ctx any) {
		handler(ctx.(http.Context))
	})
}

// POST registers a POST route
func (e *Engine) POST(path string, handler HandlerFunc) {
	e.router.Add("POST", path, func(ctx any) {
		handler(ctx.(http.Context))
	})
}

// PUT registers a PUT route
func (e *Engine) PUT(path string, handler HandlerFunc) {
	e.router.Add("PUT", path, func(ctx any) {
		handler(ctx.(http.Context))
	})
}

// DELETE registers a DELETE route
func (e *Engine) DELETE(path string, handler HandlerFunc) {
	e.router.Add("DELETE", path, func(ctx any) {
		handler(ctx.(http.Context))
	})
}

// PATCH registers a PATCH route
func (e *Engine) PATCH(path string, handler HandlerFunc) {
	e.router.Add("PATCH", path, func(ctx any) {
		handler(ctx.(http.Context))
	})
}

// HEAD registers a HEAD route
func (e *Engine) HEAD(path string, handler HandlerFunc) {
	e.router.Add("HEAD", path, func(ctx any) {
		handler(ctx.(http.Context))
	})
}

// OPTIONS registers an OPTIONS route
func (e *Engine) OPTIONS(path string, handler HandlerFunc) {
	e.router.Add("OPTIONS", path, func(ctx any) {
		handler(ctx.(http.Context))
	})
}

// Run starts the server
func (e *Engine) Run(addr string) error {
	laddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	lnFile, err := ln.File()
	if err != nil {
		return err
	}
	lfd := int(lnFile.Fd())

	if err := syscall.SetNonblock(lfd, true); err != nil {
		return err
	}

	e.poller, err = poller.NewPoller()
	if err != nil {
		return err
	}
	defer e.poller.Close()

	if err := e.poller.Add(lfd); err != nil {
		return err
	}

	log.Printf("🚀 High-Performance Server listening on %s", addr)
	log.Printf("⚡ Full epoll/kqueue with syscall.Write()")
	log.Printf("📊 Smart pools initialized with 300 objects warmup")

	go e.cleanupIdleConnections()

	for {
		// Wait up to 100ms (shorter timeout for better responsiveness)
		fds, err := e.poller.Wait(100)
		if err != nil {
			log.Printf("Poller wait error: %v", err)
			continue
		}

		for _, fd := range fds {
			if fd == lfd {
				e.acceptConnections(lfd)
			} else {
				e.handleConnectionEvent(fd)
			}
		}
	}
}

// acceptConnections accepts multiple pending connections
func (e *Engine) acceptConnections(lfd int) {
	for {
		nfd, _, err := syscall.Accept(lfd)
		if err != nil {
			if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
				return
			}
			log.Printf("Accept error: %v", err)
			return
		}

		if err := syscall.SetNonblock(nfd, true); err != nil {
			syscall.Close(nfd)
			continue
		}

		// TCP_NODELAY: Disable Nagle's algorithm
		syscall.SetsockoptInt(nfd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1)

		// SO_KEEPALIVE: Enable TCP keepalive
		syscall.SetsockoptInt(nfd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1)

		// Configure keepalive timing (macOS: TCP_KEEPALIVE = 0x10)
		// Wait 30s before first probe
		syscall.SetsockoptInt(nfd, syscall.IPPROTO_TCP, 0x10, 30)

		conn := e.connectionPool.Get().(*Connection)
		conn.SetFD(nfd)
		conn.state = StateReading
		conn.readBuf = e.bytePool.Get(8192)
		conn.readOffset = 0
		conn.keepAlive = true

		if err := e.poller.Add(nfd); err != nil {
			e.connectionPool.Put(conn)
			syscall.Close(nfd)
			continue
		}

		e.connMu.Lock()
		e.connections[nfd] = conn
		e.connMu.Unlock()
	}
}

// handleConnectionEvent handles events on a connection
func (e *Engine) handleConnectionEvent(fd int) {
	e.connMu.RLock()
	conn, ok := e.connections[fd]
	e.connMu.RUnlock()

	if !ok {
		return
	}

	conn.lastActive = time.Now()

	switch conn.state {
	case StateReading, StateKeepalive:
		e.handleRead(conn)
	case StateWriting:
		conn.state = StateKeepalive
	}
}

// handleRead reads and processes HTTP requests
func (e *Engine) handleRead(conn *Connection) {
	n, err := syscall.Read(conn.fd, conn.readBuf[conn.readOffset:])
	if err != nil {
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			return
		}
		e.closeConnection(conn.fd)
		return
	}

	if n == 0 {
		e.closeConnection(conn.fd)
		return
	}

	conn.readOffset += n

	req, err := http.ParseRequest(conn.readBuf[:conn.readOffset])
	if err != nil {
		if conn.readOffset >= len(conn.readBuf) {
			e.sendError(conn, 400, "Bad Request")
			e.closeConnection(conn.fd)
		}
		// Partial request, wait for more data
		return
	}

	conn.readOffset = 0
	conn.request = req
	conn.state = StateProcessing

	e.processRequest(conn)
}

// processRequest processes a single request
func (e *Engine) processRequest(conn *Connection) {
	// For lightweight HTTP handlers, process inline for minimal latency
	// Worker pool can be enabled for CPU-intensive handlers
	h, params := e.router.Find(conn.request.Method, conn.request.Path)

	if h == nil {
		e.sendError(conn, 404, "Not Found")
		e.checkKeepAlive(conn)
		return
	}

	ctx := e.contextPool.Get().(*http.FDContext)
	ctx.Reset(conn.fd, conn.request)

	for k, v := range params {
		ctx.SetParam(k, v)
	}

	h(ctx)

	e.contextPool.Put(ctx)
	e.checkKeepAlive(conn)
}

// sendError sends an error response
func (e *Engine) sendError(conn *Connection, code int, message string) {
	response := []byte("HTTP/1.1 ")
	response = appendInt(response, code)
	response = append(response, ' ')
	response = append(response, message...)
	response = append(response, "\r\n\r\n"...)

	syscall.Write(conn.fd, response)
}

// checkKeepAlive checks if connection should be kept alive
func (e *Engine) checkKeepAlive(conn *Connection) {
	if conn.request.Proto == "HTTP/1.0" || conn.request.Connection == "close" {
		e.closeConnection(conn.fd)
	} else {
		// Keep connection alive - reset for next request
		conn.state = StateReading
		conn.readOffset = 0
		http.ReleaseRequest(conn.request)
		conn.request = nil
		conn.lastActive = time.Now()
	}
}

// closeConnection closes and cleans up a connection
func (e *Engine) closeConnection(fd int) {
	e.connMu.Lock()
	conn, ok := e.connections[fd]
	if ok {
		delete(e.connections, fd)
	}
	e.connMu.Unlock()

	if ok {
		// Clean up in correct order:
		// 1. Remove from poller first (stop receiving events)
		e.poller.Remove(fd)

		// 2. Clean up pooled objects
		if conn.request != nil {
			e.requestPool.Put(conn.request)
			conn.request = nil
		}
		if conn.context != nil {
			e.contextPool.Put(conn.context)
			conn.context = nil
		}
		if conn.readBuf != nil {
			e.bytePool.Put(conn.readBuf)
			conn.readBuf = nil
		}

		// 3. Close the fd
		syscall.Close(fd)

		// 4. Reset and return connection to pool
		conn.Reset()
		e.connectionPool.Put(conn)
	}
}

// cleanupIdleConnections periodically removes idle connections
func (e *Engine) cleanupIdleConnections() {
	ticker := time.NewTicker(1 * time.Second) // Run every second
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		var toClose []int

		e.connMu.RLock()
		for fd, conn := range e.connections {
			// Close connections that have been idle too long (in any state except processing)
			if conn.state != StateProcessing && now.Sub(conn.lastActive) > e.idleTimeout {
				toClose = append(toClose, fd)
			}
		}
		e.connMu.RUnlock()

		for _, fd := range toClose {
			e.closeConnection(fd)
		}
	}
}

// Helper function to append int to byte slice
func appendInt(b []byte, i int) []byte {
	if i == 0 {
		return append(b, '0')
	}

	if i < 0 {
		b = append(b, '-')
		i = -i
	}

	var digits [20]byte
	n := 0
	for i > 0 {
		digits[n] = byte('0' + i%10)
		i /= 10
		n++
	}

	for n > 0 {
		n--
		b = append(b, digits[n])
	}

	return b
}
