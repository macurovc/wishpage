# Wishpage

A simple wishlist management application built with Go, HTMX, and SQLite. Features server-side rendering, minimal JavaScript, and a clean responsive UI.

## Features

- 📝 Create and manage wishlist items for family members
- 👥 Add and organize family members
- ✅ Mark items as reserved/available
- 🔐 Password-protected editing mode
- 📧 Email notifications for all changes
- 🎨 Clean, responsive UI with HTMX for dynamic updates
- 🔒 Session-based authentication
- 📦 Uses templ for type-safe Go templates

## Quick Start

### Prerequisites

- Go 1.25 or higher
- SQLite3 (required for building - the `go-sqlite3` driver uses CGO)
- `golangci-lint` (optional, for development - see Development Tools section)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd wishpage
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
# Required for edit mode
export EDIT_PASSWORD="your-secure-password"

# Optional configuration
export DATABASE_PATH="/path/to/wishlist.db"
export PORT="3002"

# Optional: Email notifications (see Email Notifications section)
export EMAIL_USER="your-email@gmail.com"
export EMAIL_PASS="your-app-password"
export EMAIL_TO="recipient@example.com"
```

4. Run the application:
```bash
go run .
```

5. Open your browser to `http://localhost:3002`

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `3002` |
| `DATABASE_PATH` | Path to SQLite database | System temp directory (`/tmp/wishlist.db` on Unix-like systems) |
| `EDIT_PASSWORD` | Password for edit mode | _(required for editing)_ |
| `RESET_DB` | Reset database on startup (use with caution!) | `0` |
| `FORCE_SECURE_COOKIES` | Always use secure cookies (for testing) | Auto-detected based on request protocol |
| `ALLOW_INSECURE_COOKIES` | Always use insecure cookies (for local dev) | Auto-detected based on request protocol |
| `EMAIL_HOST` | SMTP server hostname | `smtp.gmail.com` |
| `EMAIL_PORT` | SMTP server port | `587` |
| `EMAIL_USER` | SMTP username | _(required for email)_ |
| `EMAIL_PASS` | SMTP password | _(required for email)_ |
| `EMAIL_FROM` | Sender email address | Same as `EMAIL_USER` |
| `EMAIL_TO` | Recipient email address | _(required for email)_ |

### Email Notifications

The application sends automatic email notifications for all wishlist changes:
- Items added, reserved, or deleted
- Family members added or deleted

**Important**: Email is completely optional. If not configured, the application works normally without sending notifications.

#### Quick Setup (Gmail)

1. Enable 2-Factor Authentication on your Google account
2. Generate an App Password at https://myaccount.google.com/apppasswords
3. Set environment variables:

```bash
export EMAIL_USER=your-email@gmail.com
export EMAIL_PASS=your-16-char-app-password
export EMAIL_TO=recipient@example.com
```

#### Other Email Providers

**Office 365 / Outlook:**
```bash
export EMAIL_HOST=smtp.office365.com
export EMAIL_PORT=587
export EMAIL_USER=your-email@outlook.com
export EMAIL_PASS=your-password
export EMAIL_TO=recipient@example.com
```

**Custom SMTP Server:**
```bash
export EMAIL_HOST=smtp.example.com
export EMAIL_PORT=587  # or 465 for TLS
export EMAIL_USER=your-username
export EMAIL_PASS=your-password
export EMAIL_TO=recipient@example.com
```

#### Email Troubleshooting

**"Email service verification failed"**
- Check SMTP hostname and port are correct
- Verify firewall allows outbound SMTP connections
- Corporate networks may block SMTP ports

**"Authentication failed"**
- Gmail: Use App Password, not your regular password
- Verify username and password are correct
- Check email provider security settings

**Emails not received**
- Check spam/junk folder
- Verify `EMAIL_TO` address is correct
- Check server logs for errors
- Some providers have rate limits

## Development

### Development Tools

The project includes a Makefile with convenient commands for development:

```bash
# Validate everything (format, lint, test, tidy) - run this before committing!
make validate

# Show all available commands
make help

# Individual commands:
make fmt         # Format all Go code with go fmt
make lint        # Run golangci-lint with comprehensive checks
make test        # Run all tests with race detection and coverage
make tidy        # Clean up go.mod and verify dependencies
make clean       # Remove build artifacts and caches
```

**Prerequisites for development tools:**
- `golangci-lint` is required for `make lint` and `make validate`
- Install on macOS: `brew install golangci-lint`
- Install on Linux: `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin`
- See [installation guide](https://golangci-lint.run/usage/install/) for other platforms

**Recommended workflow:**
- Run `make validate` before committing changes
- This ensures code is formatted, passes linting, and all tests pass
- The linter checks 33 different rules for code quality, security, and style

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestEmailService

# Run tests with coverage (or use 'make test')
go test -v -race -coverprofile=coverage.out ./...
```

### Generating Templates

This project uses [templ](https://templ.guide/) for type-safe HTML templates. After modifying `.templ` files, regenerate the Go code:

```bash
# Install templ (if not already installed)
go install github.com/a-h/templ/cmd/templ@latest

# Generate template code
templ generate
```

**Note**: The generated `*_templ.go` files are checked into version control. Always regenerate them after editing `.templ` files and include both the `.templ` and `*_templ.go` files in your commits.

## Project Structure

```
wishpage/
├── main.go                  # Application entry point and server setup
├── auth.go                  # Authentication and session management
├── database.go              # Database initialization
├── handlers.go              # Main page handlers
├── items.go                 # Item CRUD operations
├── family_members.go        # Family member CRUD operations
├── email_service.go         # Email notification service
├── helpers.go               # Utility functions
├── models/
│   └── models.go           # Data models
├── templates/
│   ├── layout.templ        # Base layout template
│   ├── view.templ          # Public view templates
│   ├── edit.templ          # Edit mode templates
│   ├── login.templ         # Login page template
│   └── error.templ         # Error template
├── static/
│   └── css/
│       └── styles.css      # Application styles
├── Makefile                 # Development commands (validate, test, lint, fmt)
├── .golangci.yml            # Linter configuration (33 enabled checks)
├── go.mod                   # Go module dependencies
└── *_test.go               # Test files
```

## API Endpoints

### Public Endpoints (No Authentication Required)

**Page Routes:**
- `GET /` - Landing page (shows all family members, no items)
- `GET /family/{name}` - View items for a specific family member (e.g., `/family/Alice`)
- `GET /login` - Login page
- `GET /static/*` - Static files (CSS, JS, images)

**API Routes:**
- `GET /api/items` - Get all items (supports `?family_member_id=X` filter)
- `GET /api/items/{id}/reserve-confirm` - Show reserve confirmation buttons (HTMX)
- `GET /api/items/{id}/reserve-cancel` - Show reserve button again (HTMX)
- `PUT /api/items/{id}/reserve` - Mark item as reserved
- `GET /api/family_members` - Get all family members

### Protected Endpoints (Require Authentication)

**Authentication:**
- `POST /api/login` - Login with password, returns session cookie
- `POST /api/logout` - Logout and clear session

**Page Routes:**
- `GET /edit` - Edit mode page (full list of items and family members)

**Item Management:**
- `POST /api/items` - Create new item (requires name, family_member_id; optional link, price)
- `PUT /api/items/{id}` - Update item details
- `DELETE /api/items/{id}` - Delete item
- `PUT /api/items/{id}/unreserve` - Mark item as unreserved (admin only)

**Family Member Management:**
- `POST /api/family_members` - Create family member
- `PUT /api/family_members/{id}` - Update family member name
- `DELETE /api/family_members/{id}` - Delete family member (cascades to items)

## Features in Detail

### HTMX Integration

The application uses HTMX for dynamic updates without full page reloads:
- Item reservation toggles update in-place
- Adding/deleting items updates the list dynamically
- Family member management happens without page navigation
- Smooth transitions and feedback

### Session Management

- Secure session tokens with automatic expiration (24 hours)
- Cookie-based authentication (HttpOnly, SameSite=Strict)
- Background cleanup of expired sessions (runs hourly)
- Concurrent access handling with thread-safe session store
- Constant-time password comparison to prevent timing attacks

### Database

- SQLite for simplicity and portability
- Foreign key constraints for data integrity
- Cascade deletes for family members (deleting a member removes all their items)
- Automatic schema creation on first run
- Items automatically ordered by price (items without price shown first)

### Email Notifications

When configured, the application sends HTML-formatted email notifications for all changes:
- **Items**: added, reserved/unreserved, deleted (includes name, family member, price, link)
- **Family Members**: added or deleted

The email service supports:
- Both TLS (port 465) and STARTTLS (port 587) connections
- Gmail, Office 365, and custom SMTP servers
- Graceful degradation (app continues if email fails)
- Automatic HTML escaping for security

## Security Considerations

- Password-protected edit mode with constant-time comparison
- Session-based authentication with automatic expiration (24 hours)
- **Smart cookie security** - automatically detects HTTPS and sets Secure flag accordingly
- HttpOnly cookies prevent XSS attacks on session tokens
- SameSite=Strict cookies prevent CSRF attacks
- CSRF protection through session validation
- Input validation and sanitization
- SQL injection prevention through parameterized queries
- Email credentials stored in environment variables (never committed)
- HTML escaping in email notifications prevents injection attacks

### Cookie Security Behavior

The application automatically detects whether requests come via HTTPS and sets the `Secure` cookie flag accordingly:

- ✅ **Direct HTTPS**: Secure cookies enabled
- ✅ **Behind reverse proxy** (Cloudflare Tunnel, nginx, Caddy): Detects `X-Forwarded-Proto: https` header
- ✅ **Local HTTP access**: Secure cookies disabled (allows LAN access)
- ✅ **Mixed environment**: Each request is evaluated independently

**No configuration needed** for typical deployments! The app works seamlessly for:
- Local LAN access over HTTP (`http://192.168.1.100:3002`)
- Internet access via Cloudflare Tunnel (`https://wishlist.yourdomain.com`)
- Traditional HTTPS reverse proxies (nginx, Caddy, Traefik)

**Override options** (rarely needed):
- `FORCE_SECURE_COOKIES=true` - Always use secure cookies
- `ALLOW_INSECURE_COOKIES=true` - Never use secure cookies

## Deployment

This application uses Go's `embed` directive to include all static files (CSS, JS, images) directly in the binary. This means you only need to deploy a **single executable file** - no separate static directory required!

### Using Docker

1. **Validate the code before building (recommended):**

   ```bash
   make validate
   ```

2. **Build the Docker image:**

   ```bash
   docker build -t wishpage .
   ```

3. **Run the Docker container:**

   ```bash
   docker run -d \
     -p 3002:3002 \
     -e EDIT_PASSWORD=your_super_secret_password \
     -e DATABASE_PATH=/data/wishlist.db \
     -v $(pwd)/data:/data \
     --name wishpage \
     wishpage
   ```

   - The `-e` flag sets environment variables
   - The `-v` flag mounts a local directory (`./data`) into the container at `/data` for database persistence
   - Without the volume mount, your data will be lost if the container is removed

4. **Access the application:**

   The application will be accessible at `http://localhost:3002`

### Manual Deployment

1. **Validate the code before building:**
```bash
make validate
```

This ensures your code is properly formatted, passes all linter checks, and all tests pass.

2. **Build the application:**
```bash
go build -o wishpage
```

This creates a single, self-contained binary with all static files embedded.

3. **Copy the binary to your server:**
```bash
scp wishpage your-server:/opt/wishpage/
```

4. **Set environment variables on your server:**
```bash
export EDIT_PASSWORD="your-secure-password"
export DATABASE_PATH="/var/lib/wishpage/wishlist.db"
# Optional: Email configuration
export EMAIL_USER="your-email@gmail.com"
export EMAIL_PASS="your-app-password"
export EMAIL_TO="recipient@example.com"
```

5. **Run the binary:**
```bash
./wishpage
```

**That's it!** No need to copy the `static/` directory - it's already in the binary.

**Additional recommendations:**
- Consider using a process manager like systemd or supervisor (see example below)
- Put behind a reverse proxy (nginx/Caddy) for HTTPS

### Example systemd Service

Create `/etc/systemd/system/wishpage.service`:

```ini
[Unit]
Description=Wishpage Application
After=network.target

[Service]
Type=simple
User=wishpage
WorkingDirectory=/opt/wishpage
ExecStart=/opt/wishpage/wishpage
Environment="EDIT_PASSWORD=your-password"
Environment="DATABASE_PATH=/var/lib/wishpage/wishlist.db"
Environment="PORT=3002"
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Then:
```bash
sudo systemctl daemon-reload
sudo systemctl enable wishpage
sudo systemctl start wishpage
```

## Troubleshooting

### Database Issues

- **Default location**: If `DATABASE_PATH` is not set, the database is created in your system's temp directory (e.g., `/tmp/wishlist.db` on Unix-like systems)
- Ensure the database file path is writable
- Check file permissions (the application needs read/write access)
- Verify SQLite3 is installed (required for the go-sqlite3 driver)
- If you need to reset the database, set `RESET_DB=1` environment variable (⚠️ this will delete all data!)

### Email Not Working

- Verify all required variables are set: `EMAIL_USER`, `EMAIL_PASS`, `EMAIL_TO`
- Check server logs for detailed error messages
- Look for "Email service configured correctly" message on startup
- Test with a simple action (add an item) and check spam folder
- Ensure firewall allows outbound SMTP connections

### Authentication Problems

- Verify `EDIT_PASSWORD` is set
- Clear browser cookies and try again
- Check server logs for session-related errors

## Contributing

Contributions are welcome! Please ensure:
- All validation checks pass: `make validate`
- Templates are regenerated: `templ generate` (if you modified `.templ` files)

The `make validate` command will automatically:
- Format your code with `go fmt`
- Run comprehensive linting checks
- Run all tests with race detection
- Verify dependencies are clean

## License

See the LICENSE file in the root of the repository.

