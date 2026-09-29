//go:build !amd64 && !arm64

package noescape

// load is the portable fallback for architectures without an assembly
// implementation in this example. The compiler can see this body, so escape
// analysis works out the same answer on its own and the directive would be
// redundant.
func load(p *int64) int64 {
	return *p
}
