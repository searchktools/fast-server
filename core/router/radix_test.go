package router

import (
"testing"
)

// TestRadixRouterBasic tests basic static routing
func TestRadixRouterBasic(t *testing.T) {
router := NewRadixRouter()

handler := func(ctx any) {}
router.Add("GET", "/", handler)
router.Add("GET", "/hello", handler)
router.Add("GET", "/hello/world", handler)

tests := []struct {
path        string
shouldMatch bool
}{
{"/", true},
{"/hello", true},
{"/hello/world", true},
{"/notfound", false},
}

for _, tt := range tests {
h, _ := router.Find("GET", tt.path)
matched := (h != nil)
if matched != tt.shouldMatch {
t.Errorf("Path %s: expected match=%v, got match=%v", tt.path, tt.shouldMatch, matched)
}
}
}

// TestRadixRouterPriority tests route priority (exact > param)
func TestRadixRouterPriority(t *testing.T) {
router := NewRadixRouter()

exactHandler := func(ctx any) {}
paramHandler := func(ctx any) {}

router.Add("GET", "/user/admin", exactHandler)
router.Add("GET", "/user/:id", paramHandler)

tests := []struct {
path         string
shouldMatch  bool
isExactMatch bool
}{
{"/user/admin", true, true},
{"/user/123", true, false},
}

for _, tt := range tests {
h, params := router.Find("GET", tt.path)
if (h != nil) != tt.shouldMatch {
t.Errorf("Path %s: expected match=%v, got match=%v", tt.path, tt.shouldMatch, h != nil)
}
if tt.shouldMatch {
_, hasParam := params["id"]
if tt.isExactMatch && hasParam {
t.Errorf("Path %s: should be exact match, but got params", tt.path)
}
if !tt.isExactMatch && !hasParam {
t.Errorf("Path %s: should be param match, but no params", tt.path)
}
}
}
}

// Benchmarks
func BenchmarkRadixRouterStatic(b *testing.B) {
router := NewRadixRouter()
handler := func(ctx any) {}
router.Add("GET", "/hello/world", handler)

b.ResetTimer()
for i := 0; i < b.N; i++ {
router.Find("GET", "/hello/world")
}
}

func BenchmarkRadixRouterParam(b *testing.B) {
router := NewRadixRouter()
handler := func(ctx any) {}
router.Add("GET", "/user/:id", handler)

b.ResetTimer()
for i := 0; i < b.N; i++ {
router.Find("GET", "/user/123")
}
}
