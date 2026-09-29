package generate_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/gz-zenn/go-annotations/generate"
	"github.com/gz-zenn/go-annotations/generate/mocks"
)

// The whole point of the generated mock is that it satisfies the interface
// it was generated from; this fails at compile time if the source drifts.
var _ generate.Service = (*mocks.MockService)(nil)

func TestCacheLabelWithMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockService(ctrl)

	svc.EXPECT().
		Fetch("42").
		Return("the answer", nil).
		Times(1)

	cache := generate.NewCache(svc)
	got, err := cache.Label("42")
	if err != nil {
		t.Fatalf("Label: %v", err)
	}
	if got != "the answer" {
		t.Errorf("Label = %q, want %q", got, "the answer")
	}
}

func TestCacheLabelPropagatesServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockService(ctrl)

	boom := errors.New("boom")
	svc.EXPECT().Fetch(gomock.Any()).Return("", boom)

	cache := generate.NewCache(svc)
	got, err := cache.Label("7")
	if !errors.Is(err, boom) {
		t.Fatalf("Label error = %v, want wrapped %v", err, boom)
	}
	if got != "" {
		t.Errorf("Label = %q, want empty string on error", got)
	}
}

func TestCacheLabelEmptyValueIsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockService(ctrl)

	svc.EXPECT().Fetch("ghost").Return("", nil)

	cache := generate.NewCache(svc)
	if _, err := cache.Label("ghost"); !errors.Is(err, generate.ErrNotFound) {
		t.Fatalf("Label error = %v, want %v", err, generate.ErrNotFound)
	}
}

// reporter captures gomock failures instead of failing the test, so that the
// mock's own failure paths can be asserted on.
type reporter struct {
	failures []string
}

func (r *reporter) Helper() {}

func (r *reporter) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *reporter) Fatalf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
	panic("gomock reported a failure")
}

func TestMockFailsOnUnexpectedCall(t *testing.T) {
	rep := &reporter{}
	ctrl := gomock.NewController(rep)
	svc := mocks.NewMockService(ctrl)

	// No EXPECT was registered, so the call must be reported.
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("unexpected call did not fail the mock")
			}
		}()
		_, _ = svc.Fetch("unexpected")
	}()

	if len(rep.failures) == 0 || !strings.Contains(rep.failures[0], "Unexpected call") {
		t.Errorf("failures = %v, want one mentioning \"Unexpected call\"", rep.failures)
	}
}

func TestMockFailsOnMissingCall(t *testing.T) {
	rep := &reporter{}
	ctrl := gomock.NewController(rep)
	svc := mocks.NewMockService(ctrl)

	svc.EXPECT().Fetch("never-called").Return("", nil)

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("Finish should fail when an expectation was not met")
			}
		}()
		ctrl.Finish()
	}()

	if len(rep.failures) == 0 || !strings.Contains(rep.failures[0], "missing call") {
		t.Errorf("failures = %v, want one mentioning \"missing call\"", rep.failures)
	}
}
