# --- Build Stage ---
FROM golang:1.26.5-alpine AS builder

# Install build dependencies
# gcc and musl-dev are required for CGO_ENABLED=1 (SQLite)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=1 is required for mattn/go-sqlite3 which uses C bindings
# -ldflags="-w -s" strips debug info to reduce binary size
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o wishpage .

# --- Runtime Stage ---
FROM alpine:latest

# Install runtime TLS certificates. SQLite is compiled into the application binary.
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/wishpage .

# Create data directory for database
RUN mkdir -p /data

# Expose application port
EXPOSE 3002

# Set default environment variables
ENV PORT=3002
ENV DATABASE_PATH=/data/wishlist.db

# Run the application
CMD ["./wishpage"]
