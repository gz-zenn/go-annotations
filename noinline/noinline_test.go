package noinline_test

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/internal/gotool"
	"github.com/gz-zenn/go-annotations/noinline"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{0, 0, 0},
		{2, 3, 5},
		{-4, 4, 0},
		{-7, -8, -15},
	}

	for _, tc := range tests {
		if got := noinline.Add(tc.a, tc.b); got != tc.want {
			t.Errorf("Add(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

// inlineDecisions asks the compiler for its inlining report and returns the
// lines that mention the named functions.
func inlineDecisions(t *testing.T, names ...string) string {
	t.Helper()
	gotool.SkipIfUnavailable(t)

	// -m=2 is the verbose level: it explains why a function cannot be
	// inlined, which is what the //go:noinline line looks like.
	res := gotool.Run(t, "..", nil, "build", "-gcflags=-m=2", "./noinline/")
	res.MustSucceed(t)

	// \b keeps "Add" from matching the "AddInlinable" line.
	patterns := make([]*regexp.Regexp, len(names))
	for i, name := range names {
		patterns[i] = regexp.MustCompile(`inline ` + name + `\b`)
	}

	var lines []string
	for _, line := range strings.Split(res.Output, "\n") {
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				lines = append(lines, strings.TrimSpace(line))
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

func TestNoinlineStopsInlining(t *testing.T) {
	report := inlineDecisions(t, "Add")

	if !strings.Contains(report, "cannot inline Add:") {
		t.Errorf("compiler report has no \"cannot inline Add\" line:\n%s", report)
	}
	if !strings.Contains(strings.ToLower(report), "noinline") {
		t.Errorf("compiler report does not blame the go:noinline directive:\n%s", report)
	}
}

func TestFunctionWithoutTheDirectiveIsInlined(t *testing.T) {
	report := inlineDecisions(t, "AddInlinable")

	if !strings.Contains(report, "can inline AddInlinable") {
		t.Errorf("compiler report has no \"can inline AddInlinable\" line:\n%s", report)
	}
	if strings.Contains(report, "cannot inline AddInlinable") {
		t.Errorf("compiler refused to inline AddInlinable:\n%s", report)
	}
}

func TestBothFunctionsBehaveIdentically(t *testing.T) {
	// The directive changes how the code is compiled, never what it
	// computes.
	for _, tc := range [][2]int{{0, 1}, {1, 1}, {10, -3}, {-8, -9}} {
		got := noinline.Add(tc[0], tc[1])
		want := noinline.AddInlinable(tc[0], tc[1])
		if got != want {
			t.Errorf("Add(%d, %d) = %d, AddInlinable = %d", tc[0], tc[1], got, want)
		}
	}
}

func BenchmarkAdd(b *testing.B) {
	// //go:noinline is what makes this a call, so the measurement includes
	// the function call and not just the addition.
	for i := 0; i < b.N; i++ {
		_ = noinline.Add(1, 2)
	}
}

func BenchmarkAddInlinable(b *testing.B) {
	// The loop body is the addition itself: the call was inlined away.
	for i := 0; i < b.N; i++ {
		_ = noinline.AddInlinable(1, 2)
	}
}

func ExampleAdd() {
	fmt.Println(noinline.Add(2, 3))
	// Output: 5
}

func TestInliningReportIsStableAcrossRuns(t *testing.T) {
	// Guards the helper itself: if the -m output ever stops mentioning the
	// functions, the tests above would pass for the wrong reason.
	first := inlineDecisions(t, "Add", "AddInlinable")
	second := inlineDecisions(t, "Add", "AddInlinable")

	if first == "" {
		t.Fatal("no inlining decisions were reported at all")
	}
	if first != second {
		t.Errorf("inlining report is not reproducible:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if runtime.Compiler != "gc" {
		t.Logf("note: tests were run with the %q compiler, not gc", runtime.Compiler)
	}
}
