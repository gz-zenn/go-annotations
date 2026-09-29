//go:build arm64

#include "textflag.h"

// func load(p *int64) int64
TEXT ·load(SB), NOSPLIT, $0-16
	MOVD p+0(FP), R0
	MOVD (R0), R0
	MOVD R0, ret+8(FP)
	RET
