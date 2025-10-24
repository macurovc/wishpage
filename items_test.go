package main

import (
	"bytes"
	"context"
	"database/sql"
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
// Basic Item CRUD Tests
// ========================================

func TestGetItems(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	rr := httptest.NewRecorder()

	s.handleItems(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCreateItem(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpassword")

	cookie := loginAndGetCookie(t, s, "testpassword")

	// Setup: create a family member
	memberID := createFamilyMember(t, s, "testmember")

	form := url.Values{}
	form.Add("name", "test item")
	form.Add("family_member_id", "1")

	req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	s.handleItems(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify the item was created
	var name string
	err := s.db.QueryRowContext(t.Context(), "SELECT name FROM items WHERE id = ?", memberID).Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "test item", name)
	assert.Contains(t, rr.Body.String(), "test item")
}

func TestDeleteItem(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpassword")

	cookie := loginAndGetCookie(t, s, "testpassword")

	// Add a family member and an item to the database
	createFamilyMember(t, s, "testmember")
	createItem(t, s, 1, "test item", nil)

	req := httptest.NewRequest("DELETE", "/api/items/1", nil)
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.handleItemsID)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check if the item was actually deleted
	var count int
	err := s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM items WHERE id = 1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Check if the response no longer contains the deleted item
	assert.NotContains(t, rr.Body.String(), "test item")
}

// ========================================
// Reserve/Unreserve Tests
// ========================================

func TestReserveAndUnreserveItem(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpassword")

	cookie := loginAndGetCookie(t, s, "testpassword")

	// 1. Add a family member and an item
	createFamilyMember(t, s, "testmember")
	createItem(t, s, 1, "test item", nil)

	// 2. Reserve the item (no password)
	req := httptest.NewRequest("PUT", "/api/items/1/reserve", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.handleItemsID)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check if the item was actually reserved in the DB
	var reserved bool
	err := s.db.QueryRowContext(t.Context(), "SELECT reserved FROM items WHERE id = 1").Scan(&reserved)
	require.NoError(t, err)
	assert.True(t, reserved)

	// 3. Un-reserve the item (with password)
	req = httptest.NewRequest("PUT", "/api/items/1/unreserve", nil)
	req.AddCookie(cookie)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check if the item was actually un-reserved in the DB
	err = s.db.QueryRowContext(t.Context(), "SELECT reserved FROM items WHERE id = 1").Scan(&reserved)
	require.NoError(t, err)
	assert.False(t, reserved)
}

func TestUnreserveWithHeaderPreservesFilterAndUpdates(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "pw")

	// Two families; one item reserved under family 1
	_, err := s.db.ExecContext(t.Context(), `INSERT INTO family_members (name) VALUES ("fam1"), ("fam2")`)
	require.NoError(t, err)
	_, err = s.db.ExecContext(t.Context(), `INSERT INTO items (name, family_member_id, reserved) VALUES ("reserved item", 1, 1)`)
	require.NoError(t, err)

	cookie := loginAndGetCookie(t, s, "pw")

	req := httptest.NewRequest("PUT", "/api/items/1/unreserve", nil)
	req.AddCookie(cookie)
	req.Header.Set("X-Family-Member-ID", "1")

	rr := httptest.NewRecorder()
	http.HandlerFunc(s.handleItemsID).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// DB should reflect unreserved
	var reserved bool
	err = s.db.QueryRowContext(t.Context(), "SELECT reserved FROM items WHERE id = 1").Scan(&reserved)
	require.NoError(t, err)
	assert.False(t, reserved)

	// Response should still include the item (filter preserved)
	assert.Contains(t, rr.Body.String(), "reserved item")
}

// ========================================
// Data Validation Tests
// ========================================

func TestNullHandlingOnItemList(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "pw")

	_, err := s.db.ExecContext(t.Context(), `INSERT INTO family_members (name) VALUES ("alpha")`)
	require.NoError(t, err)
	// Insert an item with NULL link and price
	_, err = s.db.ExecContext(t.Context(), `INSERT INTO items (family_member_id, name, link, price, reserved) VALUES (1, "no extras", NULL, NULL, 0)`)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/items", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.handleItems)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "no extras")
}

func TestDeleteItemPreservesFilter(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	// Setup: create two family members with items
	aliceID := createFamilyMember(t, s, "Alice")
	bobID := createFamilyMember(t, s, "Bob")

	price10, price20, price30 := 10.0, 20.0, 30.0
	createItem(t, s, aliceID, "Alice Item 1", &price10)
	createItem(t, s, aliceID, "Alice Item 2", &price20)
	createItem(t, s, bobID, "Bob Item", &price30)

	// Delete Alice's first item with family filter header
	req := httptest.NewRequest(http.MethodDelete, "/api/items/1", nil)
	req.AddCookie(cookie)
	req.Header.Set("X-Family-Member-ID", "1")
	rr := httptest.NewRecorder()

	s.requireAuth(s.deleteItem)(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Alice Item 2")
	assert.NotContains(t, body, "Alice Item 1")
	assert.NotContains(t, body, "Bob Item")
}

// ========================================
// JSON Request Handling Tests
// ========================================

func TestCreateItemWithJSON(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	createFamilyMember(t, s, "Alice")

	payload := map[string]interface{}{
		"name":             "JSON Item",
		"family_member_id": 1,
		"price":            99.99,
		"link":             "https://example.com",
	}
	jsonBody, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/items", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	s.requireAuth(s.createItem)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify item was created
	var count int
	err = s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM items WHERE name = ?", "JSON Item").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Expected 1 item created")
}

func TestUpdateItemEmptyValuesToNull(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	_, err := s.db.ExecContext(t.Context(), "INSERT INTO family_members (name) VALUES (?)", "Alice")
	require.NoError(t, err)

	// Create item with link and price
	_, err = s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name, link, price) VALUES (?, ?, ?, ?)",
		1, "Test Item", "https://example.com", 50.0)
	require.NoError(t, err)

	// Update to clear link and price
	formData := url.Values{}
	formData.Set("name", "Test Item")
	formData.Set("link", "")
	formData.Set("price", "")

	req := httptest.NewRequest("PUT", "/api/items/1", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.updateItem)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify link and price are NULL
	var link sql.NullString
	var price sql.NullFloat64
	err = s.db.QueryRowContext(t.Context(), "SELECT link, price FROM items WHERE id = 1").Scan(&link, &price)
	require.NoError(t, err)
	assert.False(t, link.Valid, "Expected link to be NULL")
	assert.False(t, price.Valid, "Expected price to be NULL")
}

// ========================================
// Edge Case Tests
// ========================================

func TestCreateItemWithLongName(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	createFamilyMember(t, s, "Alice")

	longName := strings.Repeat("A", 500)
	formData := url.Values{}
	formData.Set("name", longName)
	formData.Set("family_member_id", "1")

	req := httptest.NewRequest("POST", "/api/items", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.createItem)(rr, req)

	// Should reject names over 200 characters
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Verify item was NOT created
	var count int
	err := s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM items WHERE family_member_id = 1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestCreateItemWithNegativePrice(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	createFamilyMember(t, s, "Alice")

	formData := url.Values{}
	formData.Set("name", "Negative Price Item")
	formData.Set("family_member_id", "1")
	formData.Set("price", "-50.00")

	req := httptest.NewRequest("POST", "/api/items", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.createItem)(rr, req)

	// Should reject negative prices
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Verify item was NOT created
	var count int
	err := s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM items WHERE name = ?", "Negative Price Item").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestDeleteItemWithInvalidID(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	req := httptest.NewRequest("DELETE", "/api/items/not-a-number", nil)
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.deleteItem)(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetAllItemsUsesContext(t *testing.T) {
	s := newTestServer(t)

	createFamilyMember(t, s, "Alice")
	createItem(t, s, 1, "Item", nil)

	// getAllItems now uses context properly
	ctx := context.Background()
	items, err := s.getAllItems(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, 1, len(items))
}

// ========================================
// Filter Preservation Tests
// ========================================

func TestCreateItemWithoutFamilyMemberDefaultsToFirst(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	formData := url.Values{}
	formData.Set("name", "Default Member Item")
	// No family_member_id provided

	req := httptest.NewRequest("POST", "/api/items", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.createItem)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var familyMemberID int
	err := s.db.QueryRowContext(t.Context(), "SELECT family_member_id FROM items WHERE name = ?", "Default Member Item").Scan(&familyMemberID)
	require.NoError(t, err)
	// Should default to first member (Alice, ID=1)
	assert.Equal(t, 1, familyMemberID)
}

func TestUpdateItemPreservesFilterWithHeader(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	// Create two family members
	createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	// Add items for both
	_, err := s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name, price) VALUES (?, ?, ?)", 1, "Alice Item 1", 10.0)
	require.NoError(t, err)
	_, err = s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name, price) VALUES (?, ?, ?)", 1, "Alice Item 2", 20.0)
	require.NoError(t, err)
	_, err = s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name, price) VALUES (?, ?, ?)", 2, "Bob Item", 30.0)
	require.NoError(t, err)

	// Update Alice's first item with family filter header
	formData := url.Values{}
	formData.Set("name", "Alice Item 1 Updated")
	formData.Set("price", "15.0")

	req := httptest.NewRequest("PUT", "/api/items/1", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	req.Header.Set("X-Family-Member-ID", "1")

	rr := httptest.NewRecorder()
	s.requireAuth(s.updateItem)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify the item was updated
	var name string
	var price float64
	err = s.db.QueryRowContext(t.Context(), "SELECT name, price FROM items WHERE id = 1").Scan(&name, &price)
	require.NoError(t, err)
	assert.Equal(t, "Alice Item 1 Updated", name)
	assert.Equal(t, 15.0, price)

	// Response should only contain Alice's items
	body := rr.Body.String()
	assert.Contains(t, body, "Alice Item 1 Updated")
	assert.Contains(t, body, "Alice Item 2")
	assert.NotContains(t, body, "Bob Item")
}

func TestUpdateItemPreservesFilterWithoutHeader(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	// Create two family members
	createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	// Add items for both
	_, err := s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name, price) VALUES (?, ?, ?)", 1, "Alice Item 1", 10.0)
	require.NoError(t, err)
	_, err = s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name, price) VALUES (?, ?, ?)", 1, "Alice Item 2", 20.0)
	require.NoError(t, err)
	_, err = s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name, price) VALUES (?, ?, ?)", 2, "Bob Item", 30.0)
	require.NoError(t, err)

	// Update Alice's first item WITHOUT family filter header
	formData := url.Values{}
	formData.Set("name", "Alice Item 1 Modified")

	req := httptest.NewRequest("PUT", "/api/items/1", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	// No X-Family-Member-ID header

	rr := httptest.NewRecorder()
	s.requireAuth(s.updateItem)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Response should still only contain Alice's items (derived from item's family_member_id)
	body := rr.Body.String()
	assert.Contains(t, body, "Alice Item 1 Modified")
	assert.Contains(t, body, "Alice Item 2")
	assert.NotContains(t, body, "Bob Item")
}

func TestUpdateItemChangeFamilyMember(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	// Create two family members
	createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	// Add item for Alice
	_, err := s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name, price) VALUES (?, ?, ?)", 1, "Movable Item", 10.0)
	require.NoError(t, err)

	// Move item from Alice (1) to Bob (2)
	formData := url.Values{}
	formData.Set("name", "Movable Item")
	formData.Set("family_member_id", "2")

	req := httptest.NewRequest("PUT", "/api/items/1", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	req.Header.Set("X-Family-Member-ID", "1") // Currently viewing Alice's items

	rr := httptest.NewRecorder()
	s.requireAuth(s.updateItem)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify item was moved to Bob
	var familyMemberID int
	err = s.db.QueryRowContext(t.Context(), "SELECT family_member_id FROM items WHERE id = 1").Scan(&familyMemberID)
	require.NoError(t, err)
	assert.Equal(t, 2, familyMemberID)

	// Response should show Alice's items (which no longer includes this item)
	body := rr.Body.String()
	assert.NotContains(t, body, "Movable Item")
}

func TestCreateItemUsesHeaderForFamilyMember(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpass")

	cookie := loginAndGetCookie(t, s, "testpass")

	// Create two family members
	createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	// Add existing item for Alice
	_, err := s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name) VALUES (?, ?)", 1, "Alice Existing Item")
	require.NoError(t, err)

	// Create new item with X-Family-Member-ID header pointing to Bob
	// (simulating clicking "Add Item" while viewing Bob's items)
	formData := url.Values{}
	formData.Set("name", "Bob's Item")

	req := httptest.NewRequest("POST", "/api/items", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	req.Header.Set("X-Family-Member-ID", "2") // Currently viewing Bob's items

	rr := httptest.NewRecorder()
	s.requireAuth(s.createItem)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify item was created for Bob (ID=2)
	var familyMemberID int
	var name string
	err = s.db.QueryRowContext(t.Context(), "SELECT family_member_id, name FROM items WHERE name = ?", "Bob's Item").Scan(&familyMemberID, &name)
	require.NoError(t, err)
	assert.Equal(t, 2, familyMemberID)
}

func TestCreateItemWithEmptyName(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpassword")
	cookie := loginAndGetCookie(t, s, "testpassword")
	createFamilyMember(t, s, "Alice")

	formData := url.Values{}
	formData.Set("name", "")
	formData.Set("family_member_id", "1")

	req := httptest.NewRequest("POST", "/api/items", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.createItem)(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "item name is required")
}

func TestCreateItemWithDuplicateName(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpassword")
	cookie := loginAndGetCookie(t, s, "testpassword")
	createFamilyMember(t, s, "Alice")

	// Create first item
	_, err := s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name) VALUES (?, ?)", 1, "Existing Item")
	require.NoError(t, err)

	// Try to create another item with the same name for the same family member
	formData := url.Values{}
	formData.Set("name", "Existing Item")
	formData.Set("family_member_id", "1")

	req := httptest.NewRequest("POST", "/api/items", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.createItem)(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "already exists")
}

func TestCreateItemWithSameNameDifferentFamilyMember(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpassword")
	cookie := loginAndGetCookie(t, s, "testpassword")
	createFamilyMember(t, s, "Alice")
	createFamilyMember(t, s, "Bob")

	// Create item for Alice
	_, err := s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name) VALUES (?, ?)", 1, "Shared Item Name")
	require.NoError(t, err)

	// Create item with same name for Bob - should succeed
	formData := url.Values{}
	formData.Set("name", "Shared Item Name")
	formData.Set("family_member_id", "2")

	req := httptest.NewRequest("POST", "/api/items", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.createItem)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify both items exist
	var count int
	err = s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM items WHERE name = ?", "Shared Item Name").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestUpdateItemWithDuplicateName(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpassword")
	cookie := loginAndGetCookie(t, s, "testpassword")
	createFamilyMember(t, s, "Alice")

	// Create two items
	_, err := s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name) VALUES (?, ?)", 1, "Item 1")
	require.NoError(t, err)
	_, err = s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name) VALUES (?, ?)", 1, "Item 2")
	require.NoError(t, err)

	// Try to rename Item 2 to Item 1
	formData := url.Values{}
	formData.Set("name", "Item 1")

	req := httptest.NewRequest("PUT", "/api/items/2", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.updateItem)(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "already exists")
}

func TestUpdateItemKeepingSameName(t *testing.T) {
	s := newTestServer(t)
	t.Setenv("EDIT_PASSWORD", "testpassword")
	cookie := loginAndGetCookie(t, s, "testpassword")
	createFamilyMember(t, s, "Alice")

	// Create item
	_, err := s.db.ExecContext(t.Context(), "INSERT INTO items (family_member_id, name, price) VALUES (?, ?, ?)", 1, "My Item", 10.0)
	require.NoError(t, err)

	// Update the price but keep the same name - should succeed
	formData := url.Values{}
	formData.Set("name", "My Item")
	formData.Set("price", "20.0")

	req := httptest.NewRequest("PUT", "/api/items/1", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)

	rr := httptest.NewRecorder()
	s.requireAuth(s.updateItem)(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify the price was updated
	var price float64
	err = s.db.QueryRowContext(t.Context(), "SELECT price FROM items WHERE id = 1").Scan(&price)
	require.NoError(t, err)
	assert.Equal(t, 20.0, price)
}
