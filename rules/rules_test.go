package rules_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/internal/gotool"
	"github.com/gz-zenn/go-annotations/rules"
)

// inlineReport asks the compiler which of the named functions it can inline.
func inlineReport(t *testing.T, names ...string) string {
	t.Helper()
	gotool.SkipIfUnavailable(t)

	res := gotool.Run(t, "..", nil, "build", "-gcflags=-m=2", "./rules/")
	res.MustSucceed(t)

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

func TestNoSpaceAfterSlashesIsADirective(t *testing.T) {
	report := inlineReport(t, "Proper")

	if !strings.Contains(report, "cannot inline Proper:") {
		t.Errorf("//go:noinline was not honoured:\n%s", report)
	}
}

func TestOneSpaceAfterSlashesIsJustAComment(t *testing.T) {
	// The only difference from Proper is a single space, and it turns the
	// directive back into a comment. Nothing warns about it.
	report := inlineReport(t, "Spaced")

	if !strings.Contains(report, "can inline Spaced") {
		t.Errorf("the spaced comment was treated as a directive:\n%s", report)
	}
	if strings.Contains(report, "cannot inline Spaced") {
		t.Errorf("\"// go:noinline\" was honoured as a directive:\n%s", report)
	}
}

func TestBlankLineDoesNotBreakTheDirective(t *testing.T) {
	report := inlineReport(t, "BlankLine")

	if !strings.Contains(report, "cannot inline BlankLine:") {
		t.Errorf("the blank line detached the directive:\n%s", report)
	}
}

func TestAllThreeStillComputeTheSameThing(t *testing.T) {
	for _, tc := range []struct{ a, b, want int }{{0, 0, 0}, {2, 3, 5}, {-1, 1, 0}} {
		if got := rules.Proper(tc.a, tc.b); got != tc.want {
			t.Errorf("Proper(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
		if got := rules.Spaced(tc.a, tc.b); got != tc.want {
			t.Errorf("Spaced(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
		if got := rules.BlankLine(tc.a, tc.b); got != tc.want {
			t.Errorf("BlankLine(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestIndentationDoesNotDisableADirective(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// Leading whitespace before // is fine: what the compiler looks at is the
	// text right after the slashes.
	m := gotool.NewModule(t).
		File("indented.go", `package indented

// Indented is documented.
	//go:noinline
func Indented(a, b int) int { return a + b }
`)

	res := m.Run(t, nil, "build", "-gcflags=-m=2", ".")
	res.MustSucceed(t)
	res.MustContain(t, "cannot inline Indented:")
}

func TestGofmtMovesAnIndentedDirectiveBackToColumnZero(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// gofmt, not the compiler, is what enforces the column. This is why the
	// examples in this module are all written at column 0.
	m := gotool.NewModule(t).
		File("indented.go", `package indented

// Indented is documented.
	//go:noinline
func Indented(a, b int) int { return a + b }
`)
	m.Write(t)

	res := gotool.Gofmt(t, m.Dir)
	res.MustSucceed(t)
	if !strings.Contains(res.Output, "indented.go") {
		t.Errorf("gofmt reported nothing, want it to flag the indented directive:\n%s", res.Output)
	}

	// And the files this repository checks in are gofmt clean.
	res = gotool.Gofmt(t, "..")
	res.MustSucceed(t)
	for _, file := range strings.Fields(res.Output) {
		if strings.HasPrefix(file, "rules/") {
			t.Errorf("gofmt would rewrite %s", file)
		}
	}
}

// generator writes one line to marker.txt in the working directory, so a
// test can tell whether go generate ran a directive at all.
const generator = `// Command gen leaves a marker behind.
package main

import (
	"fmt"
	"os"
)

func main() {
	f, err := os.OpenFile("marker.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintln(f, "generated")
}
`

// generateDirective is spelled in two pieces on purpose: go generate scans
// source files line by line, so a literal //go:generate line in this file
// would be executed when someone runs "go generate ./..." over the module.
const generateDirective = "//go:" + "generate go run ./gen"

func TestGoGenerateOnlyRunsAtColumnZero(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// //go:generate really must start at column 0, unlike the other
	// directives. Two directives in the same package, one indented, produce
	// exactly one marker.
	m := gotool.NewModule(t).
		File("gen/gen.go", generator).
		File("at_column_zero.go", "package generate\n\n"+generateDirective+"\n\nfunc AtColumnZero() {}\n").
		File("indented.go", "package generate\n\n// Indented is documented.\n\t"+generateDirective+"\n\nfunc Indented() {}\n")

	m.Generate(t).MustSucceed(t)

	if got := markerLines(t, m.Dir); got != 1 {
		t.Errorf("marker.txt has %d lines, want 1: only the column 0 directive is a directive", got)
	}
}

func TestGoGenerateSkippedEntirelyWhenIndented(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	m := gotool.NewModule(t).
		File("gen/gen.go", generator).
		File("indented.go", "package generate\n\n\t"+generateDirective+"\n\nfunc Indented() {}\n")

	m.Generate(t).MustSucceed(t)

	if _, err := os.Stat(filepath.Join(m.Dir, "marker.txt")); !os.IsNotExist(err) {
		t.Error("marker.txt exists, want go generate to have ignored the indented directive")
	}
}

func TestGoGenerateScansRawTextNotCode(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// go generate does not parse the file. It looks for lines that begin
	// with the directive, so a directive sitting inside a string literal
	// still runs. The module below would never notice at build time.
	m := gotool.NewModule(t).
		File("gen/gen.go", generator).
		File("inside_a_string.go", "package inside\n\nvar s = `\n"+generateDirective+"\n`\n\nfunc S() string { return s }\n")

	m.Generate(t).MustSucceed(t)

	if got := markerLines(t, m.Dir); got != 1 {
		t.Errorf("marker.txt has %d lines, want 1: the line inside the string was scanned", got)
	}
}

func TestGoBuildAfterThePackageClauseIsAnError(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// The one placement rule that is not a convention. The file does not
	// compile, and it is a compile error, not a warning.
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "after the package clause",
			source: `package misplaced

//go:build linux

func Which() string { return "linux" }
`,
		},
		{
			name: "at the end of the file",
			source: `package misplaced

func Which() string { return "linux" }

//go:build linux
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := gotool.NewModule(t).File("misplaced.go", tc.source)
			m.Build(t).MustFail(t, "misplaced compiler directive")
		})
	}
}

// goBuildModule builds a two file module whose second file carries the given
// //go:build line. The constraint has to sit before the package clause, so
// the source is assembled rather than written literally.
func goBuildModule(t *testing.T, buildLine string) *gotool.Module {
	t.Helper()

	return gotool.NewModule(t).
		File("always.go", "package tagged\n\nfunc Present() bool { return true }\n").
		File("tagged.go", buildLine+"\n\npackage tagged\n\nfunc Which() string { return \"linux\" }\n")
}

func TestGoBuildBeforeThePackageClauseWorks(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	m := goBuildModule(t, "//go:build linux")

	res := m.List(t, "{{join .GoFiles \" \"}}", "GOOS=linux", "GOARCH=amd64")
	res.MustSucceed(t)
	if !strings.Contains(res.Output, "tagged.go") {
		t.Errorf("GoFiles on linux = %q, want tagged.go included", res.Output)
	}

	res = m.List(t, "{{join .GoFiles \" \"}}", "GOOS=darwin", "GOARCH=amd64")
	res.MustSucceed(t)
	if strings.Contains(res.Output, "tagged.go") {
		t.Errorf("GoFiles on darwin = %q, want tagged.go excluded", res.Output)
	}
}

func TestIndentationDoesNotDisableAGoBuildConstraint(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// The go command finds the line wherever the slashes are; the column is a
	// formatting rule, not a parsing rule. Note how different this is from
	// //go:generate below, where the column is enforced.
	m := goBuildModule(t, "	//go:build linux")

	res := m.List(t, "{{join .GoFiles \" \"}}", "GOOS=linux", "GOARCH=amd64")
	res.MustSucceed(t)
	if !strings.Contains(res.Output, "tagged.go") {
		t.Errorf("GoFiles on linux = %q, want tagged.go included", res.Output)
	}

	res = m.List(t, "{{join .GoFiles \" \"}}", "GOOS=darwin", "GOARCH=amd64")
	res.MustSucceed(t)
	if strings.Contains(res.Output, "tagged.go") {
		t.Errorf("GoFiles on darwin = %q, want tagged.go excluded", res.Output)
	}
}

func TestGofmtMovesAnIndentedGoBuildToColumnZero(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	m := goBuildModule(t, "	//go:build linux")
	m.Write(t)

	res := gotool.Gofmt(t, m.Dir)
	res.MustSucceed(t)
	if !strings.Contains(res.Output, "tagged.go") {
		t.Errorf("gofmt reported nothing, want it to flag the indented constraint:\n%s", res.Output)
	}
}

func TestDirectiveMustBeOnItsOwnLine(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// A directive at the end of a line of code is misplaced, not ignored.
	m := gotool.NewModule(t).File("trailing.go", `package trailing

func f() int { return 1 } //go:noinline
`)

	m.Build(t).MustFail(t, "misplaced compiler directive")
}

func markerLines(t *testing.T, dir string) int {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, "marker.txt"))
	if err != nil {
		t.Fatalf("reading marker.txt: %v", err)
	}
	return strings.Count(strings.TrimSpace(string(data)), "\n") + 1
}
