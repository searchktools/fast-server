//go:build amd64

package optimize

// comparePathAVX2 is implemented in assembly for AVX2
// Returns true if strings are equal
//
//go:noescape
func comparePathAVX2(a, b string) bool

// comparePathNEON is a stub for x86_64 (NEON is ARM only)
func comparePathNEON(a, b string) bool {
	return a == b
}
