//go:build arm64
// +build arm64

package optimize

// comparePathNEON is implemented in assembly for ARM NEON
// Returns true if strings are equal
//
//go:noescape
func comparePathNEON(a, b string) bool

// comparePathAVX2 is a stub for ARM64 (AVX2 is x86_64 only)
func comparePathAVX2(a, b string) bool {
	return a == b
}
