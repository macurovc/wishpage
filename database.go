package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func initDB() (*sql.DB, error) {
	dbFilename := os.Getenv("DATABASE_PATH")
	if dbFilename == "" {
		dbFilename = filepath.Join(os.TempDir(), "wishlist.db")
	}

	db, err := sql.Open("sqlite3", dbFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", dbFilename, err)
	}

	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Only reset schema if explicitly requested or using in-memory DB
	reset := os.Getenv("RESET_DB") == "1" || dbFilename == ":memory:"
	if reset {
		if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS items;`); err != nil {
			return nil, fmt.Errorf("failed to drop items table: %w", err)
		}
		if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS family_members;`); err != nil {
			return nil, fmt.Errorf("failed to drop family_members table: %w", err)
		}
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

	return db, nil
}
