// Package nosplit demonstrates //go:nosplit, the directive that removes the
// stack growth check from a function.
//
// Every ordinary Go function starts with a check for enough stack and grows
// the stack when there is not. The runtime uses //go:nosplit for functions
// that must not reenter the allocator: a signal handler, a path where
// reentrancy would deadlock, a function called while the stack is already
// being moved.
//
// Outside low level code it is not safe. A nosplit function cannot grow the
// stack, so a frame larger than the reserved budget overwrites the end of the
// stack: the result is memory corruption and arbitrary failures, not a panic.
package nosplit

// consume has a real frame, which is what makes the difference between a
// stack check and no stack check visible in the generated code. It is
// deliberately not inlinable.
//
//go:noinline
func consume(n int) int {
	var pad [32]byte
	pad[0] = byte(n)
	return int(pad[0])
}

// Critical runs without the usual stack growth preamble. It is the safe shape
// to copy: a small frame, and no calls to anything that can grow the stack
// again.
//
//go:nosplit
func Critical(n int) int {
	return consume(n)
}

// Expandable is the same function without the directive. The compiler emits
// the stack check and the call to runtime.morestack.
func Expandable(n int) int {
	return consume(n)
}

// CriticalNested is two nosplit frames deep. Each one skips its own check,
// which is exactly why the frames have to be budgeted by hand.
func CriticalNested(n int) int {
	return Critical(Critical(n))
}
