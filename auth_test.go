package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// Token Generation Tests
// ========================================

func TestGenerateToken(t *testing.T) {
	token1, err1 := generateToken()
	token2, err2 := generateToken()

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEmpty(t, token1)
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, token1, token2, "Tokens should be unique")
	assert.Greater(t, len(token1), 40, "Token should be sufficiently long")
}

// ========================================
// Session Store Tests
// ========================================

func TestSessionStoreCreateAndValidate(t *testing.T) {
	store := newSessionStore()
	token := "test-token-123"
	expiry := time.Now().Add(1 * time.Hour)

	// Create session
	store.create(token, expiry)

	// Validate session
	assert.True(t, store.isValid(token))
	assert.False(t, store.isValid("invalid-token"))
}

func TestSessionStoreExpiry(t *testing.T) {
	store := newSessionStore()
	testToken := "test-token-expired"        // #nosec G101 -- This is a test token, not a real credential
	expiry := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago

	store.create(testToken, expiry)

	// Should be invalid because it's expired
	assert.False(t, store.isValid(testToken))
}

func TestSessionStoreDelete(t *testing.T) {
	store := newSessionStore()
	token := "test-token-delete"
	expiry := time.Now().Add(1 * time.Hour)

	store.create(token, expiry)
	assert.True(t, store.isValid(token))

	store.delete(token)
	assert.False(t, store.isValid(token))
}

func TestSessionStoreCleanup(t *testing.T) {
	store := newSessionStore()

	validToken := "valid-token"
	expiredToken := "expired-token"

	store.create(validToken, time.Now().Add(1*time.Hour))
	store.create(expiredToken, time.Now().Add(-1*time.Hour))

	// Before cleanup
	assert.True(t, store.isValid(validToken))
	assert.False(t, store.isValid(expiredToken))

	// Run cleanup
	store.cleanup()

	// After cleanup, expired token should be removed
	store.mu.RLock()
	_, existsValid := store.sessions[validToken]
	_, existsExpired := store.sessions[expiredToken]
	store.mu.RUnlock()

	assert.True(t, existsValid, "Valid token should still exist")
	assert.False(t, existsExpired, "Expired token should be removed")
}

func TestSessionStoreConcurrency(_ *testing.T) {
	store := newSessionStore()
	token := "concurrent-token"
	expiry := time.Now().Add(1 * time.Hour)

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			store.create(token, expiry)
			store.isValid(token)
			store.delete(token)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// ========================================
// Login Page Tests
// ========================================

func TestLoginPageDisplays(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	s.handleLoginPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Family Wishlist")
	assert.Contains(t, rr.Body.String(), "Enter password")
}

func TestLoginPageWithError(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/login?error=password", nil)
	rr := httptest.NewRecorder()

	s.handleLoginPage(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Incorrect password")
}

func TestLoginPageRedirectsIfAuthenticated(t *testing.T) {
	t.Setenv("EDIT_PASSWORD", "testpass")
	s := newTestServer(t)

	// Create a valid session
	token := "valid-token"
	s.sessions.create(token, time.Now().Add(1*time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: token,
	})
	rr := httptest.NewRecorder()

	s.handleLoginPage(rr, req)

	// Should redirect to edit page
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/edit", rr.Header().Get("Location"))
}

// ========================================
// Login Endpoint Tests
// ========================================

func TestLoginSuccess(t *testing.T) {
	t.Setenv("EDIT_PASSWORD", "testpass123")
	s := newTestServer(t)

	form := url.Values{}
	form.Add("password", "testpass123")

	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.handleLogin(rr, req)

	// Should redirect to /edit
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/edit", rr.Header().Get("Location"))

	// Check that cookie was set
	cookies := rr.Result().Cookies()
	require.Len(t, cookies, 1, "Should set exactly one cookie")

	cookie := cookies[0]
	assert.Equal(t, "session_token", cookie.Name)
	assert.NotEmpty(t, cookie.Value)
	assert.True(t, cookie.HttpOnly, "Cookie should be HTTP-only")
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
	assert.Equal(t, 86400, cookie.MaxAge, "Cookie should have 24 hour max age")

	// Verify token is valid in session store
	assert.True(t, s.sessions.isValid(cookie.Value))
}

func TestLoginIncorrectPassword(t *testing.T) {
	t.Setenv("EDIT_PASSWORD", "correctpass")
	s := newTestServer(t)

	form := url.Values{}
	form.Add("password", "wrongpass")

	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.handleLogin(rr, req)

	// Should redirect back to login page with error
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/login?error=password", rr.Header().Get("Location"))

	// Should not set any cookies
	cookies := rr.Result().Cookies()
	assert.Empty(t, cookies, "Should not set cookie on failed login")
}

func TestLoginPasswordNotConfigured(t *testing.T) {
	t.Setenv("EDIT_PASSWORD", "")
	s := newTestServer(t)

	form := url.Values{}
	form.Add("password", "anypass")

	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.handleLogin(rr, req)

	// Should redirect back to login page with error
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/login?error=config", rr.Header().Get("Location"))
}

func TestLoginInvalidMethod(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/login", nil)
	rr := httptest.NewRecorder()

	s.handleLogin(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestLoginInvalidFormData(t *testing.T) {
	t.Setenv("EDIT_PASSWORD", "testpass")
	s := newTestServer(t)

	// Send malformed form data
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader("%invalid%"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.handleLogin(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ========================================
// Logout Endpoint Tests
// ========================================

func TestLogoutSuccess(t *testing.T) {
	t.Setenv("EDIT_PASSWORD", "testpass")
	s := newTestServer(t)

	// First login to get a valid session
	form := url.Values{}
	form.Add("password", "testpass")

	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRR := httptest.NewRecorder()
	s.handleLogin(loginRR, loginReq)

	cookie := loginRR.Result().Cookies()[0]
	token := cookie.Value

	// Verify session exists
	assert.True(t, s.sessions.isValid(token))

	// Now logout
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutRR := httptest.NewRecorder()
	s.handleLogout(logoutRR, logoutReq)

	// Should redirect to home
	assert.Equal(t, http.StatusSeeOther, logoutRR.Code)
	assert.Equal(t, "/", logoutRR.Header().Get("Location"))

	// Verify session was deleted
	assert.False(t, s.sessions.isValid(token))

	// Check that cookie was cleared
	cookies := logoutRR.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "session_token", cookies[0].Name)
	assert.Equal(t, -1, cookies[0].MaxAge, "Cookie should be deleted")
}

func TestLogoutWithoutCookie(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	rr := httptest.NewRecorder()

	s.handleLogout(rr, req)

	// Should redirect even without cookie
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/", rr.Header().Get("Location"))
}

func TestLogoutInvalidMethod(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/logout", nil)
	rr := httptest.NewRecorder()

	s.handleLogout(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

// ========================================
// Authentication Middleware Tests
// ========================================

func TestRequireAuthWithValidSession(t *testing.T) {
	s := newTestServer(t)

	// Create a valid session
	token := "valid-session-token"
	s.sessions.create(token, time.Now().Add(1*time.Hour))

	// Create a test handler
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with requireAuth middleware
	protectedHandler := s.requireAuth(testHandler)

	// Make request with valid session cookie
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: token,
	})
	rr := httptest.NewRecorder()

	protectedHandler(rr, req)

	assert.True(t, handlerCalled, "Handler should be called with valid session")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireAuthWithoutCookie(t *testing.T) {
	s := newTestServer(t)

	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("Handler should not be called")
	})

	protectedHandler := s.requireAuth(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()

	protectedHandler(rr, req)

	// Should redirect to login
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/login", rr.Header().Get("Location"))
}

func TestRequireAuthWithInvalidToken(t *testing.T) {
	s := newTestServer(t)

	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("Handler should not be called")
	})

	protectedHandler := s.requireAuth(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: "invalid-token",
	})
	rr := httptest.NewRecorder()

	protectedHandler(rr, req)

	// Should redirect to login
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/login", rr.Header().Get("Location"))
}

func TestRequireAuthWithExpiredToken(t *testing.T) {
	s := newTestServer(t)

	// Create an expired session
	token := "expired-token"
	s.sessions.create(token, time.Now().Add(-1*time.Hour))

	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("Handler should not be called")
	})

	protectedHandler := s.requireAuth(testHandler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: token,
	})
	rr := httptest.NewRecorder()

	protectedHandler(rr, req)

	// Should redirect to login
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/login", rr.Header().Get("Location"))
}

// ========================================
// Integration Tests
// ========================================

func TestLoginLogoutFlow(t *testing.T) {
	t.Setenv("EDIT_PASSWORD", "testpass")
	s := newTestServer(t)

	// 1. Login
	form := url.Values{}
	form.Add("password", "testpass")
	loginReq := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRR := httptest.NewRecorder()
	s.handleLogin(loginRR, loginReq)

	assert.Equal(t, http.StatusSeeOther, loginRR.Code)
	assert.Equal(t, "/edit", loginRR.Header().Get("Location"))
	cookies := loginRR.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie := cookies[0]

	// 2. Access protected resource with session
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Protected content")); err != nil {
			t.Logf("Error writing response: %v", err)
		}
	})
	protectedHandler := s.requireAuth(testHandler)

	accessReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	accessReq.AddCookie(sessionCookie)
	accessRR := httptest.NewRecorder()
	protectedHandler(accessRR, accessReq)

	assert.Equal(t, http.StatusOK, accessRR.Code)
	assert.Contains(t, accessRR.Body.String(), "Protected content")

	// 3. Logout
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	logoutReq.AddCookie(sessionCookie)
	logoutRR := httptest.NewRecorder()
	s.handleLogout(logoutRR, logoutReq)

	assert.Equal(t, http.StatusSeeOther, logoutRR.Code)
	assert.Equal(t, "/", logoutRR.Header().Get("Location"))

	// 4. Try to access protected resource after logout
	accessReq2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	accessReq2.AddCookie(sessionCookie)
	accessRR2 := httptest.NewRecorder()
	protectedHandler(accessRR2, accessReq2)

	assert.Equal(t, http.StatusSeeOther, accessRR2.Code)
	assert.Equal(t, "/login", accessRR2.Header().Get("Location"))
}

func TestMultipleSessionsConcurrently(t *testing.T) {
	t.Setenv("EDIT_PASSWORD", "testpass")
	s := newTestServer(t)

	// Create multiple sessions concurrently
	const numSessions = 10
	done := make(chan string, numSessions)

	for i := 0; i < numSessions; i++ {
		go func() {
			form := url.Values{}
			form.Add("password", "testpass")
			req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()
			s.handleLogin(rr, req)

			if rr.Code == http.StatusSeeOther {
				cookies := rr.Result().Cookies()
				if len(cookies) > 0 {
					done <- cookies[0].Value
				} else {
					done <- ""
				}
			} else {
				done <- ""
			}
		}()
	}

	// Collect all tokens
	tokens := make(map[string]bool)
	for i := 0; i < numSessions; i++ {
		token := <-done
		if token != "" {
			tokens[token] = true
		}
	}

	// All tokens should be unique
	assert.Equal(t, numSessions, len(tokens), "All sessions should have unique tokens")

	// All tokens should be valid
	for token := range tokens {
		assert.True(t, s.sessions.isValid(token))
	}
}
