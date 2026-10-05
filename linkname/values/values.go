package values

// Calls counts the calls that went through add. It is how the linkname
// example proves the symbol it references is the real one.
var Calls int

// add is deliberately unexported: nothing outside this package is supposed to
// call it. The linkname example calls it anyway, by symbol name.
func add(a, b int) int {
	Calls++
	return a + b
}

// Add is the exported way in, kept so the example has a control to compare
// against.
func Add(a, b int) int {
	return add(a, b)
}
