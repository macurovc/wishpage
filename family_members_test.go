package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// Basic Family Member CRUD Tests
// ========================================

func TestGetFamilyMembers(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/family_members", nil)
	rr := httptest.NewRecorder()

	s.handleFamilyMembers(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCreateFamilyMember(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpassword")

	cookie := loginAndGetCookie(t, s, "testpassword")

	form := url.Values{}
	form.Add("name", "new member")

	req := httptest.NewRequest("POST", "/api/family_members", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.handleFamilyMembers)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check if the member was created
	var name string
	err := s.db.QueryRow("SELECT name FROM family_members WHERE name = 'new member'").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "new member", name)

	// Check if the response contains the new member
	assert.Contains(t, rr.Body.String(), "new member")
}

func TestCreateFamilyMemberJSONPassword(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "pw")

	cookie := loginAndGetCookie(t, s, "pw")

	body := strings.NewReader(`{"name":"json member"}`)
	req := httptest.NewRequest("POST", "/api/family_members", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.handleFamilyMembers)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify created
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM family_members WHERE name='json member'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestHeaderBasedEditPassword(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "pw")

	cookie := loginAndGetCookie(t, s, "pw")

	// Add a family member via cookie auth
	body := strings.NewReader(`{"name":"header user"}`)
	req := httptest.NewRequest("POST", "/api/family_members", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	http.HandlerFunc(s.handleFamilyMembers).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM family_members WHERE name='header user'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestDeleteFamilyMember(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpassword")

	cookie := loginAndGetCookie(t, s, "testpassword")

	// Add a member to delete
	createFamilyMember(t, s, "deleteme")

	form := url.Values{}

	req := httptest.NewRequest("DELETE", "/api/family_members/1", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.handleFamilyMembersID)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check if the member was deleted
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM family_members WHERE name = 'deleteme'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// ========================================
// Cascade Delete Tests
// ========================================

func TestCascadeDeleteFamilyMemberDeletesItems(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "pw")

	cookie := loginAndGetCookie(t, s, "pw")

	_, err := s.db.Exec(`INSERT INTO family_members (name) VALUES ("parent")`)
	require.NoError(t, err)
	_, err = s.db.Exec(`INSERT INTO items (family_member_id, name) VALUES (1, "child item")`)
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/api/family_members/1", nil)
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.handleFamilyMembersID)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// ========================================
// Template Rendering Tests
// ========================================

func TestCreateFamilyMemberUpdatesBothTargets(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "pw")

	cookie := loginAndGetCookie(t, s, "pw")

	// Submit as form, matching HTMX default
	form := url.Values{}
	form.Add("name", "newbie")
	req := httptest.NewRequest("POST", "/api/family_members", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.handleFamilyMembers)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	body := rr.Body.String()
	// Should return a new FamilyMemberSection with collapsible details
	assert.Contains(t, body, "newbie")
	assert.Contains(t, body, "family-section")
	assert.Contains(t, body, "<details")
}

// ========================================
// Form Parsing Tests
// ========================================

func TestCreateFamilyMemberWithForm(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	formData := url.Values{}
	formData.Set("name", "Form Member")

	req := httptest.NewRequest("POST", "/api/family_members", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.createFamilyMember)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify member was created
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM family_members WHERE name = ?", "Form Member").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// ========================================
// Edge Case Tests
// ========================================

func TestCreateFamilyMemberWithSpecialChars(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	specialName := "O'Brien & Sons <script>alert('xss')</script>"
	formData := url.Values{}
	formData.Set("name", specialName)

	req := httptest.NewRequest("POST", "/api/family_members", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.createFamilyMember)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify member was created with special chars preserved
	var name string
	err := s.db.QueryRow("SELECT name FROM family_members WHERE id = 1").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, specialName, name)
}

func TestUpdateFamilyMemberWithJSON(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	createFamilyMember(t, s, "OldName")

	payload := map[string]string{"name": "NewName"}
	jsonBody, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/api/family_members/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.updateFamilyMember)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var name string
	err = s.db.QueryRow("SELECT name FROM family_members WHERE id = 1").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "NewName", name)
}

func TestCreateFamilyMemberDuplicateName(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	// Create first family member
	createFamilyMember(t, s, "Alice")

	// Try to create another with the same name
	formData := url.Values{}
	formData.Set("name", "Alice")

	req := httptest.NewRequest("POST", "/api/family_members", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.createFamilyMember)(rr, req)

	// Should reject duplicate names
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "already exists")

	// Verify only one Alice exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM family_members WHERE name = ?", "Alice").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
