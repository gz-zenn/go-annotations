// Package linkname demonstrates //go:linkname, which lets one package
// reference a symbol another package does not export.
//
// It requires importing unsafe, and it is the escape hatch the standard
// library itself uses to share internals. Since Go 1.23 the linker rejects
// references into standard library internals unless the definition side
// allows them; -checklinkname=0 turns the check off.
//
// The target of a pull linkname is the full symbol name, which for a package
// outside the standard library means the full import path, not the short
// package name. See stdlib.go for the time.now example.
package linkname

import (
	// A blank import is what puts the values package into the binary. A
	// symbol that is not linked in cannot be resolved, and that failure
	// shows up at link time ("relocation target ... not defined"), not at
	// compile time.
	_ "github.com/gz-zenn/go-annotations/linkname/values"
	_ "unsafe"
)

// The short form, values.add, does not resolve: the symbol in the object file
// is the fully qualified one. The linker reports a relocation error instead
// of a compile error when the name is wrong.
//
//go:linkname internalAdd github.com/gz-zenn/go-annotations/linkname/values.add
func internalAdd(a, b int) int

// Add calls the unexported function in the sibling package through its
// symbol name. It compiles and runs exactly like a normal call.
func Add(a, b int) int {
	return internalAdd(a, b)
}
