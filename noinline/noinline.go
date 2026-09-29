// Package noinline demonstrates //go:noinline, the directive that keeps the
// compiler from inlining a function.
//
// Confirm the effect with:
//
//	go build -gcflags=-m=2 ./noinline
//
// The verbose level matters: plain -m only prints the functions the compiler
// can inline, so it lists AddInlinable and stays silent about Add. At -m=2
// the output also explains the refusals, and contains a line like
//
//	noinline/noinline.go:20:6: cannot inline Add: marked go:noinline
package noinline

// Add is a trivial function that the compiler would happily inline into
// every caller. Benchmarks use //go:noinline when the call overhead is what
// is being measured, and debuggers need it to get a real stack frame.
//
//go:noinline
func Add(a, b int) int {
	return a + b
}

// AddInlinable is the same function without the directive. It is the control
// group the tests compare against.
func AddInlinable(a, b int) int {
	return a + b
}
