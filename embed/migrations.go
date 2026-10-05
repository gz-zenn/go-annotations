package embed

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// loadMigration reads one file out of the embedded FS at runtime. The name
// is the part after migrations/.
func loadMigration(name string) ([]byte, error) {
	return migrationFiles.ReadFile("migrations/" + name)
}

// LoadMigration is the exported wrapper around loadMigration.
func LoadMigration(name string) ([]byte, error) {
	return loadMigration(name)
}

// MigrationNames lists the embedded migrations in lexical order, which is
// also the order they must be applied in.
func MigrationNames() ([]string, error) {
	entries, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry[len("migrations/"):])
	}
	sort.Strings(names)
	return names, nil
}

// ReadMigrationString returns the migration as a string, failing loudly on a
// name that was not embedded at all.
func ReadMigrationString(name string) (string, error) {
	data, err := loadMigration(name)
	if err != nil {
		return "", fmt.Errorf("migration %q: %w", name, err)
	}
	return string(data), nil
}
