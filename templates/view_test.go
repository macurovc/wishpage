package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"wishpage/models"
)

func TestItemCardViewSanitizesUnsafeLink(t *testing.T) {
	t.Parallel()

	link := "javascript:alert(document.domain)"
	item := models.Item{ID: 1, Name: "Unsafe link", Link: &link}

	var rendered bytes.Buffer
	if err := ItemCardView(item, nil).Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render item card: %v", err)
	}

	html := rendered.String()
	if strings.Contains(html, link) {
		t.Fatalf("unsafe URL was rendered without sanitization: %s", html)
	}
	if !strings.Contains(html, "about:invalid#TemplFailedSanitizationURL") {
		t.Fatalf("unsafe URL was not replaced with templ's sanitization marker: %s", html)
	}
}

func TestItemCardViewPreservesHTTPSLink(t *testing.T) {
	t.Parallel()

	link := "https://example.com/wishlist-item"
	item := models.Item{ID: 1, Name: "Safe link", Link: &link}

	var rendered bytes.Buffer
	if err := ItemCardView(item, nil).Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render item card: %v", err)
	}

	html := rendered.String()
	if !strings.Contains(html, `href="`+link+`"`) {
		t.Fatalf("HTTPS URL was not preserved: %s", html)
	}
	if !strings.Contains(html, `rel="noopener noreferrer"`) {
		t.Fatalf("external link is missing rel protection: %s", html)
	}
}
