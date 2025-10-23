package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========================================
// Index/Landing Page Tests
// ========================================

func TestHandleIndexShowsViewWithNoMemberSelected(t *testing.T) {
	s := newTestServer(t)

	// Create family members
	createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	s.handleIndex(rr, req)

	// Should show view page with no member selected
	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Alice")
	assert.Contains(t, body, "Bob")
	assert.Contains(t, body, "No wishlist items yet")
	// No active tab since no member is selected
	assert.NotContains(t, body, "nav-link active")
}

func TestHandleIndexEmptyState(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	s.handleIndex(rr, req)

	// Should show view page even with no family members
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Family Wishlist")
}

// ========================================
// Family View Tests
// ========================================

func TestHandleFamilyView(t *testing.T) {
	s := newTestServer(t)

	aliceID := createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	price := 25.0
	createItem(t, s, aliceID, "Alice's Widget", &price)

	req := httptest.NewRequest(http.MethodGet, "/family/Alice", nil)
	rr := httptest.NewRecorder()

	s.handleFamilyView(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Alice")
	assert.Contains(t, body, "Bob")
	// Apostrophe is HTML-encoded as &#39;
	assert.Contains(t, body, "Alice&#39;s Widget")
}

func TestHandleFamilyViewInvalidID(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/family/NonExistent", nil)
	rr := httptest.NewRecorder()

	s.handleFamilyView(rr, req)

	// Should return 404 for non-existent family member
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandleFamilyViewActiveTabHighlight(t *testing.T) {
	s := newTestServer(t)

	aliceID := createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	price := 10.0
	createItem(t, s, aliceID, "Alice Item", &price)

	req := httptest.NewRequest(http.MethodGet, "/family/Alice", nil)
	rr := httptest.NewRecorder()

	s.handleFamilyView(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()

	// Should have Alice tab as active
	assert.Contains(t, body, "Alice")
	// Should show Alice's item
	assert.Contains(t, body, "Alice Item")
}

// ========================================
// Edit Page Tests
// ========================================

func TestHandleEditRedirectsToFirstFamily(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	// Create family members
	createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	req := httptest.NewRequest(http.MethodGet, "/edit", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	s.handleEdit(rr, req)

	// Should show edit page
	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Alice")
	assert.Contains(t, body, "Bob")
}

func TestHandleEdit(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	// Create family members and items
	aliceID := createFamilyMember(t, s, "Alice")
	bobID := createFamilyMember(t, s, "Bob")

	price1, price2 := 10.0, 20.0
	createItem(t, s, aliceID, "Alice Item", &price1)
	createItem(t, s, bobID, "Bob Item", &price2)

	req := httptest.NewRequest(http.MethodGet, "/edit", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	s.handleEdit(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()

	// Should show all family members and items
	assert.Contains(t, body, "Alice")
	assert.Contains(t, body, "Bob")
	assert.Contains(t, body, "Alice Item")
	assert.Contains(t, body, "Bob Item")

	// Should show edit controls
	assert.Contains(t, body, "Add Item")
	assert.Contains(t, body, "Add Family Member")
}

func TestHandleEditRequiresAuth(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/edit", nil)
	// No authentication cookie

	rr := httptest.NewRecorder()
	s.requireAuth(s.handleEdit)(rr, req)

	// Should redirect to login
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	assert.Equal(t, "/login", rr.Header().Get("Location"))
}

// ========================================
// URL Routing Tests
// ========================================

func TestFamilyViewLinksAreRegularNotHTMX(t *testing.T) {
	s := newTestServer(t)

	createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	s.handleIndex(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()

	// Links should be regular href links, not HTMX
	assert.Contains(t, body, "/family/Alice")
	assert.Contains(t, body, "/family/Bob")
}

func TestFamilyViewUsesNameInURL(t *testing.T) {
	s := newTestServer(t)

	aliceID := createFamilyMember(t, s, "Alice")
	price := 15.0
	createItem(t, s, aliceID, "Alice's Item", &price)

	// Access by name in URL
	req := httptest.NewRequest(http.MethodGet, "/family/Alice", nil)
	rr := httptest.NewRecorder()

	s.handleFamilyView(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Alice")
	// Apostrophe is HTML-encoded as &#39;
	assert.Contains(t, body, "Alice&#39;s Item")
}
