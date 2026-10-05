//go:build !linux && !darwin && !windows

package buildtags

// OSName is the fallback for every other GOOS, so the package still builds
// for targets the two files above do not cover. A constraint expression
// negates the others with &&, not ||.
func OSName() string {
	return "other"
}

// BuildTag reports the constraint that selected this file.
func BuildTag() string {
	return "!linux && !darwin && !windows"
}
