/*
Package fast-server provides a high-performance, zero-allocation HTTP server framework for Go.

Fast-Server is optimized for extreme throughput and low latency, achieving 15M+ requests
per second with ~68ns average latency and only 16 bytes of memory allocation per request.

Features

  - Ultra-high performance: 15M+ RPS with ~68ns latency
  - Zero-allocation design: Minimized memory allocations (16B/request)
  - I/O multiplexing: Support for epoll (Linux), kqueue (BSD/macOS), and io_uring
  - Complete protocol support: HTTP/1.1, HTTP/2, WebSocket, SSE
  - Advanced routing: Radix tree router with compiled routes
  - Smart pooling: Worker pools, buffer pools, connection pools with GC tuning
  - Observability: Built-in monitoring and eBPF tracing support
  - Middleware pipeline: Flexible middleware system
  - SIMD optimization: Platform-specific optimizations (AMD64/ARM64)

Quick Start

Basic usage example:

package main

import (
    "github.com/searchktools/fast-server/app"
    "github.com/searchktools/fast-server/config"
    "github.com/searchktools/fast-server/core/http"
)

func main() {
    cfg := config.New()
    application := app.New(cfg)

    engine := application.Engine()
    engine.GET("/hello", func(ctx http.Context) {
        ctx.String(200, "Hello, World!")
    })

    engine.GET("/json", func(ctx http.Context) {
        ctx.JSON(200, map[string]string{
            "message": "Fast Server",
            "status":  "running",
        })
    })

    application.Run()
}

Modules

The framework is organized into several modules:

  - app: Application lifecycle management
  - config: Configuration loading and management
  - core: HTTP server core engine
  - core/http: HTTP request/response handling
  - core/router: High-performance routing
  - core/middleware: Middleware pipeline
  - core/pools: Object pooling (workers, buffers, connections)
  - core/poller: I/O multiplexing (epoll/kqueue/io_uring)
  - core/optimize: Performance optimizations (SIMD)
  - core/websocket: WebSocket support
  - core/sse: Server-Sent Events
  - core/http2: HTTP/2 support
  - core/rpc: RPC framework
  - core/observability: Monitoring and tracing

Performance

Fast-Server is designed for extreme performance:

  - Throughput: 15M+ requests/second
  - Latency: ~68ns average
  - Memory: 16 bytes per request
  - Connections: 10,000+ concurrent

For more information, see https://github.com/searchktools/fast-server
*/
package fastserver
