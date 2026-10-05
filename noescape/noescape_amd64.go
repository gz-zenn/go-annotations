//go:build amd64

package noescape

// load reads the value pointed to by p. The body lives in noescape_amd64.s,
// because a function declaration without a body must be implemented in
// assembly in the same package.
//
//go:noescape
func load(p *int64) int64
