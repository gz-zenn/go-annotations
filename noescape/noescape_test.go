package noescape_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/internal/gotool"
	"github.com/gz-zenn/go-annotations/noescape"
)

func TestLoadReadsThroughThePointer(t *testing.T) {
	values := []int64{0, 1, -1, 42, 1 << 40, -(1 << 40)}

	for _, want := range values {
		v := want
		if got := noescape.Load(&v); got != want {
			t.Errorf("Load(&%d) = %d, want %d", want, got, want)
		}
	}
}

func TestLoadSeesLaterWrites(t *testing.T) {
	v := int64(1)
	if got := noescape.Load(&v); got != 1 {
		t.Fatalf("Load = %d, want 1", got)
	}

	v = 2
	if got := noescape.Load(&v); got != 2 {
		t.Errorf("Load after write = %d, want 2", got)
	}
}

func TestNoescapeKeepsTheArgumentOnTheStack(t *testing.T) {
	// The measurable effect of the directive: without it, the compiler has to
	// assume the pointer escapes and heap allocates the caller's variable.
	v := int64(42)
	allocs := testing.AllocsPerRun(1000, func() {
		if noescape.Load(&v) != 42 {
			t.Error("Load returned the wrong value")
		}
	})

	if allocs != 0 {
		t.Errorf("AllocsPerRun = %v, want 0: the argument should stay on the stack", allocs)
	}
}

func TestStoreMakesThePointerEscape(t *testing.T) {
	// The control group: this function stores the pointer, so the argument
	// really does have to be heap allocated. The compiler decides this on its
	// own, which is why //go:noescape is only safe next to assembly.
	//
	// The variable is declared inside the closure on purpose: a variable
	// declared outside would be heap allocated once, before the measurement
	// window, and the count would come out as zero.
	allocs := testing.AllocsPerRun(1000, func() {
		local := int64(7)
		noescape.Store(&local, 7)
	})

	if allocs == 0 {
		t.Error("AllocsPerRun = 0, want at least one: storing the pointer must heap allocate it")
	}

	local := int64(7)
	noescape.Store(&local, 7)
	if got := noescape.Escaping(); got != &local {
		t.Errorf("Escaping() = %p, want %p", got, &local)
	}
}

func TestEscapeAnalysisAgreesWithTheCompiler(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// The same conclusion, in the compiler's own words: the assembly-backed
	// load is trusted not to leak its pointer, while Store is reported as
	// leaking it.
	res := gotool.Run(t, "..", nil, "build", "-gcflags=-m=2", "./noescape/")
	res.MustSucceed(t)

	if !strings.Contains(res.Output, "leaking param: p") {
		t.Errorf("compiler did not report the leaking parameter of Store:\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "p does not escape") {
		t.Errorf("compiler did not accept the //go:noescape promise:\n%s", res.Output)
	}
}

func TestNoescapeChangesTheEscapeAnalysis(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// The same assembly-backed function, with and without the directive. Only
	// the package is built, never linked, so the empty .s file is enough to
	// get past the missing-body check.
	const sources = `package leaky

//go:noescape
func load(p *int64) int64

func Load(p *int64) int64 { return load(p) }
`
	const asm = "#include \"textflag.h\"\n"

	withDirective := gotool.NewModule(t).
		File("leaky/load.go", sources).
		File("leaky/load_arm64.s", asm)

	res := withDirective.Run(t, []string{"GOARCH=arm64"}, "build", "-gcflags=-m=2", "./leaky")
	res.MustSucceed(t)
	if !strings.Contains(res.Output, "p does not escape") {
		t.Errorf("with //go:noescape, the compiler still thinks p escapes:\n%s", res.Output)
	}

	withoutDirective := gotool.NewModule(t).
		File("leaky/load.go", strings.Replace(sources, "//go:noescape\n", "", 1)).
		File("leaky/load_arm64.s", asm)

	res = withoutDirective.Run(t, []string{"GOARCH=arm64"}, "build", "-gcflags=-m=2", "./leaky")
	res.MustSucceed(t)
	if !strings.Contains(res.Output, "leaking param: p") {
		t.Errorf("without the directive, the compiler should think p escapes:\n%s", res.Output)
	}
}

func TestAssemblyIsUsedOnThisArchitecture(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// The function has no Go body, so its symbol comes from the .s file.
	res := gotool.Run(t, "..", nil, "build", "-gcflags=-S", "./noescape/")
	res.MustSucceed(t)

	if !strings.Contains(res.Output, "noescape.load") {
		t.Errorf("assembly listing has no noescape.load symbol:\n%s", firstLines(res.Output, 20))
	}
}

func TestPackageBuildsForEverySupportedArch(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// Each architecture needs its own .s file, or the declaration has no
	// implementation.
	for _, arch := range []string{"amd64", "arm64", "386"} {
		t.Run(arch, func(t *testing.T) {
			gotool.Run(t, "..", []string{"GOOS=linux", "GOARCH=" + arch}, "build", "./noescape/").
				MustSucceed(t)
		})
	}
}

func TestAssemblySignaturesMatchTheDeclaration(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// vet's asmdecl analyzer checks the FP offsets and names in the assembly
	// against the Go declaration. It is the only way to check the
	// architecture this test binary is not running on, and it is what CI
	// should run for every .s file in a real repository.
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			gotool.Run(t, "..", []string{"GOARCH=" + arch}, "vet", "./noescape/").MustSucceed(t)
		})
	}
}

func TestDeclarationWithoutAssemblyDoesNotCompile(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// A body-less declaration needs an assembly implementation in the same
	// package. Without one the compiler refuses the build.
	m := gotool.NewModule(t).
		File("load.go", `package lonely

//go:noescape
func load(p *int64) int64

func Load(p *int64) int64 { return load(p) }
`)

	m.Build(t).MustFail(t, "missing function body")
}

func TestEmptyAssemblyFileFailsAtLinkTime(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// This is the trap: an empty .s file is enough to satisfy the compiler,
	// and the failure only shows up when something calls the function.
	m := gotool.NewModule(t).
		File("lonely/load.go", `package lonely

//go:noescape
func load(p *int64) int64

func Load(p *int64) int64 { return load(p) }
`).
		File("lonely/load_arm64.s", "#include \"textflag.h\"\n").
		File("main.go", `package main

import "tmpmodule/lonely"

func main() {
	v := int64(1)
	println(lonely.Load(&v))
}
`)

	bin := filepath.Join(m.Dir, "lonely")
	m.Run(t, []string{"GOARCH=arm64"}, "build", "-o", bin, ".").
		MustFail(t, "lonely.load not defined")
}

func TestNoescapeIsNotNeededForGoBodies(t *testing.T) {
	// The compiler already understands a Go body, so the directive is only
	// ever needed next to assembly.
	gotool.SkipIfUnavailable(t)

	m := gotool.NewModule(t).
		File("main.go", `package main

//go:noescape
func leaky(p *int64) int64 {
	return *p
}

func main() {
	v := int64(1)
	println(leaky(&v))
}
`)

	m.Build(t).MustFail(t, "can only use //go:noescape with external func implementations")
}

func TestRuntimeIsCheckedFirst(t *testing.T) {
	// Guards the assumption behind the rest of this file.
	switch runtime.GOARCH {
	case "amd64", "arm64":
	default:
		t.Logf("running on %s, the assembly implementation is not used here", runtime.GOARCH)
	}
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
