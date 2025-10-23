package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailService_Disabled(t *testing.T) {
	// Save original env vars
	origUser := os.Getenv("EMAIL_USER")
	origPass := os.Getenv("EMAIL_PASS")
	origTo := os.Getenv("EMAIL_TO")

	// Cleanup
	defer func() {
		os.Setenv("EMAIL_USER", origUser)
		os.Setenv("EMAIL_PASS", origPass)
		os.Setenv("EMAIL_TO", origTo)
	}()

	tests := []struct {
		name        string
		emailUser   string
		emailPass   string
		emailTo     string
		description string
	}{
		{
			name:        "no credentials",
			emailUser:   "",
			emailPass:   "",
			emailTo:     "",
			description: "should disable when no credentials provided",
		},
		{
			name:        "missing password",
			emailUser:   "user@example.com",
			emailPass:   "",
			emailTo:     "to@example.com",
			description: "should disable when password is missing",
		},
		{
			name:        "missing user",
			emailUser:   "",
			emailPass:   "password",
			emailTo:     "to@example.com",
			description: "should disable when user is missing",
		},
		{
			name:        "missing recipient",
			emailUser:   "user@example.com",
			emailPass:   "password",
			emailTo:     "",
			description: "should disable when recipient is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("EMAIL_USER", tt.emailUser)
			os.Setenv("EMAIL_PASS", tt.emailPass)
			os.Setenv("EMAIL_TO", tt.emailTo)

			service := newEmailService()
			assert.False(t, service.enabled, tt.description)
		})
	}
}

func TestNewEmailService_Enabled(t *testing.T) {
	// Save original env vars
	origUser := os.Getenv("EMAIL_USER")
	origPass := os.Getenv("EMAIL_PASS")
	origTo := os.Getenv("EMAIL_TO")
	origHost := os.Getenv("EMAIL_HOST")
	origPort := os.Getenv("EMAIL_PORT")
	origFrom := os.Getenv("EMAIL_FROM")

	// Cleanup
	defer func() {
		os.Setenv("EMAIL_USER", origUser)
		os.Setenv("EMAIL_PASS", origPass)
		os.Setenv("EMAIL_TO", origTo)
		os.Setenv("EMAIL_HOST", origHost)
		os.Setenv("EMAIL_PORT", origPort)
		os.Setenv("EMAIL_FROM", origFrom)
	}()

	os.Setenv("EMAIL_USER", "user@example.com")
	os.Setenv("EMAIL_PASS", "password123")
	os.Setenv("EMAIL_TO", "recipient@example.com")
	os.Setenv("EMAIL_HOST", "smtp.example.com")
	os.Setenv("EMAIL_PORT", "25")
	os.Setenv("EMAIL_FROM", "sender@example.com")

	service := newEmailService()

	require.True(t, service.enabled, "service should be enabled with valid credentials")
	assert.Equal(t, "smtp.example.com", service.host)
	assert.Equal(t, 25, service.port)
	assert.Equal(t, "user@example.com", service.user)
	assert.Equal(t, "password123", service.password)
	assert.Equal(t, "sender@example.com", service.from)
	assert.Equal(t, "recipient@example.com", service.to)
}

func TestNewEmailService_Defaults(t *testing.T) {
	// Save original env vars
	origUser := os.Getenv("EMAIL_USER")
	origPass := os.Getenv("EMAIL_PASS")
	origTo := os.Getenv("EMAIL_TO")
	origHost := os.Getenv("EMAIL_HOST")
	origPort := os.Getenv("EMAIL_PORT")
	origFrom := os.Getenv("EMAIL_FROM")

	// Cleanup
	defer func() {
		os.Setenv("EMAIL_USER", origUser)
		os.Setenv("EMAIL_PASS", origPass)
		os.Setenv("EMAIL_TO", origTo)
		os.Setenv("EMAIL_HOST", origHost)
		os.Setenv("EMAIL_PORT", origPort)
		os.Setenv("EMAIL_FROM", origFrom)
	}()

	// Set only required fields
	os.Setenv("EMAIL_USER", "user@example.com")
	os.Setenv("EMAIL_PASS", "password123")
	os.Setenv("EMAIL_TO", "recipient@example.com")
	os.Unsetenv("EMAIL_HOST")
	os.Unsetenv("EMAIL_PORT")
	os.Unsetenv("EMAIL_FROM")

	service := newEmailService()

	require.True(t, service.enabled, "service should be enabled")
	assert.Equal(t, "smtp.gmail.com", service.host, "should use default host")
	assert.Equal(t, 587, service.port, "should use default port")
	assert.Equal(t, "user@example.com", service.from, "should use EMAIL_USER as default for EMAIL_FROM")
}

func TestNewEmailService_InvalidPort(t *testing.T) {
	// Save original env vars
	origUser := os.Getenv("EMAIL_USER")
	origPass := os.Getenv("EMAIL_PASS")
	origTo := os.Getenv("EMAIL_TO")
	origPort := os.Getenv("EMAIL_PORT")

	// Cleanup
	defer func() {
		os.Setenv("EMAIL_USER", origUser)
		os.Setenv("EMAIL_PASS", origPass)
		os.Setenv("EMAIL_TO", origTo)
		os.Setenv("EMAIL_PORT", origPort)
	}()

	os.Setenv("EMAIL_USER", "user@example.com")
	os.Setenv("EMAIL_PASS", "password123")
	os.Setenv("EMAIL_TO", "recipient@example.com")
	os.Setenv("EMAIL_PORT", "invalid")

	service := newEmailService()

	require.True(t, service.enabled)
	assert.Equal(t, 587, service.port, "should use default port when invalid port is provided")
}

func TestEmailService_SendNotificationEmail_Disabled(t *testing.T) {
	service := &emailService{enabled: false}

	// Should not panic when disabled
	service.sendNotificationEmail("Test Subject", "<p>Test Content</p>")
}

func TestEmailService_NotifyItemAdded(t *testing.T) {
	service := &emailService{enabled: false}

	price := 29.99
	tests := []struct {
		name             string
		itemName         string
		familyMemberName string
		link             string
		price            *float64
	}{
		{
			name:             "with all fields",
			itemName:         "Test Item",
			familyMemberName: "John Doe",
			link:             "https://example.com/item",
			price:            &price,
		},
		{
			name:             "without link",
			itemName:         "Test Item",
			familyMemberName: "John Doe",
			link:             "",
			price:            &price,
		},
		{
			name:             "without price",
			itemName:         "Test Item",
			familyMemberName: "John Doe",
			link:             "https://example.com/item",
			price:            nil,
		},
		{
			name:             "minimal fields",
			itemName:         "Test Item",
			familyMemberName: "John Doe",
			link:             "",
			price:            nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic when disabled
			service.notifyItemAdded(tt.itemName, tt.familyMemberName, tt.link, tt.price)
		})
	}
}

func TestEmailService_NotifyItemReserved(t *testing.T) {
	service := &emailService{enabled: false}

	tests := []struct {
		name             string
		itemName         string
		familyMemberName string
		reserved         bool
	}{
		{
			name:             "reserved",
			itemName:         "Test Item",
			familyMemberName: "John Doe",
			reserved:         true,
		},
		{
			name:             "unreserved",
			itemName:         "Test Item",
			familyMemberName: "John Doe",
			reserved:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic when disabled
			service.notifyItemReserved(tt.itemName, tt.familyMemberName, tt.reserved)
		})
	}
}

func TestEmailService_NotifyItemDeleted(t *testing.T) {
	service := &emailService{enabled: false}

	// Should not panic when disabled
	service.notifyItemDeleted("Test Item", "John Doe")
}

func TestEmailService_NotifyFamilyMemberAdded(t *testing.T) {
	service := &emailService{enabled: false}

	// Should not panic when disabled
	service.notifyFamilyMemberAdded("John Doe")
}

func TestEmailService_NotifyFamilyMemberDeleted(t *testing.T) {
	service := &emailService{enabled: false}

	// Should not panic when disabled
	service.notifyFamilyMemberDeleted("John Doe")
}

func TestCleanEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal email",
			input:    "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "with whitespace",
			input:    "  user@example.com  ",
			expected: "user@example.com",
		},
		{
			name:     "with uppercase",
			input:    "User@Example.COM",
			expected: "user@example.com",
		},
		{
			name:     "with both",
			input:    "  User@Example.COM  ",
			expected: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanEmail(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatLink(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty link",
			input:    "",
			expected: "N/A",
		},
		{
			name:     "short link",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "long link",
			input:    "https://example.com/very/long/path/that/exceeds/fifty/characters/in/total/length",
			expected: "https://example.com/very/long/path/that/exceeds/fi...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLink(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildHTMLEmail(t *testing.T) {
	content := "<p>Test content</p>"
	result := buildHTMLEmail(content)

	assert.Contains(t, result, "<!DOCTYPE html>")
	assert.Contains(t, result, "<html>")
	assert.Contains(t, result, "</html>")
	assert.Contains(t, result, content)
	assert.Contains(t, result, "automated notification")
}
