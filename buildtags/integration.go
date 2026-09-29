//go:build integration

package buildtags

// IntegrationMode is only compiled when the integration tag is set, which is
// how projects keep slow or environment-dependent tests out of the default
// build while still type-checking them in CI:
//
//	go test -tags=integration ./...
const IntegrationMode = "integration"

// IntegrationDSN is a helper used by the integration-only test file.
func IntegrationDSN() string {
	return "postgres://localhost:5432/testdb?sslmode=disable"
}
