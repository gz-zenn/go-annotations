package linkname_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gz-zenn/go-annotations/linkname"
	"github.com/gz-zenn/go-annotations/linkname/values"
)

func TestAddThroughLinkname(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{0, 0, 0},
		{2, 3, 5},
		{-4, 4, 0},
		{1_000_000, 2_000_000, 3_000_000},
	}

	for _, tc := range tests {
		got := linkname.Add(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("Add(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestLinknameAndTheExportedFunctionAgree(t *testing.T) {
	// The point of the trick: the linknamed call and the exported one reach
	// the same code, so there is nothing to synchronise.
	for _, pair := range [][2]int{{0, 1}, {7, -7}, {123, 456}} {
		if got, want := linkname.Add(pair[0], pair[1]), values.Add(pair[0], pair[1]); got != want {
			t.Errorf("Add(%d, %d) = %d through linkname, %d through the export", pair[0], pair[1], got, want)
		}
	}
}

func TestLinknameCallsTheSameCode(t *testing.T) {
	// values.add is the only thing that bumps values.Calls, so the counter
	// moving proves the linkname resolved to the real symbol.
	before := values.Calls
	if got := linkname.Add(1, 2); got != 3 {
		t.Fatalf("Add = %d, want 3", got)
	}
	if values.Calls != before+1 {
		t.Errorf("values.Calls = %d, want %d: the linknamed function did not run", values.Calls, before+1)
	}
}

func TestNowReadsTheRuntimeClock(t *testing.T) {
	sec, nsec, mono := linkname.Now()

	if nsec < 0 || nsec >= 1_000_000_000 {
		t.Errorf("nsec = %d, want it in [0, 1e9)", nsec)
	}
	if mono <= 0 {
		t.Errorf("mono = %d, want a positive monotonic reading", mono)
	}
	if sec <= 0 {
		t.Errorf("sec = %d, want a positive wall clock reading", sec)
	}
}

func TestNowAgreesWithTheTimePackage(t *testing.T) {
	before := time.Now()
	sec, _, _ := linkname.Now()
	after := time.Now()

	if sec < before.Unix()-1 || sec > after.Unix()+1 {
		t.Errorf("sec = %d, want it between %d and %d", sec, before.Unix(), after.Unix())
	}
}

func TestNowIsUsableToBuildATime(t *testing.T) {
	sec, nsec, _ := linkname.Now()
	if got := time.Unix(sec, int64(nsec)); got.Unix() != sec {
		t.Errorf("time.Unix(%d, %d).Unix() = %d, want %d", sec, nsec, got.Unix(), sec)
	}
}

func ExampleNow() {
	sec, nsec, mono := linkname.Now()
	fmt.Println(sec > 0, nsec >= 0, mono > 0)
	// Output: true true true
}
