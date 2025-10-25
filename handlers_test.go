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

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	s.handleIndex(rr, req)

	// Should show view page with collapsible family sections
	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Alice")
	assert.Contains(t, body, "Bob")
	// Should have collapsible sections
	assert.Contains(t, body, "family-section")
	assert.Contains(t, body, "<details")
	assert.Contains(t, body, "<summary")
	// All sections should be collapsed by default
	assert.NotContains(t, body, "open")
}

func TestHandleIndexEmptyState(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	s.handleIndex(rr, req)

	// Should show view page even with no family members
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Family Wishlist")
}

// ========================================
// Family View Tests
// ========================================

func TestHandleFamilyViewCollapsibleSections(t *testing.T) {
	s := newTestServer(t)

	aliceID := createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	price := 10.0
	createItem(t, s, aliceID, "Alice Item", &price)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	s.handleIndex(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()

	// Should have family member names
	assert.Contains(t, body, "Alice")
	// Should show Alice's item in a collapsible section
	assert.Contains(t, body, "Alice Item")
	assert.Contains(t, body, "family-section")
	assert.Contains(t, body, "<details")
	// Should show item count
	assert.Contains(t, body, "items")
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

	req := httptest.NewRequest(http.MethodGet, "/edit", http.NoBody)
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

	req := httptest.NewRequest(http.MethodGet, "/edit", http.NoBody)
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

	req := httptest.NewRequest(http.MethodGet, "/edit", http.NoBody)
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

func TestFamilyViewUsesCollapsibleSections(t *testing.T) {
	s := newTestServer(t)

	createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	s.handleIndex(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()

	// Should use collapsible details/summary elements
	assert.Contains(t, body, "<details")
	assert.Contains(t, body, "<summary")
	// Should have family names
	assert.Contains(t, body, "Alice")
	assert.Contains(t, body, "Bob")
}

func TestAllFamilySectionsCollapsedByDefault(t *testing.T) {
	s := newTestServer(t)

	// Create multiple family members with items
	aliceID := createFamilyMember(t, s, "Alice")
	bobID := createFamilyMember(t, s, "Bob")

	price1, price2 := 10.0, 20.0
	createItem(t, s, aliceID, "Alice's First Item", &price1)
	createItem(t, s, bobID, "Bob's Item", &price2)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	s.handleIndex(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()

	// Both items should be in the HTML
	assert.Contains(t, body, "Alice&#39;s First Item")
	assert.Contains(t, body, "Bob&#39;s Item")

	// Should have collapsible sections
	assert.Contains(t, body, "<details")
	assert.Contains(t, body, "family-section")

	// All sections should be collapsed by default
	assert.NotContains(t, body, "open")

	// Should show item counts
	assert.Contains(t, body, "1 items") // Alice
	assert.Contains(t, body, "1 items") // Bob
}
