package http

import (
	"testing"
)

// TestFDContextBasic 测试基本功能
func TestFDContextBasic(t *testing.T) {
	req := &Request{
		Method: "GET",
		Path:   "/test",
	}

	ctx := NewFDContext(1, req)

	if ctx.Method() != "GET" {
		t.Errorf("Expected method GET, got %s", ctx.Method())
	}

	if ctx.Path() != "/test" {
		t.Errorf("Expected path /test, got %s", ctx.Path())
	}
}

// TestFDContextParams 测试参数功能
func TestFDContextParams(t *testing.T) {
	req := &Request{
		Method: "GET",
		Path:   "/users/123",
	}

	ctx := NewFDContext(1, req)

	// 设置参数
	ctx.SetParam("id", "123")
	ctx.SetParam("name", "alice")

	// 获取参数
	if ctx.Param("id") != "123" {
		t.Errorf("Expected id=123, got %s", ctx.Param("id"))
	}

	if ctx.Param("name") != "alice" {
		t.Errorf("Expected name=alice, got %s", ctx.Param("name"))
	}

	// 不存在的参数
	if ctx.Param("notexist") != "" {
		t.Error("Expected empty string for non-existent param")
	}
}

// TestFDContextHeaders 测试头部功能
func TestFDContextHeaders(t *testing.T) {
	req := &Request{
		Method:      "POST",
		Path:        "/api",
		ContentType: "application/json",
		UserAgent:   "TestAgent/1.0",
	}

	ctx := NewFDContext(1, req)

	// 读取预定义头部
	if ctx.Header("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type=application/json, got %s", ctx.Header("Content-Type"))
	}

	if ctx.GetHeader("User-Agent") != "TestAgent/1.0" {
		t.Errorf("Expected User-Agent=TestAgent/1.0, got %s", ctx.GetHeader("User-Agent"))
	}

	// 设置响应头
	ctx.SetHeader("X-Custom", "test-value")

	// 注意：目前无法读取响应头，这里只测试不崩溃
}

// TestFDContextAbort 测试终止功能
func TestFDContextAbort(t *testing.T) {
	req := &Request{
		Method: "GET",
		Path:   "/",
	}

	ctx := NewFDContext(1, req)

	if ctx.IsAborted() {
		t.Error("New context should not be aborted")
	}

	ctx.Abort()

	if !ctx.IsAborted() {
		t.Error("Context should be aborted after calling Abort()")
	}
}

// TestFDContextStatus 测试状态码功能
func TestFDContextStatus(t *testing.T) {
	req := &Request{
		Method: "GET",
		Path:   "/",
	}

	ctx := NewFDContext(1, req)

	// 默认状态码应该是 200
	// 注意：目前无法直接读取状态码，这里只测试不崩溃
	ctx.Status(404)
	ctx.Status(200)
}

// TestFDContextReset 测试重置功能
func TestFDContextReset(t *testing.T) {
	req1 := &Request{
		Method: "GET",
		Path:   "/first",
	}

	ctx := NewFDContext(1, req1)
	ctx.SetParam("id", "123")
	ctx.SetHeader("X-Test", "value")
	ctx.Abort()

	// 重置
	req2 := &Request{
		Method: "POST",
		Path:   "/second",
	}
	ctx.Reset(2, req2)

	// 检查是否正确重置
	if ctx.Method() != "POST" {
		t.Errorf("Expected method POST after reset, got %s", ctx.Method())
	}

	if ctx.Path() != "/second" {
		t.Errorf("Expected path /second after reset, got %s", ctx.Path())
	}

	if ctx.IsAborted() {
		t.Error("Context should not be aborted after reset")
	}

	// 旧参数应该被清除
	if ctx.Param("id") != "" {
		t.Error("Old params should be cleared after reset")
	}
}

// TestFDContextJSON 测试 JSON 响应
func TestFDContextJSON(t *testing.T) {
	req := &Request{
		Method: "GET",
		Path:   "/",
	}

	ctx := NewFDContext(1, req)

	data := map[string]interface{}{
		"message": "hello",
		"count":   123,
	}

	// 调用 JSON 方法（不会实际发送，只是测试不崩溃）
	ctx.JSON(200, data)
}

// TestFDContextString 测试字符串响应
func TestFDContextString(t *testing.T) {
	req := &Request{
		Method: "GET",
		Path:   "/",
	}

	ctx := NewFDContext(1, req)

	// 调用 String 方法
	ctx.String(200, "Hello, World!")
}

// BenchmarkFDContextSetParam 参数设置基准测试
func BenchmarkFDContextSetParam(b *testing.B) {
	req := &Request{
		Method: "GET",
		Path:   "/users/123",
	}

	ctx := NewFDContext(1, req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.SetParam("id", "123")
		ctx.Reset(1, req) // 重置以便下次测试
	}
}

// BenchmarkFDContextGetParam 参数获取基准测试
func BenchmarkFDContextGetParam(b *testing.B) {
	req := &Request{
		Method: "GET",
		Path:   "/users/123",
	}

	ctx := NewFDContext(1, req)
	ctx.SetParam("id", "123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ctx.Param("id")
	}
}

// BenchmarkFDContextJSON JSON 响应基准测试
func BenchmarkFDContextJSON(b *testing.B) {
	req := &Request{
		Method: "GET",
		Path:   "/",
	}

	ctx := NewFDContext(1, req)
	data := map[string]interface{}{
		"message": "hello",
		"count":   123,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.JSON(200, data)
		ctx.Reset(1, req)
	}
}
