package generate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/internal/gotool"
)

func TestGoGenerateDiscoversBothDirectives(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	// -n lists the commands without running them. Both directives are at
	// column 0 in this package, which is the only strict placement rule for
	// //go:generate.
	res := gotool.Run(t, ".", nil, "generate", "-n", "./...")
	res.MustSucceed(t)

	for _, want := range []string{
		"stringer -type=Weekday",
		"mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks",
	} {
		if !strings.Contains(res.Output, want) {
			t.Errorf("go generate -n output missing %q\noutput:\n%s", want, res.Output)
		}
	}
}

func TestGeneratedFilesAreCheckedIn(t *testing.T) {
	// Generators are not part of the build, so their output must be
	// committed. Building this module from a clean checkout depends on it.
	for _, name := range []string{"weekday_string.go", filepath.Join("mocks", "service_mock.go")} {
		if _, err := os.Stat(name); err != nil {
			t.Errorf("generated file %s is missing: %v", name, err)
		}
	}
}
