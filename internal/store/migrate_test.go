package store

import (
	"strings"
	"testing"
)

func TestMigrationVersion(t *testing.T) {
	t.Parallel()
	version, err := migrationVersion("001_initial.sql")
	if err != nil || version != 1 {
		t.Fatalf("unexpected version=%d err=%v", version, err)
	}
	for _, name := range []string{"initial.sql", "000_invalid.sql", "x_bad.sql"} {
		if _, err := migrationVersion(name); err == nil {
			t.Errorf("%q should be rejected", name)
		}
	}
}

func TestEmbeddedMigrationLedgerIsContiguous(t *testing.T) {
	t.Parallel()
	version, err := expectedSchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if version != 15 {
		t.Fatalf("embedded migration version = %d, want 15", version)
	}
}

func TestSearchIndexingMigrationDefaultsOffAndRequiresPublicVisibility(t *testing.T) {
	t.Parallel()
	sql, err := migrationFiles.ReadFile("migrations/005_site_search_indexing.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := string(sql)
	for _, expected := range []string{
		"search_indexing boolean NOT NULL DEFAULT false",
		"NOT search_indexing OR visibility = 'public'",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("migration missing %q: %s", expected, body)
		}
	}
}
