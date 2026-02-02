//go:build !(amd64 || arm64)

package optimize

// Fallback implementations for architectures without SIMD support

// comparePathAVX2 fallback for non-amd64 architectures
func comparePathAVX2(a, b string) bool {
	return a == b
}

// comparePathNEON fallback for non-arm64 architectures
func comparePathNEON(a, b string) bool {
	return a == b
}
