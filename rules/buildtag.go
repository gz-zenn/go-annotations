//go:build linux || darwin

package rules

// IndentedBuildTag is compiled for Linux and macOS only.
func IndentedBuildTag() string { return "posix" }
