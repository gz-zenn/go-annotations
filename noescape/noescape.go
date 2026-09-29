// Package noescape demonstrates //go:noescape, the directive that tells the
// compiler a function does not let its pointer arguments escape.
//
// It is meant for functions implemented in assembly, where the compiler
// cannot see the body: it stores nothing that outlives the call and returns
// no pointer derived from the argument. The declaration therefore has no
// body, and the package must carry the assembly that implements it, or the
// build fails.
//
// The directive is a promise to the compiler. Lying here does not produce an
// error, it produces memory corruption.
package noescape

// Load reads the value pointed to by p through the assembly implementation.
func Load(p *int64) int64 {
	return load(p)
}

// Store writes v through p and keeps the pointer in a package-level
// variable, so p escapes for real. This function is deliberately NOT marked
// //go:noescape: the directive would be a lie.
func Store(p *int64, v int64) {
	escaping = p
	*escaping = v
}

// escaping exists so Store has a reason to keep its argument alive.
var escaping *int64

// Escaping returns the pointer the last call to Store received. It makes the
// escape visible to the tests.
func Escaping() *int64 {
	return escaping
}
