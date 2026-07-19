package store

import "testing"

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
