package linkname

import _ "unsafe"

// This is the pattern the standard library itself uses to share internals:
// reach an unexported symbol of another package. time.now is what time.Now
// calls, without the exported wrapper around it.
//
// Because //go:linkname marks the declaration as an external symbol, this
// file builds with no assembly at all, unlike //go:noescape.
//
//go:linkname now time.now
func now() (sec int64, nsec int32, mono int64)

// Now returns the wall clock the way the runtime reports it.
func Now() (sec int64, nsec int32, mono int64) {
	return now()
}
