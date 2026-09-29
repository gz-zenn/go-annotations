package embed

import (
	"embed"
	"io/fs"
	"sort"
)

// Three FS variables over the same directory, one per pattern style from the
// article. The differences are only about files whose name starts with "." or
// "_":

//go:embed static
var plainDir embed.FS

//go:embed static/*
var firstLevelWildcard embed.FS

//go:embed all:static
var everything embed.FS

// PlainDir embeds the directory itself: dotfiles are skipped at every level.
func PlainDir() embed.FS { return plainDir }

// FirstLevelWildcard embeds the direct children: the dotfiles right inside
// the directory are included, the ones in subdirectories are not.
func FirstLevelWildcard() embed.FS { return firstLevelWildcard }

// Everything embeds the whole tree including every dotfile, at every level.
// The all: prefix needs Go 1.18 or later.
func Everything() embed.FS { return everything }

// DotfileReport maps the three patterns to the files each of them picked up,
// so a test can assert on the whole table.
func DotfileReport() (map[string][]string, error) {
	report := make(map[string][]string, 3)
	for name, filesys := range map[string]embed.FS{
		"static":     plainDir,
		"static/*":   firstLevelWildcard,
		"all:static": everything,
	} {
		var found []string
		err := fs.WalkDir(filesys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				found = append(found, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		sort.Strings(found)
		report[name] = found
	}
	return report, nil
}
