package main

import (
	"path"
	"strconv"
	"strings"
	"testing"
)

// Two branches numbering their migration the same collides only at deploy time,
// where goose panics: git sees two different filenames and merges both cleanly.
// TestMigrations catches it too, but it is behind the integration build tag and
// needs Docker, so it does not run in the normal suite. This one does.
func TestMigrationVersionsAreUnique(t *testing.T) {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatalf("reading the embedded migrations: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no migrations embedded — the guard would pass vacuously")
	}

	seen := make(map[int]string, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || path.Ext(name) != ".sql" {
			continue
		}
		prefix, _, ok := strings.Cut(name, "_")
		if !ok {
			t.Errorf("%s does not start with a version prefix", name)
			continue
		}
		version, err := strconv.Atoi(prefix)
		if err != nil {
			t.Errorf("%s has a non-numeric version prefix %q", name, prefix)
			continue
		}
		if other, dup := seen[version]; dup {
			t.Errorf("duplicate migration version %d: %s and %s", version, other, name)
			continue
		}
		seen[version] = name
	}
}
