// +build arm64

#include "textflag.h"

// func comparePathNEON(a, b string) bool
// Compare two strings using ARM NEON SIMD instructions
TEXT ·comparePathNEON(SB), NOSPLIT, $0-33
    MOVD a_base+0(FP), R0     // R0 = &a[0]
    MOVD a_len+8(FP), R2      // R2 = len(a)
    MOVD b_base+16(FP), R1    // R1 = &b[0]
    
    // Quick check: if length is 0, strings are equal
    CBZ R2, equal
    
    // Process 16-byte chunks using NEON (if length >= 16)
    CMP $16, R2
    BLT remainder
    
loop16:
    // Load 16 bytes from each string
    VLD1 (R0), [V0.B16]       // Load 16 bytes from a
    VLD1 (R1), [V1.B16]       // Load 16 bytes from b
    
    // Compare
    VCMEQ V0.B16, V1.B16, V2.B16
    
    // Check if all bytes are equal
    // If V2 has all 0xFF, all bytes matched
    VMOV V2.D[0], R3
    VMOV V2.D[1], R4
    AND R4, R3, R3
    CMP $-1, R3               // Check if R3 == 0xFFFFFFFFFFFFFFFF
    BNE not_equal
    
    // Advance pointers
    ADD $16, R0, R0
    ADD $16, R1, R1
    SUB $16, R2, R2
    
    // Continue if more than 16 bytes remain
    CMP $16, R2
    BGE loop16
    
remainder:
    // Handle remaining bytes (< 16)
    CBZ R2, equal
    
    // Compare remaining bytes one by one
remainder_loop:
    MOVBU.P 1(R0), R3
    MOVBU.P 1(R1), R4
    CMP R4, R3
    BNE not_equal
    SUBS $1, R2, R2
    BNE remainder_loop
    
equal:
    MOVD $1, R0
    MOVB R0, ret+32(FP)       // Return true
    RET
    
not_equal:
    MOVB ZR, ret+32(FP)       // Return false
    RET
