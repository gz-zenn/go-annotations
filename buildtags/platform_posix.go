//go:build linux || darwin

package buildtags

// OSName is compiled on Linux and macOS only.
func OSName() string {
	return "unix-like"
}

// BuildTag reports the constraint that selected this file.
func BuildTag() string {
	return "linux || darwin"
}
