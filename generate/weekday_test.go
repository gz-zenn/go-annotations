package generate_test

import (
	"fmt"
	"testing"

	"github.com/gz-zenn/go-annotations/generate"
)

func TestWeekdayString(t *testing.T) {
	days := []struct {
		day  generate.Weekday
		want string
	}{
		{generate.Sunday, "Sunday"},
		{generate.Monday, "Monday"},
		{generate.Tuesday, "Tuesday"},
		{generate.Wednesday, "Wednesday"},
		{generate.Thursday, "Thursday"},
		{generate.Friday, "Friday"},
		{generate.Saturday, "Saturday"},
	}

	for _, tc := range days {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.day.String(); got != tc.want {
				t.Errorf("Weekday(%d).String() = %q, want %q", int(tc.day), got, tc.want)
			}
		})
	}
}

func TestWeekdayStringOutOfRange(t *testing.T) {
	// Values outside the constant block have no name; stringer falls back to
	// the numeric form rather than panicking.
	for _, day := range []generate.Weekday{-1, 7, 99} {
		want := fmt.Sprintf("Weekday(%d)", int(day))
		if got := day.String(); got != want {
			t.Errorf("Weekday(%d).String() = %q, want %q", int(day), got, want)
		}
	}
}

func TestWeekdayStringerIsUsedByFmt(t *testing.T) {
	// String makes the type usable with %v and %s without a wrapper method.
	if got := fmt.Sprintf("%v/%s", generate.Friday, generate.Sunday); got != "Friday/Sunday" {
		t.Errorf("formatting = %q, want %q", got, "Friday/Sunday")
	}
}

func TestWeekdayConstantsAreSequential(t *testing.T) {
	// The generated file indexes the constants positionally, so the iota
	// block must stay contiguous for the _() check in weekday_string.go to
	// keep compiling.
	for i, day := range []generate.Weekday{
		generate.Sunday, generate.Monday, generate.Tuesday, generate.Wednesday,
		generate.Thursday, generate.Friday, generate.Saturday,
	} {
		if int(day) != i {
			t.Errorf("constant %d has value %d, want %d", i, int(day), i)
		}
	}
}
