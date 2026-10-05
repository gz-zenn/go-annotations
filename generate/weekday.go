// Package generate demonstrates the //go:generate directive: a hook that
// runs external commands from the go tool. It has no effect on compilation,
// it only tells "go generate" what to run.
//
// Run it with:
//
//	go generate ./...
//
// The generated files are checked in, exactly as they are in real projects:
// the commit contains the code that was produced, so building the module
// never depends on having the generators installed.
package generate

// Weekday is an enumerated type; the directive below asks stringer to write a
// String method for it into weekday_string.go.
//
//go:generate stringer -type=Weekday
type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)
