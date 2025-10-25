package main

import (
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed static
var staticFiles embed.FS

type server struct {
	db           *sql.DB
	mux          *http.ServeMux
	sessions     *sessionStore
	emailService *emailService
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting wishlist server...")

	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database initialized successfully")

	sessions := newSessionStore()
	emailService := newEmailService()

	// Start session cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			sessions.cleanup()
			log.Println("Session cleanup completed")
		}
	}()

	s := &server{
		db:           db,
		mux:          http.NewServeMux(),
		sessions:     sessions,
		emailService: emailService,
	}

	s.routes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
	}

	// Create HTTP server with timeouts for security
	srv := &http.Server{
		Addr:           ":" + port,
		Handler:        s.mux,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	log.Printf("Server listening on http://localhost:%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *server) routes() {
	// Serve embedded static files (CSS, JS, images, etc.)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to access embedded static files: %v", err)
	}
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Application routes - Public view
	s.mux.HandleFunc("/", s.handleIndex)

	// Auth routes
	s.mux.HandleFunc("/login", s.handleLoginPage)
	s.mux.HandleFunc("/api/login", s.handleLogin)
	s.mux.HandleFunc("/api/logout", s.handleLogout)

	// Application routes - Authenticated edit
	s.mux.HandleFunc("/edit", s.requireAuth(s.handleEdit))

	// API routes
	s.mux.HandleFunc("/api/items", s.handleItems)
	s.mux.HandleFunc("/api/items/", s.handleItemsID)
	s.mux.HandleFunc("/api/family_members", s.handleFamilyMembers)
	s.mux.HandleFunc("/api/family_members/", s.handleFamilyMembersID)
}
