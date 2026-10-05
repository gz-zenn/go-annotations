package buildtags_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/buildtags"
	"github.com/gz-zenn/go-annotations/internal/gotool"
)

func TestOSNameMatchesTheRunningPlatform(t *testing.T) {
	want := map[string]string{
		"linux":   "unix-like",
		"darwin":  "unix-like",
		"windows": "windows",
	}

	expected, known := want[runtime.GOOS]
	if !known {
		expected = "other"
	}

	if got := buildtags.OSName(); got != expected {
		t.Errorf("OSName() = %q on %s, want %q", got, runtime.GOOS, expected)
	}
}

func TestBuildTagMatchesTheCompiledFile(t *testing.T) {
	want := map[string]string{
		"linux":   "linux || darwin",
		"darwin":  "linux || darwin",
		"windows": "windows",
	}

	expected, known := want[runtime.GOOS]
	if !known {
		expected = "!linux && !darwin && !windows"
	}

	if got := buildtags.BuildTag(); got != expected {
		t.Errorf("BuildTag() = %q on %s, want %q", got, runtime.GOOS, expected)
	}
}

// listedFiles asks the go tool which files of this package it would compile
// for the given environment.
func listedFiles(t *testing.T, env []string, args ...string) string {
	t.Helper()
	gotool.SkipIfUnavailable(t)

	full := append([]string{"list", "-f", "{{join .GoFiles \" \"}}"}, args...)
	full = append(full, ".")

	res := gotool.Run(t, ".", env, full...)
	res.MustSucceed(t)
	return res.Output
}

func TestBuildConstraintsSelectTheRightFile(t *testing.T) {
	tests := []struct {
		name    string
		env     []string
		want    string
		notWant string
	}{
		{
			name:    "linux amd64 picks the posix file",
			env:     []string{"GOOS=linux", "GOARCH=amd64"},
			want:    "platform_posix.go",
			notWant: "platform_win.go",
		},
		{
			name:    "darwin arm64 picks the posix file",
			env:     []string{"GOOS=darwin", "GOARCH=arm64"},
			want:    "platform_posix.go",
			notWant: "platform_win.go",
		},
		{
			name:    "windows amd64 picks the windows file",
			env:     []string{"GOOS=windows", "GOARCH=amd64"},
			want:    "platform_win.go",
			notWant: "platform_posix.go",
		},
		{
			name:    "an unlisted target falls back to the negated file",
			env:     []string{"GOOS=plan9", "GOARCH=amd64"},
			want:    "platform_other.go",
			notWant: "platform_win.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files := listedFiles(t, tc.env)

			if !strings.Contains(files, tc.want) {
				t.Errorf("compiled files = %q, want it to include %q", files, tc.want)
			}
			if strings.Contains(files, tc.notWant) {
				t.Errorf("compiled files = %q, want it to exclude %q", files, tc.notWant)
			}
		})
	}
}

func TestCustomTagGatesTheFile(t *testing.T) {
	without := listedFiles(t, nil)
	if strings.Contains(without, "integration.go") {
		t.Errorf("integration.go compiled without the tag: %q", without)
	}

	with := listedFiles(t, nil, "-tags=integration")
	if !strings.Contains(with, "integration.go") {
		t.Errorf("compiled files = %q, want it to include integration.go", with)
	}
}

func TestCustomTagGatesTheTestFile(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	res := gotool.Run(t, ".", nil, "list", "-f", "{{join .TestGoFiles \" \"}}", "-tags=integration", ".")
	res.MustSucceed(t)
	if !strings.Contains(res.Output, "integration_test.go") {
		t.Errorf("test files with the tag = %q, want it to include integration_test.go", res.Output)
	}

	res = gotool.Run(t, ".", nil, "list", "-f", "{{join .TestGoFiles \" \"}}", ".")
	res.MustSucceed(t)
	if strings.Contains(res.Output, "integration_test.go") {
		t.Errorf("test files without the tag = %q, want it to exclude integration_test.go", res.Output)
	}
}

func TestPackageCompilesForEveryTarget(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// The constraint is what makes the two implementations coexist; proving
	// it is enough to compile the package for each target.
	for _, env := range [][]string{
		{"GOOS=linux", "GOARCH=amd64"},
		{"GOOS=linux", "GOARCH=arm64"},
		{"GOOS=darwin", "GOARCH=arm64"},
		{"GOOS=windows", "GOARCH=amd64"},
		{"GOOS=windows", "GOARCH=arm64"},
		{"GOOS=plan9", "GOARCH=amd64"},
	} {
		t.Run(strings.Join(env, "/"), func(t *testing.T) {
			gotool.Run(t, "..", env, "build", "./buildtags/").MustSucceed(t)
		})
	}
}

func TestLegacyBuildLineIsStillHonoured(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	m := gotool.NewModule(t).
		File("always.go", "package legacy\n\nfunc Present() bool { return true }\n").
		File("legacy.go", "// +build linux\n\npackage legacy\n\nfunc Which() string { return \"linux\" }\n")

	// The old syntax is deprecated, not dead: the toolchain still reads it,
	// which is why a file written years ago keeps building for the platform
	// it was written for.
	res := m.List(t, "{{join .GoFiles \" \"}}", "GOOS=linux", "GOARCH=amd64")
	res.MustSucceed(t)
	if !strings.Contains(res.Output, "legacy.go") {
		t.Errorf("GoFiles on linux = %q, want it to include legacy.go", res.Output)
	}

	res = m.List(t, "{{join .GoFiles \" \"}}", "GOOS=darwin", "GOARCH=arm64")
	res.MustSucceed(t)
	if strings.Contains(res.Output, "legacy.go") {
		t.Errorf("GoFiles on darwin = %q, want legacy.go to be excluded", res.Output)
	}
}

func TestGoBuildLineWinsOverLegacyLine(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// When both lines are present and disagree, //go:build is authoritative.
	m := gotool.NewModule(t).
		File("always.go", "package conflict\n\nfunc Present() bool { return true }\n").
		File("conflict.go", "//go:build windows\n// +build linux\n\npackage conflict\n\nfunc Which() string { return \"windows\" }\n")

	res := m.List(t, "{{join .GoFiles \" \"}}", "GOOS=linux", "GOARCH=amd64")
	res.MustSucceed(t)
	if strings.Contains(res.Output, "conflict.go") {
		t.Errorf("GoFiles on linux = %q, want conflict.go to be excluded by //go:build windows", res.Output)
	}

	res = m.List(t, "{{join .GoFiles \" \"}}", "GOOS=windows", "GOARCH=amd64")
	res.MustSucceed(t)
	if !strings.Contains(res.Output, "conflict.go") {
		t.Errorf("GoFiles on windows = %q, want it to include conflict.go", res.Output)
	}
}
