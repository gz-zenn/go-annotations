//go:build arm64

package noescape

// load reads the value pointed to by p. The body lives in noescape_arm64.s.
//
//go:noescape
func load(p *int64) int64
