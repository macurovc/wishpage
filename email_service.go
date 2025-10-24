package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

// emailService handles sending email notifications.
type emailService struct {
	host     string
	from     string
	to       string
	user     string
	password string
	port     int
	enabled  bool
	// skipTLSVerify allows disabling TLS verification for testing
	// #nosec G402 -- This is configurable and defaults to false (secure)
	skipTLSVerify bool
}

// newEmailService creates and initializes an email service from environment variables.
func newEmailService() *emailService {
	user := os.Getenv("EMAIL_USER")
	password := os.Getenv("EMAIL_PASS")
	host := os.Getenv("EMAIL_HOST")
	portStr := os.Getenv("EMAIL_PORT")
	to := os.Getenv("EMAIL_TO")
	from := os.Getenv("EMAIL_FROM")

	// If credentials are not configured, disable email service
	if user == "" || password == "" {
		log.Println("Email service not configured: EMAIL_USER or EMAIL_PASS is missing. Email notifications will be disabled.")
		return &emailService{enabled: false}
	}

	if to == "" {
		log.Println("Email service not configured: EMAIL_TO is missing. Email notifications will be disabled.")
		return &emailService{enabled: false}
	}

	// Default values
	if host == "" {
		host = "smtp.gmail.com"
	}
	port := 587
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		} else {
			log.Printf("Invalid EMAIL_PORT value '%s', using default 587", portStr)
		}
	}
	if from == "" {
		from = user
	}

	service := &emailService{
		enabled:       true,
		host:          host,
		port:          port,
		user:          user,
		password:      password,
		from:          from,
		to:            to,
		skipTLSVerify: false,
	}

	log.Printf("Email service configured: { host: '%s', port: %d, from: '%s', to: '%s' }", host, port, from, to)

	// Verify connection on startup
	if err := service.verify(); err != nil {
		log.Printf("Email service verification failed: %v", err)
	} else {
		log.Println("Email service is configured correctly and ready to send messages.")
	}

	return service
}

// verify checks if the email service can connect to the SMTP server.
func (e *emailService) verify() error {
	if !e.enabled {
		return fmt.Errorf("email service is not enabled")
	}

	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing SMTP client: %v", err)
		}
	}()

	// Start TLS if port is 587
	if e.port == 587 {
		// #nosec G402 -- TLS verification is enabled by default; skipTLSVerify is only for testing
		tlsConfig := &tls.Config{
			ServerName:         e.host,
			InsecureSkipVerify: e.skipTLSVerify,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	return nil
}

// sendNotificationEmail sends an email with the given subject and HTML content.
// This should typically be called with 'go' to avoid blocking the HTTP response.
func (e *emailService) sendNotificationEmail(subject, htmlContent string) {
	if !e.enabled {
		log.Println("Email service is not enabled. Skipping email notification.")
		return
	}

	// Build email message
	from := fmt.Sprintf("From: %s\r\n", e.from)
	to := fmt.Sprintf("To: %s\r\n", e.to)
	subj := fmt.Sprintf("Subject: Wishlist Notification: %s\r\n", subject)
	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"

	message := []byte(from + to + subj + mime + htmlContent)

	// Setup authentication
	auth := smtp.PlainAuth("", e.user, e.password, e.host)

	// Connect and send email
	addr := fmt.Sprintf("%s:%d", e.host, e.port)

	// Handle different port configurations
	var err error
	if e.port == 465 {
		// Use TLS for port 465
		err = e.sendMailTLS(addr, auth, e.from, []string{e.to}, message)
	} else {
		// Use STARTTLS for port 587 or other ports
		err = smtp.SendMail(addr, auth, e.from, []string{e.to}, message)
	}

	if err != nil {
		log.Printf("Error sending notification email: %v", err)
		return
	}

	log.Printf("Notification email sent successfully: %s", subject)
}

// sendMailTLS sends email using TLS connection (for port 465).
func (e *emailService) sendMailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// #nosec G402 -- TLS verification is enabled by default; skipTLSVerify is only for testing
	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName:         e.host,
		InsecureSkipVerify: e.skipTLSVerify,
	}

	// Connect to server with TLS using context-aware dialing
	conn, err := e.dialTLSWithContext(addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to dial with TLS: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("Error closing TLS connection: %v", closeErr)
		}
	}()

	// Create SMTP client
	client, err := smtp.NewClient(conn, e.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Error closing SMTP client: %v", closeErr)
		}
	}()

	// Authenticate
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	// Send message body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			log.Printf("Error closing data writer: %v", closeErr)
		}
	}()

	if _, err = writer.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// dialTLSWithContext establishes a TLS connection using context-aware dialing.
func (e *emailService) dialTLSWithContext(addr string, tlsConfig *tls.Config) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("Error closing connection after handshake failure: %v", closeErr)
		}
		return nil, err
	}

	return tlsConn, nil
}

// Helper functions for generating email content

func (e *emailService) notifyItemAdded(itemName, familyMemberName, link string, price *float64) {
	priceStr := "N/A"
	if price != nil {
		priceStr = fmt.Sprintf("$%.2f", *price)
	}

	linkStr := "N/A"
	if link != "" {
		linkStr = fmt.Sprintf(`<a href=%q>%s</a>`, template.HTMLEscapeString(link), template.HTMLEscapeString(link))
	}

	htmlContent := fmt.Sprintf(`
		<p>A new item has been added to the wishlist:</p>
		<ul>
			<li><strong>Item:</strong> %s</li>
			<li><strong>For:</strong> %s</li>
			<li><strong>Price:</strong> %s</li>
			<li><strong>Link:</strong> %s</li>
		</ul>
	`, template.HTMLEscapeString(itemName), template.HTMLEscapeString(familyMemberName), priceStr, linkStr)

	go e.sendNotificationEmail("New Item Added", htmlContent)
}

func (e *emailService) notifyItemReserved(itemName, familyMemberName string, reserved bool) {
	status := "Available"
	action := "Un-reserved"
	if reserved {
		status = "Reserved"
		action = "Reserved"
	}

	htmlContent := fmt.Sprintf(`
		<p>An item's reservation status has been updated:</p>
		<ul>
			<li><strong>Item:</strong> %s</li>
			<li><strong>For:</strong> %s</li>
			<li><strong>Status:</strong> %s</li>
		</ul>
	`, template.HTMLEscapeString(itemName), template.HTMLEscapeString(familyMemberName), status)

	go e.sendNotificationEmail(fmt.Sprintf("Item %s", action), htmlContent)
}

func (e *emailService) notifyItemDeleted(itemName, familyMemberName string) {
	htmlContent := fmt.Sprintf(`
		<p>An item has been deleted from the wishlist:</p>
		<ul>
			<li><strong>Item:</strong> %s</li>
			<li><strong>For:</strong> %s</li>
		</ul>
	`, template.HTMLEscapeString(itemName), template.HTMLEscapeString(familyMemberName))

	go e.sendNotificationEmail("Item Deleted", htmlContent)
}

func (e *emailService) notifyFamilyMemberAdded(name string) {
	htmlContent := fmt.Sprintf(
		`<p>A new family member has been added: <strong>%s</strong></p>`,
		template.HTMLEscapeString(name),
	)
	go e.sendNotificationEmail("New Family Member Added", htmlContent)
}

func (e *emailService) notifyFamilyMemberDeleted(name string) {
	htmlContent := fmt.Sprintf(
		`<p>A family member has been deleted: <strong>%s</strong></p>`,
		template.HTMLEscapeString(name),
	)
	go e.sendNotificationEmail("Family Member Deleted", htmlContent)
}
