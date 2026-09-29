package embed_test

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/gz-zenn/go-annotations/embed"
)

func TestLoadMigrationReadsAnEmbeddedFile(t *testing.T) {
	got, err := embed.LoadMigration("0001_create_users.sql")
	if err != nil {
		t.Fatalf("LoadMigration: %v", err)
	}
	if !strings.Contains(string(got), "CREATE TABLE users") {
		t.Errorf("migration = %q, want the CREATE TABLE statement", got)
	}
}

func TestLoadMigrationUnknownName(t *testing.T) {
	// The set of migrations is fixed at build time: a name that was not
	// embedded cannot be added at runtime.
	_, err := embed.LoadMigration("0003_not_there.sql")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("error = %v, want fs.ErrNotExist", err)
	}
}

func TestReadMigrationStringWrapsTheError(t *testing.T) {
	_, err := embed.ReadMigrationString("../../etc/passwd")
	if err == nil {
		t.Fatal("expected an error for a traversal attempt")
	}
	if !strings.Contains(err.Error(), "../../etc/passwd") {
		t.Errorf("error = %v, want it to mention the name", err)
	}
}

func TestMigrationNamesAreSorted(t *testing.T) {
	names, err := embed.MigrationNames()
	if err != nil {
		t.Fatalf("MigrationNames: %v", err)
	}

	want := []string{"0001_create_users.sql", "0002_seed_users.sql"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("names = %v, want %v", names, want)
	}
}

func TestEveryMigrationIsReadable(t *testing.T) {
	names, err := embed.MigrationNames()
	if err != nil {
		t.Fatalf("MigrationNames: %v", err)
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			body, err := embed.ReadMigrationString(name)
			if err != nil {
				t.Fatalf("ReadMigrationString(%s): %v", name, err)
			}
			if strings.TrimSpace(body) == "" {
				t.Errorf("migration %s is empty", name)
			}
		})
	}
}
