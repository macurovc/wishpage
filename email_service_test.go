package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailService_Disabled(t *testing.T) {
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
			t.Setenv("EMAIL_USER", tt.emailUser)
			t.Setenv("EMAIL_PASS", tt.emailPass)
			t.Setenv("EMAIL_TO", tt.emailTo)

			service := newEmailService()
			assert.False(t, service.enabled, tt.description)
		})
	}
}

func TestNewEmailService_Enabled(t *testing.T) {
	t.Setenv("EMAIL_USER", "user@example.com")
	t.Setenv("EMAIL_PASS", "password123")
	t.Setenv("EMAIL_TO", "recipient@example.com")
	t.Setenv("EMAIL_HOST", "smtp.example.com")
	t.Setenv("EMAIL_PORT", "25")
	t.Setenv("EMAIL_FROM", "sender@example.com")

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
	// Set only required fields, unset optional ones
	t.Setenv("EMAIL_USER", "user@example.com")
	t.Setenv("EMAIL_PASS", "password123")
	t.Setenv("EMAIL_TO", "recipient@example.com")
	// EMAIL_HOST, EMAIL_PORT, and EMAIL_FROM are unset by default

	service := newEmailService()

	require.True(t, service.enabled, "service should be enabled")
	assert.Equal(t, "smtp.gmail.com", service.host, "should use default host")
	assert.Equal(t, 587, service.port, "should use default port")
	assert.Equal(t, "user@example.com", service.from, "should use EMAIL_USER as default for EMAIL_FROM")
}

func TestNewEmailService_InvalidPort(t *testing.T) {
	t.Setenv("EMAIL_USER", "user@example.com")
	t.Setenv("EMAIL_PASS", "password123")
	t.Setenv("EMAIL_TO", "recipient@example.com")
	t.Setenv("EMAIL_PORT", "invalid")

	service := newEmailService()

	require.True(t, service.enabled)
	assert.Equal(t, 587, service.port, "should use default port when invalid port is provided")
}

func TestEmailService_SendNotificationEmail_Disabled(_ *testing.T) {
	service := &emailService{enabled: false}

	// Should not panic when disabled
	service.sendNotificationEmail("Test Subject", "<p>Test Content</p>")
}

func TestEmailService_NotifyItemAdded(t *testing.T) {
	service := &emailService{enabled: false}

	price := 29.99
	tests := []struct {
		itemName         string
		familyMemberName string
		link             string
		name             string
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
		t.Run(tt.name, func(_ *testing.T) {
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
		t.Run(tt.name, func(_ *testing.T) {
			// Should not panic when disabled
			service.notifyItemReserved(tt.itemName, tt.familyMemberName, tt.reserved)
		})
	}
}

func TestEmailService_NotifyItemDeleted(_ *testing.T) {
	service := &emailService{enabled: false}

	// Should not panic when disabled
	service.notifyItemDeleted("Test Item", "John Doe")
}

func TestEmailService_NotifyFamilyMemberAdded(_ *testing.T) {
	service := &emailService{enabled: false}

	// Should not panic when disabled
	service.notifyFamilyMemberAdded("John Doe")
}

func TestEmailService_NotifyFamilyMemberDeleted(_ *testing.T) {
	service := &emailService{enabled: false}

	// Should not panic when disabled
	service.notifyFamilyMemberDeleted("John Doe")
}
