package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// Test helpers

func newTestServer(t *testing.T) *server {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err, "Failed to open test database")
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("Error closing test database: %v", err)
		}
	})

	_, err = db.ExecContext(t.Context(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err, "Failed to enable foreign keys")

	// Database initialization (create tables if they don't exist)
	_, err = db.ExecContext(t.Context(), `
        CREATE TABLE IF NOT EXISTS family_members (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL UNIQUE
        );
    `)
	require.NoError(t, err, "Failed to create family_members table")

	_, err = db.ExecContext(t.Context(), `
        CREATE TABLE IF NOT EXISTS items (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            family_member_id INTEGER NOT NULL,
            name TEXT NOT NULL,
            link TEXT,
            price REAL,
            reserved BOOLEAN DEFAULT 0,
            FOREIGN KEY (family_member_id) REFERENCES family_members(id) ON DELETE CASCADE
        );
    `)
	require.NoError(t, err, "Failed to create items table")

	s := &server{
		db:           db,
		mux:          http.NewServeMux(),
		sessions:     newSessionStore(),
		emailService: &emailService{enabled: false}, // Disabled for tests
	}
	s.routes()
	return s
}

// createFamilyMember is a test helper to create a family member in the database.
func createFamilyMember(t *testing.T, s *server, name string) int {
	t.Helper()

	result, err := s.db.ExecContext(t.Context(), "INSERT INTO family_members (name) VALUES (?)", name)
	require.NoError(t, err, "Failed to create family member %q", name)

	id, err := result.LastInsertId()
	require.NoError(t, err, "Failed to get last insert ID")

	return int(id)
}

// createItem is a test helper to create an item in the database.
func createItem(t *testing.T, s *server, familyMemberID int, name string, price *float64) {
	t.Helper()

	_, err := s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name, price) VALUES (?, ?, ?)",
		familyMemberID, name, price)
	require.NoError(t, err, "Failed to create item %q", name)
}

// loginAndGetCookie is a test helper to login and get a session cookie.
func loginAndGetCookie(t *testing.T, s *server, password string) *http.Cookie {
	t.Helper()

	form := url.Values{}
	form.Add("password", password)

	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.handleLogin(rr, req)

	cookies := rr.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			return cookie
		}
	}

	t.Fatal("No session cookie found after login")
	return nil
}
