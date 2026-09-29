//go:build windows

package buildtags

// OSName is compiled on Windows only. The file is deliberately not named
// platform_windows.go: a _windows filename suffix is an implicit constraint
// and this example is about the explicit one.
func OSName() string {
	return "windows"
}

// BuildTag reports the constraint that selected this file.
func BuildTag() string {
	return "windows"
}
