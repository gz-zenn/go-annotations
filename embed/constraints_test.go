package embed_test

import (
	"testing"

	"github.com/gz-zenn/go-annotations/internal/gotool"
)

// The rules that are enforced by the compiler cannot be shown by a passing
// test, only by a failing build. Each case below hands a snippet to the real
// compiler in a throwaway module and checks the diagnostic.
func TestInvalidEmbedUsageIsRejected(t *testing.T) {
	gotool.SkipIfUnavailable(t)

	tests := []struct {
		name    string
		sources map[string]string
		want    string
	}{
		{
			name: "target must be string, []byte or embed.FS",
			sources: map[string]string{
				"main.go": `package main

import _ "embed"

//go:embed asset.txt
var v int

func main() { println(v) }
`,
			},
			want: "go:embed cannot apply to var of type int",
		},
		{
			name: "the variable must be package level",
			sources: map[string]string{
				"main.go": `package main

import _ "embed"

func load() string {
	//go:embed asset.txt
	var v string
	return v
}

func main() { println(load()) }
`,
			},
			want: "go:embed cannot apply to var inside func",
		},
		{
			name: "the embed package must be imported",
			sources: map[string]string{
				"main.go": `package main

//go:embed asset.txt
var v string

func main() { println(v) }
`,
			},
			want: `go:embed requires import "embed"`,
		},
		{
			name: "the pattern must match something",
			sources: map[string]string{
				"main.go": `package main

import _ "embed"

//go:embed missing/*.txt
var v string

func main() { println(v) }
`,
			},
			want: "no matching files found",
		},
		{
			name: "the pattern cannot escape the package directory",
			sources: map[string]string{
				"main.go": `package main

import _ "embed"

//go:embed ../secret.txt
var v string

func main() { println(v) }
`,
				"secret.txt": "do not embed me",
			},
			want: "invalid pattern syntax",
		},
		{
			name: "blank lines and // comments may sit between directive and var",
			sources: map[string]string{
				"main.go": `package main

import _ "embed"

//go:embed asset.txt
//
// an explanation between the directive and the variable is fine

var v string

func main() { println(v) }
`,
				"asset.txt": "still embedded",
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := gotool.NewModule(t).File("asset.txt", "content")
			for name, source := range tc.sources {
				m.File(name, source)
			}

			res := m.Build(t)
			if tc.want == "" {
				res.MustSucceed(t)
				return
			}
			res.MustFail(t, tc.want)
		})
	}
}

func TestAllPrefixWorksOnThisToolchain(t *testing.T) {
	// all: needs Go 1.18 or later. On 1.16 and 1.17 the prefix is unknown and
	// the build fails with "no matching files found".
	gotool.SkipIfUnavailable(t)

	m := gotool.NewModule(t).
		File("assets/.env", "SECRET=1").
		File("assets/sub/.env", "SECRET=2").
		File("main.go", `package main

import (
	"embed"
	"fmt"
)

//go:embed all:assets
var files embed.FS

func main() {
	data, err := files.ReadFile("assets/sub/.env")
	if err != nil {
		panic(err)
	}
	fmt.Print(string(data))
}
`)

	m.Build(t).MustSucceed(t)
}
