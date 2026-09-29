// Package rules covers the syntax rules that turn a //go: comment into a
// directive, and the placement rules that are conventions rather than
// syntax.
//
// The variants that would have to be written badly formatted (an indented
// directive, a //go:generate that is not at column 0) are built in throwaway
// modules by the tests instead of being checked in here, because gofmt
// rewrites them, which is the whole point of the rule.
package rules

// Proper is the normal shape: no space after //, at column 0.
//
//go:noinline
func Proper(a, b int) int {
	return a + b
}

// Spaced looks like a directive but is a comment: one space after // and it
// is text like any other. The compiler inlines this function.
//
// go:noinline
func Spaced(a, b int) int {
	return a + b
}

// BlankLine has a blank line between the directive and the declaration. The
// compiler does not care; the embed documentation only promises that blank
// lines and // comments may sit in between.
//
//go:noinline

func BlankLine(a, b int) int {
	return a + b
}
