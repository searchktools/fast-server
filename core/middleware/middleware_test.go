package middleware

import (
	"github.com/searchktools/fast-server/core/http"
	"testing"
	"time"
)

// TestPipelineBasic 测试基本管道功能
func TestPipelineBasic(t *testing.T) {
	pipeline := NewPipeline()

	executed := false
	middleware := func(ctx *http.FDContext) {
		executed = true
	}

	pipeline.Use(middleware)

	// 创建测试上下文
	ctx := &http.FDContext{}
	finalHandler := func(ctx *http.FDContext) {}

	pipeline.Execute(ctx, finalHandler)

	if !executed {
		t.Error("Middleware was not executed")
	}
}

// TestPipelineAbort 测试中间件终止
func TestPipelineAbort(t *testing.T) {
	pipeline := NewPipeline()

	middleware1Executed := false
	middleware2Executed := false
	finalExecuted := false

	middleware1 := func(ctx *http.FDContext) {
		middleware1Executed = true
		ctx.Abort() // 终止后续处理
	}

	middleware2 := func(ctx *http.FDContext) {
		middleware2Executed = true
	}

	pipeline.Use(middleware1)
	pipeline.Use(middleware2)

	ctx := &http.FDContext{}
	finalHandler := func(ctx *http.FDContext) {
		finalExecuted = true
	}

	pipeline.Execute(ctx, finalHandler)

	if !middleware1Executed {
		t.Error("Middleware 1 should be executed")
	}

	if middleware2Executed {
		t.Error("Middleware 2 should not be executed after abort")
	}

	if finalExecuted {
		t.Error("Final handler should not be executed after abort")
	}
}

// TestPipelineOrder 测试中间件执行顺序
func TestPipelineOrder(t *testing.T) {
	pipeline := NewPipeline()

	order := []int{}

	middleware1 := func(ctx *http.FDContext) {
		order = append(order, 1)
	}

	middleware2 := func(ctx *http.FDContext) {
		order = append(order, 2)
	}

	middleware3 := func(ctx *http.FDContext) {
		order = append(order, 3)
	}

	pipeline.Use(middleware1)
	pipeline.Use(middleware2)
	pipeline.Use(middleware3)

	ctx := &http.FDContext{}
	finalHandler := func(ctx *http.FDContext) {
		order = append(order, 4)
	}

	pipeline.Execute(ctx, finalHandler)

	expected := []int{1, 2, 3, 4}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d executions, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("Expected order[%d] = %d, got %d", i, v, order[i])
		}
	}
}

// TestRecoveryMiddleware 测试 Recovery 中间件
func TestRecoveryMiddleware(t *testing.T) {
	pipeline := NewPipeline()
	pipeline.Use(Recovery())

	recovered := false
	defer func() {
		if r := recover(); r != nil {
			recovered = true
		}
	}()

	ctx := &http.FDContext{}
	finalHandler := func(ctx *http.FDContext) {
		panic("test panic")
	}

	pipeline.Execute(ctx, finalHandler)

	// Recovery 应该捕获 panic，不应该传播到这里
	if recovered {
		t.Error("Panic should be recovered by Recovery middleware")
	}
}

// TestRequestIDMiddleware 测试 RequestID 中间件
func TestRequestIDMiddleware(t *testing.T) {
	middleware := RequestID()

	ctx := &http.FDContext{}
	middleware(ctx)

	// 应该设置了 X-Request-ID 头
	// 注意：需要 FDContext 支持 GetResponseHeader 才能完整测试
	// 这里只测试不会崩溃
}

// TestRateLimiter 测试限流中间件
func TestRateLimiter(t *testing.T) {
	limiter := RateLimiter(2) // 每秒 2 个请求

	ctx1 := &http.FDContext{}
	ctx2 := &http.FDContext{}
	ctx3 := &http.FDContext{}

	// 前两个请求应该通过
	limiter(ctx1)
	if ctx1.IsAborted() {
		t.Error("First request should not be rate limited")
	}

	limiter(ctx2)
	if ctx2.IsAborted() {
		t.Error("Second request should not be rate limited")
	}

	// 第三个请求应该被限流
	limiter(ctx3)
	if !ctx3.IsAborted() {
		t.Error("Third request should be rate limited")
	}

	// 等待 1 秒后应该恢复
	time.Sleep(1100 * time.Millisecond)

	ctx4 := &http.FDContext{}
	limiter(ctx4)
	if ctx4.IsAborted() {
		t.Error("Request after refill should not be rate limited")
	}
}

// TestAsyncPipeline 测试异步管道
func TestAsyncPipeline(t *testing.T) {
	asyncPipeline := NewAsyncPipeline(2)

	syncExecuted := false
	asyncExecuted := false

	syncMiddleware := func(ctx *http.FDContext) {
		syncExecuted = true
	}

	asyncMiddleware := func(ctx *http.FDContext) {
		asyncExecuted = true
	}

	asyncPipeline.UseSync(syncMiddleware)
	asyncPipeline.UseAsync(asyncMiddleware)

	ctx := &http.FDContext{}
	finalHandler := func(ctx *http.FDContext) {}

	asyncPipeline.Execute(ctx, finalHandler)

	if !syncExecuted {
		t.Error("Sync middleware was not executed")
	}

	// 异步中间件可能还没执行，等待一下
	time.Sleep(100 * time.Millisecond)

	if !asyncExecuted {
		t.Error("Async middleware was not executed")
	}
}

// BenchmarkPipeline 管道性能基准测试
func BenchmarkPipeline(b *testing.B) {
	pipeline := NewPipeline()

	middleware1 := func(ctx *http.FDContext) {}
	middleware2 := func(ctx *http.FDContext) {}
	middleware3 := func(ctx *http.FDContext) {}

	pipeline.Use(middleware1)
	pipeline.Use(middleware2)
	pipeline.Use(middleware3)
	pipeline.Compile()

	ctx := &http.FDContext{}
	finalHandler := func(ctx *http.FDContext) {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipeline.Execute(ctx, finalHandler)
		ctx.Reset(0, nil) // 重置 abort 状态
	}
}

// BenchmarkRecoveryMiddleware Recovery 中间件基准测试
func BenchmarkRecoveryMiddleware(b *testing.B) {
	middleware := Recovery()
	ctx := &http.FDContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware(ctx)
	}
}

// BenchmarkRequestIDMiddleware RequestID 中间件基准测试
func BenchmarkRequestIDMiddleware(b *testing.B) {
	middleware := RequestID()
	ctx := &http.FDContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware(ctx)
	}
}

// BenchmarkRateLimiter RateLimiter 中间件基准测试
func BenchmarkRateLimiter(b *testing.B) {
	middleware := RateLimiter(1000000) // 很高的限制，避免影响测试
	ctx := &http.FDContext{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware(ctx)
		ctx.Reset(0, nil)
	}
}
