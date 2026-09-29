//go:build amd64

#include "textflag.h"

// func load(p *int64) int64
TEXT ·load(SB), NOSPLIT, $0-16
	MOVQ p+0(FP), AX
	MOVQ (AX), AX
	MOVQ AX, ret+8(FP)
	RET
