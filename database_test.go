package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDatabasePath(t *testing.T) {
	workingDir := t.TempDir()
	legacyDir := t.TempDir()
	t.Chdir(workingDir)
	t.Setenv("TMPDIR", legacyDir)
	t.Setenv("DATABASE_PATH", "")

	if got := resolveDatabasePath(); got != defaultDatabaseFilename {
		t.Fatalf("default database path = %q, want %q", got, defaultDatabaseFilename)
	}

	legacyPath := filepath.Join(legacyDir, defaultDatabaseFilename)
	if err := os.WriteFile(legacyPath, nil, 0o600); err != nil {
		t.Fatalf("create legacy database: %v", err)
	}
	if got := resolveDatabasePath(); got != legacyPath {
		t.Fatalf("legacy database path = %q, want %q", got, legacyPath)
	}

	if err := os.WriteFile(defaultDatabaseFilename, nil, 0o600); err != nil {
		t.Fatalf("create local database: %v", err)
	}
	if got := resolveDatabasePath(); got != defaultDatabaseFilename {
		t.Fatalf("local database path = %q, want %q", got, defaultDatabaseFilename)
	}

	t.Setenv("DATABASE_PATH", "/configured/wishlist.db")
	if got := resolveDatabasePath(); got != "/configured/wishlist.db" {
		t.Fatalf("configured database path = %q, want configured path", got)
	}
}

func TestInitDBPreservesDataAndEnforcesConnectionSettings(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "wishlist.db")
	t.Setenv("DATABASE_PATH", databasePath)

	db, err := initDB()
	if err != nil {
		t.Fatalf("initialize database: %v", err)
	}

	if got := db.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("max open connections = %d, want 1", got)
	}

	var foreignKeys int
	if queryErr := db.QueryRowContext(t.Context(), "PRAGMA foreign_keys").Scan(&foreignKeys); queryErr != nil {
		t.Fatalf("read foreign key setting: %v", queryErr)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign key setting = %d, want 1", foreignKeys)
	}

	var busyTimeout int
	if queryErr := db.QueryRowContext(t.Context(), "PRAGMA busy_timeout").Scan(&busyTimeout); queryErr != nil {
		t.Fatalf("read busy timeout: %v", queryErr)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy timeout = %d, want 5000", busyTimeout)
	}

	if _, insertErr := db.ExecContext(t.Context(), "INSERT INTO family_members (name) VALUES (?)", "Alice"); insertErr != nil {
		t.Fatalf("insert family member: %v", insertErr)
	}
	if _, insertErr := db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name) VALUES (?, ?)", 1, "Book"); insertErr != nil {
		t.Fatalf("insert item: %v", insertErr)
	}
	if closeErr := db.Close(); closeErr != nil {
		t.Fatalf("close database: %v", closeErr)
	}

	// A stale RESET_DB setting must never delete an existing database.
	t.Setenv("RESET_DB", "1")
	reopened, err := initDB()
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := reopened.Close(); closeErr != nil {
			t.Errorf("close reopened database: %v", closeErr)
		}
	})

	var itemCount int
	if queryErr := reopened.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM items").Scan(&itemCount); queryErr != nil {
		t.Fatalf("count preserved items: %v", queryErr)
	}
	if itemCount != 1 {
		t.Fatalf("item count after reopen = %d, want 1", itemCount)
	}

	if _, deleteErr := reopened.ExecContext(t.Context(), "DELETE FROM family_members WHERE id = 1"); deleteErr != nil {
		t.Fatalf("delete family member: %v", deleteErr)
	}
	if queryErr := reopened.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM items").Scan(&itemCount); queryErr != nil {
		t.Fatalf("count items after cascade: %v", queryErr)
	}
	if itemCount != 0 {
		t.Fatalf("item count after cascade = %d, want 0", itemCount)
	}
}
