package db

import "testing"

func TestLoadMigrations(t *testing.T) {
	migs, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations error: %v", err)
	}
	if len(migs) == 0 {
		t.Fatalf("expected at least one migration")
	}
	for i, m := range migs {
		if m.Version == 0 {
			t.Fatalf("migration at %d has zero version", i)
		}
		if m.Content == "" {
			t.Fatalf("migration %d has empty content", m.Version)
		}
		if i > 0 && migs[i-1].Version >= m.Version {
			t.Fatalf("migrations not sorted ascending at %d", i)
		}
	}
}
