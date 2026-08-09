package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const (
	testItemNameField = "name"
	testItemLinkField = "link"
)

func TestValidateItemLinkAcceptsFragment(t *testing.T) {
	t.Parallel()

	const link = "https://example.com#section"
	validated, err := validateItemLink(link)
	if err != nil {
		t.Fatalf("validate URL with fragment: %v", err)
	}
	if validated != link {
		t.Fatalf("validated link = %q, want %q", validated, link)
	}
}

func TestItemWritesRejectUnsafeLinks(t *testing.T) {
	t.Setenv("EDIT_PASSWORD", "testpassword")
	s := newTestServer(t)
	cookie := loginAndGetCookie(t, s, "testpassword")
	createFamilyMember(t, s, "Alice")

	createForm := url.Values{
		testItemNameField:  {"Unsafe item"},
		"family_member_id": {"1"},
		testItemLinkField:  {"javascript:alert(document.domain)"},
	}
	createRequest := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/items", strings.NewReader(createForm.Encode()))
	createRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createRequest.AddCookie(cookie)
	createResponse := httptest.NewRecorder()
	s.mux.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusBadRequest {
		t.Fatalf("create status = %d, want %d", createResponse.Code, http.StatusBadRequest)
	}

	createItem(t, s, 1, "Existing item", nil)
	updateForm := url.Values{
		testItemNameField: {"Existing item"},
		testItemLinkField: {"data:text/html,<script>alert(document.domain)</script>"},
	}
	updateRequest := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/api/items/1", strings.NewReader(updateForm.Encode()))
	updateRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateRequest.AddCookie(cookie)
	updateResponse := httptest.NewRecorder()
	s.mux.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusBadRequest {
		t.Fatalf("update status = %d, want %d", updateResponse.Code, http.StatusBadRequest)
	}

	var unsafeLinkCount int
	if queryErr := s.db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM items WHERE link IS NOT NULL").Scan(&unsafeLinkCount); queryErr != nil {
		t.Fatalf("count stored links: %v", queryErr)
	}
	if unsafeLinkCount != 0 {
		t.Fatalf("stored link count = %d, want 0", unsafeLinkCount)
	}
}
