package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const defaultDatabaseFilename = "wishlist.db"

func resolveDatabasePath() string {
	if configuredPath := os.Getenv("DATABASE_PATH"); configuredPath != "" {
		return configuredPath
	}

	// Keep new databases in the working directory so they survive OS temp cleanup.
	if _, err := os.Stat(defaultDatabaseFilename); err == nil || !os.IsNotExist(err) {
		return defaultDatabaseFilename
	}

	// Preserve installations that previously used the temporary-directory default.
	legacyPath := filepath.Join(os.TempDir(), defaultDatabaseFilename)
	if _, err := os.Stat(legacyPath); err == nil {
		log.Printf("Using legacy database at %s; set DATABASE_PATH or move it to %s", legacyPath, defaultDatabaseFilename)
		return legacyPath
	}

	return defaultDatabaseFilename
}

func sqliteDSN(filename string) string {
	separator := "?"
	if strings.Contains(filename, "?") {
		separator = "&"
	}
	return filename + separator + "_foreign_keys=on&_busy_timeout=5000"
}

func initDB() (*sql.DB, error) {
	dbFilename := resolveDatabasePath()

	db, err := sql.Open("sqlite3", sqliteDSN(dbFilename))
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", dbFilename, err)
	}
	db.SetMaxOpenConns(1)

	initialized := false
	defer func() {
		if !initialized {
			_ = db.Close()
		}
	}()

	ctx := context.Background()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database %s: %w", dbFilename, err)
	}

	if _, err := db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS family_members (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL UNIQUE
        );
    `); err != nil {
		return nil, fmt.Errorf("failed to create family_members table: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS items (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            family_member_id INTEGER NOT NULL,
            name TEXT NOT NULL,
            link TEXT,
            price REAL,
            reserved BOOLEAN DEFAULT 0,
            FOREIGN KEY (family_member_id) REFERENCES family_members(id) ON DELETE CASCADE
        );
    `); err != nil {
		return nil, fmt.Errorf("failed to create items table: %w", err)
	}

	initialized = true
	return db, nil
}
