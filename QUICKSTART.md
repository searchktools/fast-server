# Fast-Server 快速使用指南

## 📦 库位置

- **源码**: `/Users/hardy/golang/fast-server`
- **包名**: `github.com/searchktools/fast-server`

## 🚀 在新项目中使用

### 1. 添加依赖

```bash
# 方法 1: 本地开发 (推荐)
go mod edit -replace github.com/searchktools/fast-server=/Users/hardy/golang/fast-server
go get github.com/searchktools/fast-server

# 方法 2: 发布到 GitHub 后
go get github.com/searchktools/fast-server@latest
```

### 2. 最小示例

```go
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
    engine.GET("/", func(ctx http.Context) {
        ctx.String(200, "Hello, Fast Server!")
    })
    
    application.Run()
}
```

### 3. 完整示例

参考 `/Users/hardy/golang/fast-server/examples/basic/main.go`

## 📖 主要 API

### 应用配置

```go
import "github.com/searchktools/fast-server/config"

cfg := config.New() // 从命令行参数和环境变量加载配置
// -port=8080
// -env=production
// -read-timeout=10
// -write-timeout=30
```

### 应用创建

```go
import "github.com/searchktools/fast-server/app"

// 方式 1: 标准创建
app := app.New(cfg)

// 方式 2: 自定义引擎
engine := core.NewEngine()
app := app.NewWithEngine(cfg, engine)

// 获取引擎用于路由注册
engine := app.Engine()
```

### 路由注册

```go
import "github.com/searchktools/fast-server/core/http"

engine := app.Engine()

// GET 请求
engine.GET("/path", func(ctx http.Context) {
    ctx.String(200, "response")
})

// POST 请求
engine.POST("/api/users", func(ctx http.Context) {
    ctx.JSON(201, map[string]string{"status": "created"})
})

// 路径参数
engine.GET("/users/:id", func(ctx http.Context) {
    id := ctx.Param("id")
    ctx.JSON(200, map[string]string{"user_id": id})
})

// 查询参数
engine.GET("/search", func(ctx http.Context) {
    q := ctx.Query("q")
    page := ctx.Query("page")
    // ...
})
```

### Context 方法

```go
// 请求信息
method := ctx.Method()       // HTTP 方法
path := ctx.Path()           // 请求路径
param := ctx.Param("key")    // 路径参数
query := ctx.Query("key")    // 查询参数
header := ctx.Header("key")  // 请求头
body := ctx.Body()           // 请求体

// 响应方法
ctx.String(200, "text")      // 文本响应
ctx.JSON(200, data)          // JSON 响应
ctx.Bytes(200, []byte{})     // 字节响应
ctx.Data(200, "text/html", []byte{}) // 自定义内容类型
ctx.Error(500, "error msg")  // 错误响应
ctx.Success(data)            // 成功响应 (200 + JSON)
ctx.ServeFile("/path/file")  // 文件响应

// 数据绑定
err := ctx.Bind(&struct{})   // 绑定请求体到结构体
```

## 🛠 高级功能

### 中间件

```go
import "github.com/searchktools/fast-server/core/middleware"

// 使用内置中间件
pipeline := middleware.NewPipeline()
pipeline.Use(middleware.Recovery())    // 恢复 panic
pipeline.Use(middleware.RequestID())   // 添加请求 ID
pipeline.Use(middleware.CORS())        // CORS 支持
pipeline.Use(middleware.RateLimiter(1000)) // 限流

// 异步中间件 (非阻塞)
asyncPipeline := middleware.NewAsyncPipeline(4)
asyncPipeline.UseAsync(middleware.Logger())   // 异步日志
asyncPipeline.UseAsync(middleware.Metrics())  // 异步指标
```

### WebSocket

```go
import "github.com/searchktools/fast-server/core/websocket"

hub := websocket.NewHub()
go hub.Run()

engine.GET("/ws", func(ctx http.Context) {
    websocket.HandleWebSocket(ctx, hub)
})
```

### Server-Sent Events

```go
import "github.com/searchktools/fast-server/core/sse"

broker := sse.NewBroker()
go broker.Run()

engine.GET("/events", func(ctx http.Context) {
    sse.HandleSSE(ctx, broker)
})
```

### RPC 服务

```go
import (
    "github.com/searchktools/fast-server/core/rpc/server"
    "github.com/searchktools/fast-server/core/rpc/codec"
)

rpcServer := server.NewRPCServer(":9090", codec.NewJSONCodec())
// 注册服务...
go rpcServer.Serve()
```

### 可观测性

```go
import "github.com/searchktools/fast-server/core/observability"

monitor := observability.NewPerformanceMonitor()
monitor.RecordRequest(path, latency, isError)

// eBPF 追踪 (需要 Linux + root)
tracer := observability.NewEBPFTracer()
tracer.Start()
defer tracer.Stop()
```

## 📊 性能特性

- **15M+ RPS**: 每秒处理 1500 万+ 请求
- **~68ns 延迟**: 超低延迟
- **16B/请求**: 极低内存占用
- **零分配**: 最小化 GC 压力

## 🏗 架构组件

| 组件 | 包路径 | 说明 |
|------|--------|------|
| 应用框架 | `app` | 应用封装和生命周期管理 |
| 配置管理 | `config` | 配置加载和管理 |
| 核心引擎 | `core` | HTTP 服务器核心引擎 |
| HTTP 处理 | `core/http` | HTTP 请求/响应处理 |
| 路由器 | `core/router` | 高性能路由匹配 |
| 中间件 | `core/middleware` | 中间件管道 |
| 对象池 | `core/pools` | 各种对象池 |
| I/O 复用 | `core/poller` | epoll/kqueue/io_uring |
| 性能优化 | `core/optimize` | SIMD 等优化 |
| WebSocket | `core/websocket` | WebSocket 支持 |
| SSE | `core/sse` | Server-Sent Events |
| HTTP/2 | `core/http2` | HTTP/2 支持 |
| RPC | `core/rpc` | RPC 框架 |
| 可观测性 | `core/observability` | 监控和追踪 |

## 🔗 相关文件

- **README**: `/Users/hardy/golang/fast-server/README.md`
- **示例**: `/Users/hardy/golang/fast-server/examples/`
- **测试**: `/Users/hardy/golang/fast-server/core/tests/`

## 📝 迁移注意事项

如果从 cluster-builder 迁移，将以下导入路径替换：

```go
// 旧路径 → 新路径
"cluster-builder/internal/app" 
  → "github.com/searchktools/fast-server/app"

"cluster-builder/internal/config" 
  → "github.com/searchktools/fast-server/config"

"cluster-builder/internal/server/core" 
  → "github.com/searchktools/fast-server/core"

"cluster-builder/internal/server/core/http" 
  → "github.com/searchktools/fast-server/core/http"

// 以此类推...
```

## 🐛 调试

```bash
# 启用详细日志
GODEBUG=gctrace=1 ./your-app

# 性能分析
go tool pprof http://localhost:8080/debug/pprof/profile

# 编译优化
go build -ldflags="-s -w" -o app
```

## 🚢 发布到 GitHub

```bash
cd /Users/hardy/golang/fast-server
git init
git add .
git commit -m "Initial commit"
git remote add origin git@github.com:searchktools/fast-server.git
git push -u origin main
git tag v1.0.0
git push --tags
```

然后更新项目依赖：

```go
// go.mod
require github.com/searchktools/fast-server v1.0.0
// 移除 replace 指令
```
