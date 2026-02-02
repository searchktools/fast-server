# Core - High-Performance HTTP Engine Core

This directory contains the core implementation of the zero-allocation HTTP engine, organized into multiple sub-packages by functionality.

## 📁 Directory Structure

```
core/
├── engine.go           # Main engine entry point
├── constants.go        # Global constants and errors
│
├── http/               # HTTP protocol layer
│   ├── context.go      # Context interface and implementation
│   ├── request.go      # Request structure
│   └── parser.go       # HTTP parser
│
├── router/             # Routing layer
│   └── radix.go        # Radix Tree router
│
├── poller/             # IO multiplexing layer
│   ├── poller.go       # Poller interface
│   ├── kqueue.go       # macOS kqueue implementation
│   ├── epoll.go        # Linux epoll implementation
│   └── uring.go        # Linux io_uring (experimental)
│
└── optimize/           # Performance optimizations
    ├── simd.go         # SIMD string comparison
    └── simd_amd64.s    # AVX2 assembly implementation
```

## 🎯 Architecture Design

### Layered Architecture

```
┌─────────────────────────────────────┐
│         Engine (core)               │  ← Main engine, coordinates layers
├─────────────────────────────────────┤
│         HTTP Layer                  │  ← Request parsing, context management
│   (http.Context, Request, Parser)   │
├─────────────────────────────────────┤
│        Router Layer                 │  ← Route matching
│      (router.RadixRouter)           │
├─────────────────────────────────────┤
│       Poller Layer                  │  ← IO multiplexing
│  (poller.Poller interface)          │
│   ├─ kqueue (macOS)                 │
│   ├─ epoll (Linux)                  │
│   └─ uring (Linux 5.1+)             │
└─────────────────────────────────────┘
          ↓
    ┌─────────────┐
    │  Optimize   │  ← Performance optimizations like SIMD
    └─────────────┘
```

### Package Dependencies

```
core (engine)
  ├─> core/http (Context, Request, Parser)
  ├─> core/router (RadixRouter)
  └─> core/poller (Poller interface)

core/http
  └─> (independent, no external dependencies)

core/router  
  └─> (independent, no external dependencies)

core/poller
  └─> (independent, no external dependencies)

core/optimize
  └─> (independent, no external dependencies)
```

**Design Principle**: Unidirectional dependencies, avoiding circular references

## 📦 Sub-Package Descriptions

### 1. core (main package)

**Files**: `engine.go`, `constants.go`

**Responsibilities**:
- Provides main `Engine`
- Defines `HandlerFunc` type
- Coordinates layer components

**Usage**:
```go
import "cluster-builder/internal/server/core"

engine := core.NewEngine()
engine.GET("/path", handler)
engine.Run(":8080")
```

### 2. core/http

**Files**: `context.go`, `request.go`, `parser.go`

**Responsibilities**:
- HTTP request parsing
- Zero-allocation context management
- Request/response handling

**Core Types**:
- `Context` - Interface, defines context behavior
- `StandardContext` - Implementation, zero-allocation optimization
- `Request` - HTTP request structure
- `ParseRequest()` - HTTP parser

**Usage**:
```go
import corehttp "cluster-builder/internal/server/core/http"

func handler(ctx corehttp.Context) {
    name := ctx.Param("name")
    ctx.JSON(200, map[string]string{
        "message": "Hello, " + name,
    })
}
```

### 3. core/router

**Files**: `radix.go`

**Responsibilities**:
- Radix Tree route matching
- O(k) lookup complexity (k = path length)

**Core Types**:
- `RadixRouter` - Router
- `HandlerFunc` - Handler function type

**Features**:
- Supports static routes
- Supports parameter routes (`:param`)
- Memory-efficient trie structure

### 4. core/poller

**Files**: `poller.go`, `kqueue.go`, `epoll.go`, `uring.go`

**Responsibilities**:
- IO event monitoring
- Platform-specific implementations
- Non-blocking IO

**Core Interface**:
```go
type Poller interface {
    Add(fd int) error
    Remove(fd int) error
    Wait(timeout int) ([]int, error)
    Close() error
}
```

**Platform Support**:
- **macOS**: kqueue
- **Linux**: epoll (default)
- **Linux 5.1+**: io_uring (experimental)

### 5. core/optimize

**Files**: `simd.go`, `simd_amd64.s`

**Responsibilities**:
- SIMD string comparison
- AVX2 acceleration (amd64)
- Automatic fallback to standard implementation

## 🚀 Usage Examples

### Basic Usage

```go
package main

import (
    "cluster-builder/internal/server/core"
    corehttp "cluster-builder/internal/server/core/http"
)

func main() {
    engine := core.NewEngine()
    
    engine.GET("/", func(ctx corehttp.Context) {
        ctx.JSON(200, map[string]string{
            "message": "Hello, World!",
        })
    })
    
    engine.Run(":8080")
}
```

### Using Path Parameters

```go
engine.GET("/users/:id", func(ctx corehttp.Context) {
    id := ctx.Param("id")
    ctx.JSON(200, map[string]any{
        "user_id": id,
    })
})
```

### POST Request Handling

```go
engine.POST("/api/create", func(ctx corehttp.Context) {
    var input struct {
        Name string `json:"name"`
    }
    
    if err := ctx.Bind(&input); err != nil {
        ctx.Error(400, "Invalid request")
        return
    }
    
    ctx.Success(map[string]string{
        "name": input.Name,
    })
})
```

## ⚡ Performance Characteristics

### Zero-Allocation Optimizations

**Fixed Array Parameters (1-4)**:
```go
// StandardContext internals
paramKeys   [4]string    // zero allocation
paramValues [4]string    // zero allocation
```

**Predefined Header Fields**:
```go
// Request internals
ContentType   string  // zero-allocation access
UserAgent     string  // zero-allocation access
Host          string  // zero-allocation access
```

**Object Pooling**:
```go
var contextPool = sync.Pool{
    New: func() any {
        return &StandardContext{
            responseBuf: make([]byte, 0, 4096),
        }
    },
}
```

### Performance Metrics

- **Latency**: ~100-120ns per request
- **Throughput**: 10M+ requests/second
- **Memory**: ~100 bytes per request
- **Allocations**: 5-6 allocations per request

## 🔧 Extension Guide

### Adding New HTTP Methods

Edit `engine.go`:
```go
func (e *Engine) PATCH(path string, handler HandlerFunc) {
    e.router.Add("PATCH", path, func(ctx any) {
        handler(ctx.(http.Context))
    })
}
```

### Adding Custom Header Fields

Edit `http/request.go`:
```go
type Request struct {
    // ...
    CustomHeader string  // Add predefined field
}
```

### Implementing a New Poller

Create `poller/mypoller.go`:
```go
package poller

type MyPoller struct {
    // ...
}

func (p *MyPoller) Add(fd int) error { /* ... */ }
func (p *MyPoller) Remove(fd int) error { /* ... */ }
func (p *MyPoller) Wait(timeout int) ([]int, error) { /* ... */ }
func (p *MyPoller) Close() error { /* ... */ }
```

## 📝 Notes

### Import Aliases

To avoid package name conflicts, use aliases:
```go
import (
    corehttp "cluster-builder/internal/server/core/http"
)
```

### Context Type

- Use `corehttp.Context` (interface), not `*Context`
- Handler function signature: `func(corehttp.Context)`

### Build Tags

Platform-specific code uses build tags:
- `// +build darwin` - macOS
- `// +build linux` - Linux
- `// +build amd64` - x86-64

## 🎯 Design Philosophy

1. **Single Responsibility** - Each sub-package handles one core functionality
2. **Zero Allocation** - Achieved through fixed arrays, predefined fields, and object pooling
3. **Platform Optimization** - Uses platform-specific optimal implementations
4. **Interface First** - Achieves decoupling and testability through interfaces
5. **Simple and Efficient** - Concise code, ultimate performance

## 📚 More Resources

- [Performance Analysis](../../docs/BENCHMARK_RESULTS_V6.0.md)
- [Optimization Plan](../../docs/V6_OPTIMIZATION_PLAN.md)
- [Usage Examples](../../examples/)

---

**Core Refactoring Complete** - Clear layered architecture + zero-allocation performance ⚡
