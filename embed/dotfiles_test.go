package embed_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/embed"
)

func TestDotfilePatternsFollowTheTable(t *testing.T) {
	report, err := embed.DotfileReport()
	if err != nil {
		t.Fatalf("DotfileReport: %v", err)
	}

	// The three patterns, over the same static/ directory. The first-level
	// dotfile is static/.nojekyll, the deeper one is static/sub/.env.
	tests := []struct {
		pattern    string
		want       []string
		wantAbsent []string
	}{
		{
			pattern: "static",
			want: []string{
				"static/css/app.css",
				"static/index.html",
				"static/js/app.js",
				"static/sub/nested.txt",
			},
			wantAbsent: []string{"static/.nojekyll", "static/sub/.env"},
		},
		{
			pattern: "static/*",
			want: []string{
				"static/.nojekyll",
				"static/css/app.css",
				"static/index.html",
				"static/js/app.js",
				"static/sub/nested.txt",
			},
			wantAbsent: []string{"static/sub/.env"},
		},
		{
			pattern: "all:static",
			want: []string{
				"static/.nojekyll",
				"static/css/app.css",
				"static/index.html",
				"static/js/app.js",
				"static/sub/.env",
				"static/sub/nested.txt",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.pattern, func(t *testing.T) {
			got, ok := report[tc.pattern]
			if !ok {
				t.Fatalf("report has no entry for %q, got %v", tc.pattern, keys(report))
			}

			joined := strings.Join(got, ",")
			for _, want := range tc.want {
				if !contains(got, want) {
					t.Errorf("%s is missing %q\ngot: %v", tc.pattern, want, got)
				}
			}
			for _, absent := range tc.wantAbsent {
				if contains(got, absent) {
					t.Errorf("%s embedded %q, want it excluded\ngot: %s", tc.pattern, absent, joined)
				}
			}
			if len(got) != len(tc.want) {
				t.Errorf("%s embedded %d files (%s), want %d", tc.pattern, len(got), joined, len(tc.want))
			}
		})
	}
}

func TestAllPrefixIsTheOnlyWayToReachTheDeeperDotfile(t *testing.T) {
	report, err := embed.DotfileReport()
	if err != nil {
		t.Fatalf("DotfileReport: %v", err)
	}

	// This is the row of the table that matters for secrets: static/sub/.env
	// is in the binary if and only if the pattern is all:static.
	onlyInAll := 0
	for _, path := range report["all:static"] {
		if strings.HasSuffix(path, ".env") {
			onlyInAll++
		}
	}
	if onlyInAll != 1 {
		t.Errorf("all:static contains %d .env files, want 1", onlyInAll)
	}

	for _, pattern := range []string{"static", "static/*"} {
		for _, path := range report[pattern] {
			if strings.HasSuffix(path, ".env") {
				t.Errorf("%s embedded %q, want dotfiles excluded", pattern, path)
			}
		}
	}
}

func TestPlainDirIsTheSameTreeAsAll(t *testing.T) {
	// Both variables cover the same directory; only the dotfile rule differs.
	report, err := embed.DotfileReport()
	if err != nil {
		t.Fatalf("DotfileReport: %v", err)
	}

	if len(report["all:static"]) <= len(report["static"]) {
		t.Errorf("all:static (%d files) should embed strictly more than static (%d files)",
			len(report["all:static"]), len(report["static"]))
	}
}

func contains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}

func keys(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
