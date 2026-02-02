// +build amd64

#include "textflag.h"

// func comparePathAVX2(a, b string) bool
// Compare two strings using AVX2 SIMD instructions
TEXT ·comparePathAVX2(SB), NOSPLIT, $0-33
    MOVQ a_base+0(FP), SI    // SI = &a[0]
    MOVQ a_len+8(FP), BX     // BX = len(a)
    MOVQ b_base+16(FP), DI   // DI = &b[0]
    
    // Quick check: if length is 0, strings are equal
    TESTQ BX, BX
    JZ equal
    
    // Process 32-byte chunks using AVX2 (if length >= 32)
    CMPQ BX, $32
    JL remainder
    
loop32:
    // Load 32 bytes from each string
    VMOVDQU (SI), Y0         // Load 32 bytes from a
    VMOVDQU (DI), Y1         // Load 32 bytes from b
    
    // Compare
    VPCMPEQB Y0, Y1, Y2      // Compare bytes
    VPMOVMSKB Y2, AX         // Move comparison mask to AX
    CMPL AX, $0xFFFFFFFF     // Check if all bytes equal
    JNE not_equal
    
    // Advance pointers
    ADDQ $32, SI
    ADDQ $32, DI
    SUBQ $32, BX
    
    // Continue if more than 32 bytes remain
    CMPQ BX, $32
    JGE loop32
    
remainder:
    // Handle remaining bytes (< 32)
    TESTQ BX, BX
    JZ equal
    
    // Compare remaining bytes one by one
remainder_loop:
    MOVB (SI), AL
    CMPB AL, (DI)
    JNE not_equal
    INCQ SI
    INCQ DI
    DECQ BX
    JNZ remainder_loop
    
equal:
    MOVB $1, ret+32(FP)      // Return true
    RET
    
not_equal:
    MOVB $0, ret+32(FP)      // Return false
    RET
