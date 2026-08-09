// Package main implements a wishlist web application with authentication and email notifications.
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"wishpage/templates"
)

const sessionCookieName = "session_token"

func setSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
		Path:     "/",
	})
}

// Session management.
type sessionStore struct {
	sessions map[string]time.Time // token -> expiry
	mu       sync.RWMutex
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		sessions: make(map[string]time.Time),
	}
}

func (s *sessionStore) create(token string, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = expiry
}

func (s *sessionStore) isValid(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	expiry, exists := s.sessions[token]
	if !exists {
		return false
	}
	return time.Now().Before(expiry)
}

func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *sessionStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, expiry := range s.sessions {
		if now.After(expiry) {
			delete(s.sessions, token)
		}
	}
}

// generateToken creates a secure random token for session management.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// requireAuth is middleware that requires a valid session cookie.
func (s *server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || !s.sessions.isValid(cookie.Value) {
			// For page requests, redirect to login
			// For API requests, return error
			if strings.HasPrefix(r.URL.Path, "/api/") {
				w.WriteHeader(http.StatusUnauthorized)
				if err := templates.Error("Unauthorized: Please log in").Render(r.Context(), w); err != nil {
					log.Printf("Error rendering template: %v", err)
				}
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (s *server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// Display the login page
	if r.URL.Path != "/login" {
		http.NotFound(w, r)
		return
	}

	// If already authenticated, redirect to edit page
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && s.sessions.isValid(cookie.Value) {
		http.Redirect(w, r, "/edit", http.StatusSeeOther)
		return
	}

	// Get error message from query params
	errorMsg := r.URL.Query().Get("error")

	if err := templates.Login(errorMsg).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering template: %v", err)
	}
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := templates.Error("Method not allowed").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}

	// Parse password from form
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if err := templates.Error("Invalid form data").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}

	password := r.FormValue("password")
	correctPassword := os.Getenv("EDIT_PASSWORD")

	if correctPassword == "" {
		// Redirect back to login page with error
		http.Redirect(w, r, "/login?error=config", http.StatusSeeOther)
		return
	}

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(password), []byte(correctPassword)) != 1 {
		// Redirect back to login page with error
		http.Redirect(w, r, "/login?error=password", http.StatusSeeOther)
		return
	}

	// Generate secure session token
	token, err := generateToken()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := templates.Error("Failed to create session").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}

	// Store session with 24 hour expiry
	s.sessions.create(token, time.Now().Add(24*time.Hour))

	setSessionCookie(w, token, 86400)

	// Redirect to edit page
	http.Redirect(w, r, "/edit", http.StatusSeeOther)
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := templates.Error("Method not allowed").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}

	// Get and delete session token
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.sessions.delete(cookie.Value)
	}

	setSessionCookie(w, "", -1)

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
