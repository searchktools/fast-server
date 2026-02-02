# Fast Server

A high-performance, zero-allocation HTTP server framework for Go, optimized for extreme throughput and low latency.

## Features

- 🚀 **Ultra-High Performance**: 15M+ requests/second with ~68ns latency
- 🔋 **Zero Allocation**: Minimized memory allocations (16B/request)
- ⚡ **I/O Multiplexing**: Support for epoll (Linux), kqueue (BSD/macOS), and io_uring
- 🛠 **Complete Feature Set**: HTTP/1.1, HTTP/2, WebSocket, SSE, RPC
- 🎯 **Advanced Routing**: Radix tree router with compiled routes
- 🏊 **Smart Pooling**: Worker pools, buffer pools, connection pools with GC tuning
- 📊 **Observability**: Built-in monitoring and eBPF tracing support
- 🔧 **Middleware Pipeline**: Flexible middleware system
- 🎨 **SIMD Optimization**: Platform-specific SIMD optimizations (AMD64/ARM64)

## Installation

```bash
go get github.com/searchktools/fast-server
```

## Quick Start

```go
package main

import (
	"log"
	
	"github.com/searchktools/fast-server/app"
	"github.com/searchktools/fast-server/config"
	"github.com/searchktools/fast-server/core"
	"github.com/searchktools/fast-server/core/http"
)

func main() {
	// Create configuration
	cfg := config.New()
	
	// Create application
	application := app.New(cfg)
	
	// Register routes
	engine := application.Engine()
	engine.GET("/hello", func(ctx http.Context) {
		ctx.String(200, "Hello, World!")
	})
	
	engine.GET("/json", func(ctx http.Context) {
		ctx.JSON(200, map[string]string{
			"message": "Fast Server",
			"status": "running",
		})
	})
	
	// Start server
	log.Println("Starting server on :8080")
	application.Run()
}
```

## Advanced Usage

### WebSocket Support

```go
import (
	"github.com/searchktools/fast-server/core"
	"github.com/searchktools/fast-server/core/websocket"
)

func main() {
	engine := core.NewEngine()
	
	hub := websocket.NewHub()
	go hub.Run()
	
	engine.GET("/ws", func(ctx *http.Context) {
		websocket.HandleWebSocket(ctx, hub)
	})
	
	// ... start server
}
```

### Server-Sent Events (SSE)

```go
import (
	"github.com/searchktools/fast-server/core"
	"github.com/searchktools/fast-server/core/sse"
)

func main() {
	engine := core.NewEngine()
	
	broker := sse.NewBroker()
	go broker.Run()
	
	engine.GET("/events", func(ctx *http.Context) {
		sse.HandleSSE(ctx, broker)
	})
	
	// ... start server
}
```

## Architecture

### Core Components

- **Engine**: Main HTTP server engine with zero-allocation design
- **Router**: High-performance radix tree router with pattern matching
- **Pools**: Smart object pooling for workers, buffers, and connections
- **Poller**: Platform-specific I/O multiplexing (epoll/kqueue/io_uring)
- **Middleware**: Composable middleware pipeline
- **Observability**: Monitoring and eBPF-based tracing

### Performance Optimizations

1. **Zero-Copy I/O**: Using sendfile for static file serving
2. **SIMD Operations**: Platform-specific SIMD optimizations
3. **Smart Pooling**: Reuse of objects to minimize GC pressure
4. **GC Tuning**: Automatic GC parameter tuning based on workload
5. **Connection Pooling**: Efficient connection management
6. **Buffer Pooling**: Reusable buffers to reduce allocations

## Benchmarks

```
Requests/sec:   15,000,000+
Latency (avg):  ~68ns
Throughput:     ~2.5GB/s
Memory/req:     16 bytes
```

See `core/tests/benchmark_test.go` for detailed benchmarks.

## Configuration

Configuration can be provided via command-line flags or environment variables:

```bash
# Command-line flags
./your-app -port 8080 -env production

# Environment variables
export PORT=8080
export ENV=production
./your-app
```

## API Documentation

For detailed API documentation, see the [GoDoc](https://pkg.go.dev/github.com/searchktools/fast-server).

## Requirements

- Go 1.25.5 or higher
- Linux (for epoll/io_uring), macOS/BSD (for kqueue), or other Unix-like systems

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

This project is extracted from the cluster-builder project and optimized for general-purpose use.
