package nosplit_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/internal/gotool"
	"github.com/gz-zenn/go-annotations/nosplit"
)

// pkgPath is the fully qualified name the compiler uses in the -S listing.
const pkgPath = "github.com/gz-zenn/go-annotations/nosplit."

// symbolRe matches the header line the compiler prints for each function in
// the -S listing, for example:
//
//	github.com/gz-zenn/go-annotations/nosplit.Critical STEXT nosplit size=32 ...
var symbolRe = regexp.MustCompile(`^(\S+)\s+STEXT\b(.*)$`)

// asmOf returns the listing block of one function: the flags from its header
// and the instructions that follow.
func asmOf(t *testing.T, symbol string) (flags, body string) {
	t.Helper()
	gotool.SkipIfUnavailable(t)

	res := gotool.Run(t, "..", nil, "build", "-gcflags=-S", "./nosplit/")
	res.MustSucceed(t)

	symbol = pkgPath + strings.TrimPrefix(symbol, "nosplit.")

	var found bool
	var lines []string
	for _, line := range strings.Split(res.Output, "\n") {
		// A line at column 0 starts a new symbol, which also ends the
		// listing of the previous one.
		if !strings.HasPrefix(line, "\t") {
			if found {
				break
			}
			if m := symbolRe.FindStringSubmatch(line); m != nil && m[1] == symbol {
				flags = m[2]
				found = true
			}
			continue
		}
		if found {
			lines = append(lines, line)
		}
	}
	if !found {
		t.Fatalf("symbol %s is not in the -S listing", symbol)
	}
	return flags, strings.Join(lines, "\n")
}

func TestCriticalSkipsTheStackCheck(t *testing.T) {
	flags, body := asmOf(t, "nosplit.Critical")

	if !strings.Contains(flags, "nosplit") {
		t.Errorf("symbol flags = %q, want them to contain nosplit", flags)
	}
	if !strings.Contains(body, "NOSPLIT") {
		t.Errorf("TEXT line = %q, want it to contain NOSPLIT", body)
	}
	if strings.Contains(body, "morestack") {
		t.Errorf("the function calls morestack, so the stack check was not removed:\n%s", body)
	}
}

func TestExpandableKeepsTheStackCheck(t *testing.T) {
	// The control group: the same function without the directive still calls
	// runtime.morestack, which is what can grow the stack.
	flags, body := asmOf(t, "nosplit.Expandable")

	if strings.Contains(flags, "nosplit") {
		t.Errorf("symbol flags = %q, want them not to contain nosplit", flags)
	}
	if !strings.Contains(body, "morestack") {
		t.Errorf("the function does not call morestack:\n%s", body)
	}
}

func TestOnlyTheMarkedFunctionLosesTheCheck(t *testing.T) {
	// CriticalNested is not marked, so it keeps its own check even though it
	// only calls nosplit functions. //go:nosplit is not inherited.
	flags, body := asmOf(t, "nosplit.CriticalNested")

	if strings.Contains(flags, "nosplit") {
		t.Errorf("symbol flags = %q, want them not to contain nosplit", flags)
	}
	if !strings.Contains(body, "morestack") {
		t.Errorf("CriticalNested lost its stack check:\n%s", body)
	}
}

func TestCriticalFrameIsSmall(t *testing.T) {
	// This is what makes the directive survivable: a nosplit function cannot
	// grow the stack, so its frame has to fit in the budget the runtime
	// reserves. A frame in the kilobytes is a memory corruption bug, not a
	// slow path, and nothing warns about it.
	_, body := asmOf(t, "nosplit.Critical")
	frame := frameSize(t, body)

	const budget = 800
	if frame > budget {
		t.Errorf("frame = %d bytes, want at most %d for a nosplit function", frame, budget)
	}
	t.Logf("Critical frame = %d bytes", frame)
}

func TestCriticalComputesTheSameResult(t *testing.T) {
	// The directive changes the prologue, not the semantics.
	for _, n := range []int{0, 1, 2, 10, 255, 256, -3} {
		got, want := nosplit.Critical(n), nosplit.Expandable(n)
		if got != want {
			t.Errorf("Critical(%d) = %d, Expandable(%d) = %d", n, got, n, want)
		}
		if got != int(byte(n)) {
			t.Errorf("Critical(%d) = %d, want %d", n, got, int(byte(n)))
		}
	}
}

func TestCriticalNestedComputesTheSameResult(t *testing.T) {
	// Two nosplit frames in a row, and the result is still the same.
	for _, n := range []int{0, 7, 200, 1000} {
		if got, want := nosplit.CriticalNested(n), nosplit.Critical(n); got != want {
			t.Errorf("CriticalNested(%d) = %d, Critical(%d) = %d", n, got, n, want)
		}
	}
}

func TestCriticalIsSafeWhenTheStackIsDeep(t *testing.T) {
	// A nosplit function cannot grow the stack, so it must still work when
	// the goroutine is close to its limit. This walks the stack deep enough
	// to force growth on the way down, then calls the nosplit function at
	// the bottom of it.
	depth := 2000
	var descend func(int) int
	descend = func(n int) int {
		if n == 0 {
			return nosplit.Critical(1)
		}
		// A local array keeps the frame real, so the recursion consumes
		// actual stack.
		var pad [256]byte
		pad[0] = byte(n)
		return int(pad[0]) & descend(n-1)
	}

	if got := descend(depth); got < 0 {
		t.Errorf("descend returned %d", got)
	}
}

func TestCriticalIsSmallerThanExpandable(t *testing.T) {
	// The shape of the difference: the stack check costs a compare, a branch
	// and the morestack call.
	critical, _ := asmOf(t, "nosplit.Critical")
	expandable, _ := asmOf(t, "nosplit.Expandable")

	criticalSize := codeSize(t, critical)
	expandableSize := codeSize(t, expandable)

	if criticalSize >= expandableSize {
		t.Errorf("Critical is %d bytes and Expandable is %d, want the nosplit one to be smaller",
			criticalSize, expandableSize)
	}
	t.Logf("Critical = %d bytes, Expandable = %d bytes", criticalSize, expandableSize)
}

func codeSize(t *testing.T, flags string) int {
	t.Helper()

	m := regexp.MustCompile(`\bsize=(\d+)`).FindStringSubmatch(flags)
	if m == nil {
		t.Fatalf("no code size in the symbol flags: %q", flags)
	}
	size, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("parsing code size %q: %v", m[1], err)
	}
	return size
}

// frameSize reads the "$frame-args" numbers off the TEXT line of a listing.
//
//	0x0000 00000 (nosplit.go:31)  TEXT  nosplit.Critical(SB), NOSPLIT|ABIInternal, $32-8
func frameSize(t *testing.T, body string) int {
	t.Helper()

	for _, line := range strings.Split(body, "\n") {
		if !strings.Contains(line, "\tTEXT\t") {
			continue
		}
		m := regexp.MustCompile(`\$(\d+)-\d+`).FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("no frame size on the TEXT line: %q", line)
		}
		size, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("parsing frame size %q: %v", m[1], err)
		}
		return size
	}
	t.Fatalf("no TEXT line in the listing:\n%s", body)
	return 0
}
