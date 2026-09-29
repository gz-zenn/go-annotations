package linkname_test

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/internal/gotool"
)

// The module used by the tests below links to runtime.morestack, an internal
// standard library symbol that nothing pushes for third party use, so the
// linker check of Go 1.23 and later rejects it. The flag turns the check off.
const blockedProgram = `package main

import _ "unsafe"

//go:linkname morestack runtime.morestack
func morestack()

// Taking the address keeps the reference alive without calling a function
// that must never be called by hand.
var sink = morestack

func main() { _ = sink }
`

func blockedModule(t *testing.T, extraFiles map[string]string) *gotool.Module {
	t.Helper()
	m := gotool.NewModule(t).File("main.go", blockedProgram)
	for name, content := range extraFiles {
		m.File(name, content)
	}
	return m
}

func TestLinknameCheckRejectsInternalStandardLibrarySymbols(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	m := blockedModule(t, nil)
	bin := filepath.Join(m.Dir, "app")
	m.Run(t, nil, "build", "-o", bin, ".").
		MustFail(t, "invalid reference to runtime.morestack")
}

func TestChecklinknameZeroDisablesTheCheck(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	m := blockedModule(t, nil)
	bin := filepath.Join(m.Dir, "app")
	m.Run(t, nil, "build", "-ldflags=-checklinkname=0", "-o", bin, ".").MustSucceed(t)
}

func TestLinknameCheckDoesNotCoverUserPackages(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// Symbols outside the standard library are not part of the check, which
	// is why the sibling-package example in linkname.go needs no flag. The
	// target must still use the full import path.
	m := gotool.NewModule(t).
		File("values/values.go", `package values

var Calls int

func add(a, b int) int {
	Calls++
	return a + b
}
`).
		File("lnk/lnk.go", `package lnk

import (
	_ "tmpmodule/values"
	_ "unsafe"
)

//go:linkname internalAdd tmpmodule/values.add
func internalAdd(a, b int) int

func Add(a, b int) int { return internalAdd(a, b) }
`).
		File("main.go", `package main

import (
	"fmt"

	"tmpmodule/lnk"
	"tmpmodule/values"
)

func main() {
	before := values.Calls
	fmt.Println(lnk.Add(2, 3) == 5, values.Calls == before+1)
}
`)

	bin := filepath.Join(m.Dir, "app")
	m.Run(t, nil, "build", "-o", bin, ".").MustSucceed(t)
}

func TestShortPackageNameDoesNotResolve(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// The mistake behind "relocation target values.add not defined": a
	// linkname target outside the standard library is the fully qualified
	// symbol, so the short package name is not enough. It fails at link
	// time, which is why the diagnostic mentions relocations.
	m := gotool.NewModule(t).
		File("values/values.go", `package values

func add(a, b int) int { return a + b }

func Add(a, b int) int { return add(a, b) }
`).
		File("lnk/lnk.go", `package lnk

import (
	_ "tmpmodule/values"
	_ "unsafe"
)

//go:linkname internalAdd values.add
func internalAdd(a, b int) int

func Add(a, b int) int { return internalAdd(a, b) }
`).
		File("main.go", `package main

import "tmpmodule/lnk"

func main() { _ = lnk.Add(1, 2) }
`)

	bin := filepath.Join(m.Dir, "app")
	m.Run(t, nil, "build", "-o", bin, ".").
		MustFail(t, "relocation target values.add not defined")
}

func TestMissingPackageIsALinkErrorNotACompileError(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// Without the blank import, nothing pulls the package into the binary
	// and the reference cannot be resolved.
	m := gotool.NewModule(t).
		File("values/values.go", `package values

func add(a, b int) int { return a + b }
`).
		File("lnk/lnk.go", `package lnk

import _ "unsafe"

//go:linkname internalAdd tmpmodule/values.add
func internalAdd(a, b int) int

func Add(a, b int) int { return internalAdd(a, b) }
`).
		File("main.go", `package main

import "tmpmodule/lnk"

func main() { _ = lnk.Add(1, 2) }
`)

	bin := filepath.Join(m.Dir, "app")
	m.Run(t, nil, "build", "-o", bin, ".").
		MustFail(t, "relocation target tmpmodule/values.add not defined")
}

func TestLinknameNeedsTheUnsafeImport(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	m := gotool.NewModule(t).
		File("values/values.go", `package values

func add(a, b int) int { return a + b }
`).
		File("lnk/lnk.go", `package lnk

//go:linkname internalAdd tmpmodule/values.add
func internalAdd(a, b int) int

func Add(a, b int) int { return internalAdd(a, b) }
`)

	m.Build(t).MustFail(t, "//go:linkname only allowed in Go files that import \"unsafe\"")
}

func TestTimeNowNeedsTheFlagOnOlderToolchains(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// The article's time.now example. Linking to it worked without any flag
	// up to Go 1.25, and Go 1.23 through 1.25 needed -checklinkname=0
	// because nothing on the definition side allowed the reference. Later
	// toolchains bless it explicitly, so both outcomes are valid here and the
	// test asserts on the diagnostic rather than on a fixed answer.
	bin := filepath.Join(t.TempDir(), "app")

	withFlag := gotool.Run(t, "..", nil, "build", "-ldflags=-checklinkname=0", "-o", bin, "./linkname/")
	withFlag.MustSucceed(t)

	withoutFlag := gotool.Run(t, "..", nil, "build", "-o", bin, "./linkname/")
	switch {
	case withoutFlag.Failed():
		withoutFlag.MustFail(t, "invalid reference to time.now")
	default:
		t.Log("this toolchain allows //go:linkname to time.now without -checklinkname=0")
	}
}

func TestChecklinknameFlagIsAvailable(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// Guards the tests above: if the flag were removed from the linker the
	// two of them would have to be deleted rather than quietly pass.
	res := gotool.Run(t, "..", nil, "tool", "link", "-h")
	res.MustContain(t, "-checklinkname")
}

func TestGoVersionSupportsTheCheck(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	res := gotool.Run(t, "..", nil, "list", "-m", "-f", "{{.GoVersion}}")
	res.MustSucceed(t)

	version := strings.TrimPrefix(strings.TrimSpace(res.Output), "go1.")
	major, _, _ := strings.Cut(version, ".")
	n, err := strconv.Atoi(major)
	if err != nil {
		t.Skipf("cannot parse the toolchain version %q", res.Output)
	}
	if n < 23 {
		t.Skipf("the linkname check landed in Go 1.23, this toolchain is %s", strings.TrimSpace(res.Output))
	}
}
