package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTMXIsServedLocally(t *testing.T) {
	s := newTestServer(t)

	pageRequest := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	pageResponse := httptest.NewRecorder()
	s.mux.ServeHTTP(pageResponse, pageRequest)

	if pageResponse.Code != http.StatusOK {
		t.Fatalf("page status = %d, want %d", pageResponse.Code, http.StatusOK)
	}
	page := pageResponse.Body.String()
	if !strings.Contains(page, `<script src="/static/js/htmx.min.js"></script>`) {
		t.Fatalf("page does not reference the local HTMX runtime")
	}
	if strings.Contains(page, `<script src="http`) {
		t.Fatalf("page contains a remote executable script")
	}

	assetRequest := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/static/js/htmx.min.js", http.NoBody)
	assetResponse := httptest.NewRecorder()
	s.mux.ServeHTTP(assetResponse, assetRequest)

	if assetResponse.Code != http.StatusOK {
		t.Fatalf("HTMX asset status = %d, want %d", assetResponse.Code, http.StatusOK)
	}
	if !strings.Contains(assetResponse.Body.String(), "e.htmx=e.htmx||t()") {
		t.Fatalf("served HTMX asset does not contain the expected runtime")
	}
}
