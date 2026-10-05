#!/usr/bin/env bash
#
# Build and test everything in the repository.
#
#   The rules of the game: this repo demonstrates //go: directives, so some
#   checks only bite under the right GOOS/GOARCH or with the right build tag.
#   The script runs the default build, the default tests, the race detector,
#   the integration-tagged tests, and a cross-compilation matrix that
#   exercises every platform-specific file and assembly implementation.
#
# Override the target matrix with, e.g.:  TARGETS="linux/amd64 windows/amd64" ./build.sh
set -euo pipefail

cd "$(dirname "$0")"

# The toolchain the nested gotool tests peck at is the one in PATH.
go version

# Platform matrix below covers every build-constrained file:
#   platform_posix.go  (linux || darwin)
#   platform_win.go    (windows)
#   platform_other.go  (!linux && !darwin && !windows)
#   noescape_amd64.s / noescape_arm64.s / noescape_other.go
: "${TARGETS:=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/386 windows/amd64 freebsd/amd64 js/wasm}"

echo
echo "==> gofmt (anything listed here is unformatted; aborting)"
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
    printf '%s\n' "$unformatted"
    echo "FAIL: gofmt"
    exit 1
fi
echo "    gofmt: OK"

echo
echo "==> go build ./..."
go build ./...
echo "    build: OK"

echo
echo "==> go vet ./..."
go vet ./...
echo "    vet: OK"

echo
echo "==> go generate -n ./... (directives resolve without running generators)"
go generate -n ./... >/dev/null
echo "    generate: OK"

echo
echo "==> go test -count=1 ./..."
go test -count=1 ./...

echo
echo "==> go test -race -count=1 ./..."
go test -race -count=1 ./...

echo
echo "==> go test -count=1 -tags=integration ./..."
go test -count=1 -tags=integration ./...

echo
echo "==> cross-compilation matrix"
for target in $TARGETS; do
    os=${target%/*}
    arch=${target#*/}
    GOOS=$os GOARCH=$arch go build ./...
    echo "    build: OK $target"
done

echo
echo "ALL CHECKS PASSED"