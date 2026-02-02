package optimize

import (
	"golang.org/x/sys/cpu"
)

// SIMD capabilities detection
var (
	useAVX2 bool // x86_64 AVX2
	useNEON bool // ARM64 NEON
)

func init() {
	// Check CPU features based on architecture
	if cpu.ARM64.HasASIMD {
		// ARM64: NEON is standard on ARMv8 (ASIMD = Advanced SIMD)
		useNEON = true
	}
	if cpu.X86.HasAVX2 {
		// x86_64: Check for AVX2
		useAVX2 = true
	}
}

// ComparePathSIMD compares two paths efficiently
// Uses SIMD instructions if available, otherwise falls back to standard comparison
func ComparePathSIMD(a, b string) bool {
	// Quick length check
	if len(a) != len(b) {
		return false
	}

	// For short strings, standard comparison is faster
	if len(a) < 16 {
		return a == b
	}

	// Use NEON on ARM64
	if useNEON {
		return comparePathNEON(a, b)
	}

	// Use AVX2 on x86_64
	if useAVX2 {
		return comparePathAVX2(a, b)
	}

	// Fallback to standard comparison
	return a == b
}
