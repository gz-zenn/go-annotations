// Package gotool runs the real Go toolchain against throwaway modules.
//
// Several claims in the article can only be checked by handing a broken
// snippet to the compiler or the linker, which is exactly what this helper
// does: it materialises a module in a temp directory and runs a go command
// in it, so tests can assert on real compiler and linker diagnostics.
package gotool

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Available reports whether a usable go toolchain is in PATH.
func Available() bool {
	_, err := exec.LookPath("go")
	return err == nil
}

// SkipIfUnavailable skips t when no go toolchain is in PATH.
func SkipIfUnavailable(t *testing.T) {
	t.Helper()
	if !Available() {
		t.Skip("go toolchain not available in PATH")
	}
}

// Result holds the outcome of a toolchain invocation.
type Result struct {
	Output string
	Err    error
}

// Failed reports whether the command exited with a non-zero status.
func (r Result) Failed() bool { return r.Err != nil }

// Module is a throwaway module rooted in a temporary directory.
type Module struct {
	Dir   string
	files map[string][]byte
}

// NewModule creates an empty module in a temp directory that is removed when
// the test finishes.
func NewModule(t *testing.T) *Module {
	t.Helper()
	dir := t.TempDir()
	m := &Module{Dir: dir, files: map[string][]byte{}}
	m.File("go.mod", "module tmpmodule\n\ngo 1.23\n")
	return m
}

// File queues a file to be written to the module.
func (m *Module) File(name, content string) *Module {
	m.files[name] = []byte(content)
	return m
}

// Bytes queues a file with raw contents.
func (m *Module) Bytes(name string, content []byte) *Module {
	m.files[name] = content
	return m
}

// Write materialises every queued file plus go.mod on disk.
func (m *Module) Write(t *testing.T) {
	t.Helper()
	for name, content := range m.files {
		full := filepath.Join(m.Dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
}

// Run executes a go command inside the module with the given extra
// environment (as "KEY=VALUE" strings) added on top of the current one.
func (m *Module) Run(t *testing.T, env []string, args ...string) Result {
	t.Helper()
	m.Write(t)
	return Run(t, m.Dir, env, args...)
}

// Run executes a go command in an arbitrary directory. It is handy for
// commands that must run against this repository, such as "go generate -n".
func Run(t *testing.T, dir string, env []string, args ...string) Result {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return Result{Output: buf.String(), Err: err}
}

// Gofmt runs the gofmt binary that belongs to the toolchain in use, so the
// checks do not depend on the one installed in PATH. It lists the files
// gofmt would rewrite, like "gofmt -l".
func Gofmt(t *testing.T, dir string, env ...string) Result {
	t.Helper()

	binary := gofmtBinary(t)
	if binary == "" {
		t.Skip("gofmt was not found for this toolchain")
	}

	cmd := exec.Command(binary, "-l", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return Result{Output: buf.String(), Err: err}
}

func gofmtBinary(t *testing.T) string {
	t.Helper()

	res := Run(t, ".", nil, "env", "GOROOT")
	if res.Failed() {
		return ""
	}
	candidate := filepath.Join(strings.TrimSpace(res.Output), "bin", "gofmt")
	if _, err := os.Stat(candidate); err != nil {
		// Fall back to whatever is in PATH, which is the gofmt that
		// go install leaves behind on some systems.
		if found, err := exec.LookPath("gofmt"); err == nil {
			return found
		}
		return ""
	}
	return candidate
}

// Build runs "go build" on every package in the module.
func (m *Module) Build(t *testing.T, env ...string) Result {
	return m.Run(t, env, "build", "./...")
}

// List runs "go list" with the given template on the module root package.
// env entries are "KEY=VALUE" strings.
func (m *Module) List(t *testing.T, template string, env ...string) Result {
	return m.Run(t, env, "list", "-f", template, ".")
}

// Vet runs "go vet" on every package in the module.
func (m *Module) Vet(t *testing.T, env ...string) Result {
	return m.Run(t, env, "vet", "./...")
}

// Generate runs "go generate" on every package in the module.
func (m *Module) Generate(t *testing.T, env ...string) Result {
	return m.Run(t, env, "generate", "./...")
}

// MustFail fails the test unless the command failed and its output contains
// every one of the given substrings.
func (r Result) MustFail(t *testing.T, wantSubstrings ...string) {
	t.Helper()
	if !r.Failed() {
		t.Fatalf("expected failure, got success\noutput:\n%s", r.Output)
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(r.Output, want) {
			t.Errorf("output does not contain %q\noutput:\n%s", want, r.Output)
		}
	}
}

// MustSucceed fails the test unless the command succeeded.
func (r Result) MustSucceed(t *testing.T) {
	t.Helper()
	if r.Failed() {
		t.Fatalf("expected success, got %v\noutput:\n%s", r.Err, r.Output)
	}
}

// MustContain fails the test unless the output contains want.
func (r Result) MustContain(t *testing.T, want string) {
	t.Helper()
	if !strings.Contains(r.Output, want) {
		t.Errorf("output does not contain %q\noutput:\n%s", want, r.Output)
	}
}
