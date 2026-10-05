//go:build integration

package buildtags

import (
	"testing"
)

// This file only exists with -tags=integration, so this test only runs when
// the tag is set: go test -tags=integration ./...
func TestIntegrationModeIsAvailable(t *testing.T) {
	if IntegrationMode != "integration" {
		t.Errorf("IntegrationMode = %q, want %q", IntegrationMode, "integration")
	}
}

func TestIntegrationDSNIsAvailable(t *testing.T) {
	// Nothing connects to it: the point is that the symbol is compiled only
	// when the tag is on, so the code is still type-checked in the tagged
	// build without dragging a database into the default test run.
	if got := IntegrationDSN(); got == "" {
		t.Error("IntegrationDSN() returned an empty DSN")
	}
}
