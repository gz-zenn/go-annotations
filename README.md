# Go Annotations: A Practical Guide to //go: Directives

Go doesn't have annotations in the Java or C# sense — no `@Override` or `[Serializable]`. Instead, it has **compiler directives**: specially formatted comments that start with `//go:` and talk directly to the compiler, linker, or `go` tool. They look like ordinary comments, but they change how your code is built, linked, or documented.

This article walks through the directives you'll actually encounter, with working examples.

## The Rules That Make Them "Directives" Instead of Comments

A `//go:` comment is only treated as a directive if there is **no space between `//` and `go:`**.

```go
//go:noinline          // ✅ directive
// go:noinline         // ❌ just a comment (space after //)
```

Two other things are often stated as rules, but they are really conventions:

- **Column 0.** The compiler accepts leading spaces and tabs before `//go:noinline`, `//go:embed` and `//go:build`, and they still take effect. `//go:generate` is the exception: it really must start at column 0. Writing every directive at column 0 is still the right habit, but what enforces it is `gofmt`, not the compiler.
- **No blank line.** A blank line between the directive and the declaration doesn't break the association. The `embed` docs say only blank lines and `//` line comments may sit between the directive and the variable, and a `//go:noinline` followed by a blank line and `func F()` still suppresses inlining. Keeping the directive directly above the declaration is clearer, but it isn't required.

One placement rule is strict: `//go:build` must appear **before the `package` clause**, or the file fails to compile.

## 1. `//go:generate` — Code Generation

The most commonly used directive. It doesn't affect compilation at all — it's a hook for `go generate` to run arbitrary commands. It must start at column 0.

```go
package generate

//go:generate stringer -type=Weekday

type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)
```

Running `go generate ./...` executes `stringer -type=Weekday`, which produces a `weekday_string.go` file with a `String()` method.

`stringer` is a separate program, so install it before you generate:

```sh
go install golang.org/x/tools/cmd/stringer@latest
```

This is true of every `//go:generate` tool: the directive is only a hook, and `go generate` just runs whatever is already on your `PATH` — it never installs anything for you. `go install` drops the binary in `$GOBIN` (or `$(go env GOPATH)/bin`), so if that directory isn't on your `PATH`, resolve it once per shell:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

The same pattern, used heavily for mock generation:

```go
//go:generate mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks
type Service interface {
	Fetch(id string) (string, error)
}
```

Install `mockgen` before running `go generate`, using the same `go install` as above:

```sh
go install go.uber.org/mock/mockgen@latest
```

Note: `github.com/golang/mock` was archived in June 2023 and points to its maintained fork, `go.uber.org/mock`. The `mockgen` command-line flags are the same, so only the install path changes.

## 2. `//go:build` — Build Constraints

Controls which files get compiled for which platform, architecture, or custom tag. It replaced the older `// +build` syntax (Go 1.17+).

```go
//go:build linux || darwin

package buildtags

func OSName() string {
	return "unix-like"
}
```

A Windows-only counterpart:

```go
//go:build windows

package buildtags

func OSName() string {
	return "windows"
}
```

And a fallback, so the package still builds for targets neither file covers. A constraint expression negates the others with `&&`, not `||`:

```go
//go:build !linux && !darwin && !windows

package buildtags

func OSName() string {
	return "other"
}
```

The directive must appear before the `package` clause (this is required). It should also be followed by a blank line, which is the documented convention.

**Building each version.** The `go:build` tag on its own doesn't pick a platform — it just marks which file is eligible. You select the target with `GOOS`/`GOARCH` env vars (or `-tags` for custom tags):

```sh
# Build for Linux — pulls in the linux/darwin file
GOOS=linux GOARCH=amd64 go build -o app-linux .

# Build for macOS — also pulls in the linux/darwin file
GOOS=darwin GOARCH=arm64 go build -o app-mac .

# Build for Windows — pulls in the windows file instead
GOOS=windows GOARCH=amd64 go build -o app.exe .

# Build with a custom tag (e.g. //go:build integration)
go build -tags=integration .
go test -tags=integration ./...
```

Go automatically compiles only the file whose constraint matches the target `GOOS`/`GOARCH`; the other files are simply excluded from that build, so every `OSName` implementation can share the same signature without ever conflicting.

## 3. `//go:embed` — Embedding Files

Introduced in Go 1.16, this embeds static assets straight into the binary at compile time. There's no separate build step, no `ioutil.ReadFile` at runtime, and no risk of the file going missing in production — whatever you embed ships inside the binary itself.

**Embedding a whole directory as a virtual filesystem**, then serving it over HTTP. The directive belongs in the package that owns the assets:

```go
// server.go
package embed

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

// StaticHandler serves the embedded static directory from the site root.
func StaticHandler() (http.Handler, error) {
	// embed.FS keeps the "static/" prefix, so strip it with fs.Sub to serve
	// the directory's contents from the site root.
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(sub)), nil
}
```

and the command just wires it up:

```go
// cmd/embedserve/main.go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gz-zenn/go-annotations/embed"
)

func main() {
	handler, err := embed.StaticHandler()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
```

Why `fs.Sub`? An `embed.FS` preserves the directory path, so a file at `static/app.css` lives at `static/app.css` inside the FS. If you serve the FS directly at `/`, `GET /app.css` returns 404 and the file is only reachable at `/static/app.css`. The catch is that this compiles and starts without complaint: the server binds the port and answers `200` to `/static/app.css`, so it is easy to assume it worked. Note that swapping the pattern doesn't help — `static` and `static/*` produce identical paths in the FS and differ only in which dotfiles they pick up, so the `static/` prefix is there either way.

**Embedding a single file as a string** — handy for config defaults, SQL migrations, or prompt templates:

```go
//go:embed config/default.yaml
var defaultConfig string
```

**Embedding as `[]byte`** instead of `string`, useful for binary assets like images:

```go
//go:embed assets/logo.png
var logoBytes []byte
```

**Embedding multiple patterns into one `embed.FS`** — combine directories and file types with several patterns on one directive:

```go
//go:embed templates/*.html static/css/*.css static/js/*.js
var webAssets embed.FS
```

**Reading an individual file out of an embedded FS** at runtime:

```go
//go:embed migrations/*.sql
var migrationFiles embed.FS

func loadMigration(name string) ([]byte, error) {
	return migrationFiles.ReadFile("migrations/" + name)
}
```

**Dotfiles and `_` files.** Which files get embedded depends on how you write the pattern:

| Pattern      | First-level `.`/`_` files (`dir/.env`) | Deeper ones (`dir/sub/.env`) |
| ------------ | -------------------------------------- | ---------------------------- |
| `dir`        | excluded                               | excluded                     |
| `dir/*`      | **included**                           | excluded                     |
| `all:dir`    | included                               | included                     |

In other words, naming the directory itself (`dir`) skips dotfiles at every level, while `dir/*` picks up the ones directly inside it. If you embed a directory that might contain a `.env` or similar secret file, prefer plain `dir`, and never use `dir/*` or `all:dir` there without checking what's inside.

All three patterns in one file, over the same directory:

```go
//go:embed static
var plainDir embed.FS

//go:embed static/*
var firstLevelWildcard embed.FS

//go:embed all:static
var everything embed.FS
```

With `static/sub/.env` on disk, that yields exactly the table above: it reaches the binary if and only if the pattern is `all:static`. Use `all:` when you really do want everything.

The `all:` prefix requires Go 1.18 or later; on Go 1.16 and 1.17 it fails to build with "no matching files found".

A few constraints worth knowing:

- The variable must be package-level (not inside a function).
- Only `string`, `[]byte`, and `embed.FS` are valid target types.
- The `embed` package must be imported, even if only for its side effect (`import _ "embed"`) when you're embedding into a `string` or `[]byte` and don't otherwise use the package.

## 4. `//go:noinline` — Prevent Inlining

Used mostly in benchmarks, testing, or when debugging with tools that need real stack frames.

```go
//go:noinline
func Add(a, b int) int {
	return a + b
}
```

You can confirm it took effect with `go build -gcflags=-m=2`, which reports inlining decisions along with the reason for each refusal:

```sh
$ go build -gcflags=-m=2 ./noinline
noinline/noinline.go:20:6: cannot inline Add: marked go:noinline
noinline/noinline.go:26:6: can inline AddInlinable with cost 4 as: func(int, int) int { return a + b }
```

Use `-m=2` rather than `-m`. Plain `-m` prints only what *can* be inlined, so it lists the control function and says nothing about the one you marked.

## 5. `//go:noescape` — Suppress Escape Analysis

Tells the compiler a function (typically implemented in assembly) doesn't let its pointer arguments escape to the heap, either by being stored somewhere that outlives the call or by flowing into the values the function returns. It's real, but dangerous — lying to the compiler here can cause memory corruption.

The declaration has no body, so the function must be implemented in assembly in the same package. A declaration alone won't compile, and an empty `.s` file isn't enough either: it silences the compile-time error, but the linker fails as soon as anything calls the function. A minimal example:

```go
// noescape_amd64.go
//go:build amd64

package noescape

//go:noescape
func load(p *int64) int64
```

```asm
// noescape_amd64.s
//go:build amd64

#include "textflag.h"

// func load(p *int64) int64
TEXT ·load(SB), NOSPLIT, $0-16
	MOVQ p+0(FP), AX
	MOVQ (AX), AX
	MOVQ AX, ret+8(FP)
	RET
```

Both files carry `//go:build`, because every architecture needs its own pair — `noescape_arm64.go`/`.s` for arm64, and a plain Go body for everything else, where the compiler can see the implementation and the directive would be redundant. `go vet`'s `asmdecl` analyzer checks the frame offsets in the assembly against the Go declaration, which is the only way to check an architecture you are not building for.

This pattern is common in the standard library for assembly-backed functions. Note the contrast with `//go:linkname` (next section), which suppresses the same missing-body compile error on its own.

## 6. `//go:linkname` — Cross-Package Symbol Linking

Lets one package reference an unexported symbol in another, bypassing normal visibility rules. Requires importing `unsafe`. This is how the standard library and runtime share internals:

```go
package linkname

import _ "unsafe"

//go:linkname now time.now
func now() (sec int64, nsec int32, mono int64)
```

The target is `time.now`, the unexported function behind `time.Now`. Reaching into a package of your own works the same way, with one extra wrinkle: outside the standard library the symbol in the object file is the **fully qualified** name, not the short package name.

```go
package linkname

import (
	// The blank import is what puts the values package into the binary. A
	// symbol that is not linked in cannot be resolved, and that failure shows
	// up at link time, not at compile time.
	_ "github.com/gz-zenn/go-annotations/linkname/values"
	_ "unsafe"
)

//go:linkname internalAdd github.com/gz-zenn/go-annotations/linkname/values.add
func internalAdd(a, b int) int
```

Writing `values.add` instead does not compile-fail; it fails at link time with `relocation target values.add not defined`. Drop the blank import and you get the same class of error for a different reason: nothing pulls `values` into the binary.

Because `//go:linkname` marks the function as linked to an external symbol, this builds without any `.s` file, unlike `//go:noescape`.

This is considered a low-level escape hatch, and the Go team has warned that it may break between versions. Since Go 1.23, the linker rejects a linkname reference into a standard-library internal symbol unless the definition side is explicitly marked to allow it; `-checklinkname=0` turns the check off. This is documented behavior in the Go 1.23 release notes.

## 7. `//go:nosplit` — Disable Stack-Growth Checks

Used in runtime-critical code where a stack check/split could cause reentrancy problems (e.g., signal handlers).

```go
// consume has a real frame, which is what makes the difference visible in
// the generated code. It is deliberately not inlinable.
//
//go:noinline
func consume(n int) int {
	var pad [32]byte
	pad[0] = byte(n)
	return int(pad[0])
}

//go:nosplit
func Critical(n int) int {
	return consume(n)
}

// The same function without the directive: the compiler emits the stack check
// and the call to runtime.morestack.
func Expandable(n int) int {
	return consume(n)
}
```

Every ordinary function starts with a check for enough stack, and calls `runtime.morestack` when there isn't. Drop the directive and you can see it in the assembly:

```sh
$ go build -gcflags=-S ./nosplit | grep -E 'TEXT.*nosplit\.(Critical|Expandable)\(SB\)|morestack'
```

```text
TEXT ...nosplit.Critical(SB), NOSPLIT|ABIInternal, $32-8
TEXT ...nosplit.Expandable(SB), ABIInternal, $32-8
CALL runtime.morestack_noctxt(SB)      <- from Expandable
```

`Critical`'s symbol is flagged `NOSPLIT` and never mentions `morestack`; `Expandable` keeps the call. The directive is per-function and is not inherited: `CriticalNested`, a plain function that only calls `nosplit` functions, keeps its own check and its own `morestack` call.

**Warning:** this one is easy to try because it compiles cleanly, but the Go documentation is blunt about it. Outside low-level runtime code it is not safe. A `nosplit` function cannot grow the stack, so a frame larger than the budget the runtime reserves overwrites the end of the stack, leading to memory corruption and arbitrary program failure. Nothing warns you about it: in the example above `Critical` has a 32-byte frame (`$32-8` above) and is fine, but the same annotation on a function with a kilobyte frame is a memory corruption bug, not a slow path.

## Quick Reference

| Directive       | Purpose                    | Typical use                |
| --------------- | -------------------------- | -------------------------- |
| `//go:generate` | Trigger external codegen   | mocks, stringers, protobuf |
| `//go:build`    | Conditional compilation    | OS/arch-specific files     |
| `//go:embed`    | Embed files into binary    | static assets, configs     |
| `//go:noinline` | Block inlining             | benchmarking, debugging    |
| `//go:noescape` | Skip escape analysis       | assembly-backed functions  |
| `//go:linkname` | Link to unexported symbols | runtime/stdlib internals   |
| `//go:nosplit`  | Skip stack-split checks    | low-level runtime code     |

<!--
TODO (author): add a short section here, "What we actually use in our own Go code",
covering which of these directives GVA TECH uses and what problems you ran into
(e.g. embed + fs.Sub for serving assets, go:generate for mocks, build tags for tests).
-->

## Closing Thoughts

Most Go code never needs anything beyond `//go:generate`, `//go:build`, and `//go:embed` — these three cover code generation, cross-platform builds, and static assets, which is the vast majority of everyday use. The lower-level ones (`noescape`, `linkname`, `nosplit`) are mainly the domain of standard library and runtime authors; reach for them only when you're writing assembly-adjacent or performance-critical code and you understand exactly what guarantees you're giving up.

If you only remember a few, remember those three. `//go:build` shows up in almost any codebase that ships binaries for more than one platform, `//go:embed` in anything that ships assets or config in a single binary, and `//go:generate` in most repos with mocks, stringers, or protobuf. `//go:noinline` is a niche debugging and benchmarking tool, and the remaining three barely appear outside the standard library — you can safely skip them entirely until you are writing assembly-adjacent code.

