package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/time/rate"
)

const (
	ADMIN_PASSWORD_ENV = "ADMIN_PASSWORD"
	DEV_MODE_ENV       = "DEV_MODE"
	ITEMS_TABLE        = "items"
	DB_DIR_ENV         = "DATABASE_DIR"
	NULL_STRING        = "NULL"
	NULL_UINT          = -1
)

type Server struct {
	adminPasswordHash string
	jwtSecret         []byte
	limiter           *rate.Limiter
	ipLimiters        map[string]*rate.Limiter
	ipLimitersLock    sync.Mutex
}

func newServer() *Server {
	server := &Server{
		limiter:    rate.NewLimiter(rate.Every(time.Second), 3), // 3 requests per second
		ipLimiters: make(map[string]*rate.Limiter),
	}

	// Generate hashed admin password
	plainPassword := os.Getenv(ADMIN_PASSWORD_ENV)
	if len(plainPassword) == 0 {
		log.Fatalf("missing admin password")
	}

	hashedPassword := sha256.Sum256([]byte(plainPassword))
	server.adminPasswordHash = fmt.Sprintf("%x", hashedPassword)

	// Generate random JWT secret
	server.jwtSecret = make([]byte, 32)
	_, err := rand.Read(server.jwtSecret)
	if err != nil {
		log.Fatal("Failed to generate JWT secret:", err)
	}
	log.Println("JWT Secret generated:", base64.StdEncoding.EncodeToString(server.jwtSecret))

	return server
}

func (s *Server) getIPLimiter(ip string) *rate.Limiter {
	s.ipLimitersLock.Lock()
	defer s.ipLimitersLock.Unlock()

	limiter, exists := s.ipLimiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Every(time.Minute), 10) // 10 requests per minute per IP
		s.ipLimiters[ip] = limiter
	}

	return limiter
}

func getLoginHandler(s *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Global rate limit
		if !s.limiter.Allow() {
			slog.Error("too many login requests", "limit", s.limiter.Limit())
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		// Per-IP rate limit
		ipLimiter := s.getIPLimiter(r.RemoteAddr)
		if !ipLimiter.Allow() {
			slog.Error("too many login requests from ip", "ip", getClientIP(r))
			http.Error(w, "Too many requests from this IP", http.StatusTooManyRequests)
			return
		}

		var loginData struct {
			Password string `json:"password"`
		}

		err := json.NewDecoder(r.Body).Decode(&loginData)
		if err != nil {
			slog.Error("Invalid login request")
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Compare the received hashed password with the stored hashed password
		if s.adminPasswordHash != loginData.Password {
			slog.Error("invalid password", "expected_hash", s.adminPasswordHash, "actual_hash", loginData.Password)
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp": time.Now().Add(time.Hour * 24).Unix(),
		})

		tokenString, err := token.SignedString(s.jwtSecret)
		if err != nil {
			http.Error(w, "Could not generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
		slog.Info("login token correctly generated", "ip", getClientIP(r))
	}

}

func authMiddleware(server *Server, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "No token provided", http.StatusUnauthorized)
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return server.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header first
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// X-Forwarded-For may contain multiple IPs; use the first one
		return strings.Split(ip, ",")[0]
	}

	// If no X-Forwarded-For header, use RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func getItemsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			slog.Info("received get items request", "ip", getClientIP(r))
			items, err := getItems(db)
			if err != nil {
				slog.Error("cannot get the items", "error", err)
				http.Error(w, "cannot get the items", http.StatusInternalServerError)
			}
			w.Header().Set("content-type", "application/json")
			jsonErr := json.NewEncoder(w).Encode(items)
			if jsonErr != nil {
				slog.Error("cannot serialize items to json", "error", jsonErr)
				http.Error(w, "cannot serialize items to json", http.StatusInternalServerError)
			}
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func getReserveItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract the ID from the path
		path := strings.TrimPrefix(r.URL.Path, "/reserve/")
		id := strings.TrimSuffix(path, "/")

		if id == "" {
			http.Error(w, "Missing ID in the path", http.StatusBadRequest)
			return
		}

		count, err := decreaseAmount(db, id)

		if err != nil {
			slog.Error("cannot reserve the item", "id", id, "error", err)
			http.Error(w, "Cannot reserve the item", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "%d", count)
	}
}

func getUpdateItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract the ID from the path
		path := strings.TrimPrefix(r.URL.Path, "/admin/update/")
		id := strings.TrimSuffix(path, "/")

		if id == "" {
			http.Error(w, "Missing ID in the path", http.StatusBadRequest)
			return
		}

		body := NullItem
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			slog.Error("cannot parse the request body", "error", err.Error())
			http.Error(w, "cannot parse the request body", http.StatusBadRequest)
			return
		}

		updateErr := updateItem(db, id, body)
		if updateErr != nil {
			slog.Error("cannot update the item", "error", updateErr.Error(), "id", id)
			http.Error(w, "cannot update the item", http.StatusInternalServerError)
			return
		}
	}
}

func getInsertItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body := Item{Count: 1}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			slog.Error("cannot parse the request body", "error", err.Error())
			http.Error(w, "cannot parse the request body", http.StatusBadRequest)
			return
		}

		insertErr := insertItem(db, body)
		if insertErr != nil {
			slog.Error("cannot insert the item", "error", insertErr.Error(), "item", body)
			http.Error(w, "cannot insert the item", http.StatusInternalServerError)
			return
		}
	}
}

func getDeleteItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract the ID from the path
		path := strings.TrimPrefix(r.URL.Path, "/admin/delete/")
		id := strings.TrimSuffix(path, "/")

		if id == "" {
			http.Error(w, "Missing ID in the path", http.StatusBadRequest)
			return
		}

		err := deleteItem(db, id)
		if err != nil {
			slog.Error("cannot delete the item", "error", err.Error())
			http.Error(w, "cannot delete the item", http.StatusInternalServerError)
			return
		}
	}
}

func main() {
	server := newServer()
	db := initializeDatabase()
	defer db.Close()

	// Serve frontend static files
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", http.StripPrefix("/", fs))

	http.HandleFunc("/items", getItemsHandler(db))
	http.HandleFunc("/reserve/", getReserveItemHandler(db))

	http.HandleFunc("/login", getLoginHandler(server))
	http.HandleFunc("/admin/update/", authMiddleware(server, getUpdateItemHandler(db)))
	http.HandleFunc("/admin/insert", authMiddleware(server, getInsertItemHandler(db)))
	http.HandleFunc("/admin/delete/", authMiddleware(server, getDeleteItemHandler(db)))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type Item struct {
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	Person   string `json:"person"`
	Link     string `json:"link"`
	Price    int32  `json:"price"`
	Count    int32  `json:"count"`
	Category string `json:"category"`
}

var NullItem = Item{
	Id:       NULL_UINT,
	Name:     NULL_STRING,
	Person:   NULL_STRING,
	Link:     NULL_STRING,
	Price:    NULL_UINT,
	Count:    NULL_UINT,
	Category: NULL_STRING,
}

func initializeDatabase() *sql.DB {
	dbPath := getDatabasePath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(1)

	err = createTable(db)
	if err != nil {
		log.Fatal(err)
	}

	if isDevMode() {
		insertDummyData(db)
	}

	slog.Info("database correctly initialized", "path", dbPath)

	return db
}

func getDatabasePath() string {
	path := os.Getenv(DB_DIR_ENV)
	if len(path) == 0 {
		return ":memory:"
	}
	return filepath.Join(path, "items.db")
}

func insertDummyData(db *sql.DB) {
	dummyItems := []Item{
		{
			Name:     "Lunch together at a Biergarten",
			Person:   "Bob",
			Count:    2,
			Category: "Shared Experience",
		},
		{
			Name:     "Shoes",
			Person:   "Bob",
			Link:     "https://www.amazon.de",
			Price:    35,
			Count:    1,
			Category: "Specific Item",
		},
		{
			Name:     "Shirt",
			Person:   "Bob",
			Link:     "https://www.amazon.de",
			Price:    25,
			Count:    3,
			Category: "Specific Item",
		},
		{
			Name:     "Pants",
			Person:   "Alice",
			Link:     "https://www.amazon.de",
			Price:    55,
			Count:    2,
			Category: "Specific Item",
		},
		{
			Name:     "T-Shirts size 116",
			Person:   "Carol",
			Count:    2,
			Category: "Buyer's Choice",
		},
	}

	_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", ITEMS_TABLE))
	if err != nil {
		log.Fatal(err)
	}
	err = createTable(db)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range dummyItems {
		err := insertItem(db, item)
		if err != nil {
			log.Fatal(err)
		}
	}

	slog.Info("dummy data correctly inserted")
}

func insertItem(db *sql.DB, item Item) error {
	insertSql := fmt.Sprintf("INSERT INTO %s (name, person, link, price, count, category) VALUES (?, ?, ?, ?, ?, ?)",
		ITEMS_TABLE)
	_, err := db.Exec(insertSql, item.Name, item.Person, item.Link, item.Price, item.Count, item.Category)
	if err != nil {
		return err
	}
	return nil
}

func deleteItem(db *sql.DB, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", ITEMS_TABLE)
	res, err := db.Exec(query, id)
	if err != nil {
		return err
	}

	numRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if numRows == 0 {
		return fmt.Errorf("couldn't find an item with id %s", id)
	}

	return nil
}

func createTable(db *sql.DB) error {
	createTableSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		person TEXT NOT NULL,
		link TEXT,
		price INTEGER,
		count INTEGER NOT NULL DEFAULT 1,
		category TEXT NOT NULL
	);
	PRAGMA journal_mode=WAL;`, ITEMS_TABLE)

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return err
	}

	return nil
}

func getItems(db *sql.DB) ([]Item, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY price ASC, name ASC", ITEMS_TABLE))
	if err != nil {
		return []Item{}, nil
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		// Scan row data into variables
		item := Item{}
		err := rows.Scan(&item.Id, &item.Name, &item.Person, &item.Link, &item.Price,
			&item.Count, &item.Category)
		if err != nil {
			return []Item{}, err
		}
		items = append(items, item)
	}

	return items, nil
}

func decreaseAmount(db *sql.DB, id string) (int32, error) {
	// Begin a transaction
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() // Rollback the transaction if it's not committed

	// Select the updated value
	selectQuery := fmt.Sprintf("SELECT count FROM %s WHERE id = $1", ITEMS_TABLE)
	var finalValue int32
	err = tx.QueryRow(selectQuery, id).Scan(&finalValue)
	if err != nil {
		return 0, err
	}
	if finalValue == 0 {
		return 0, fmt.Errorf("this item has already been reserved")
	}

	// Update the amount
	updateQuery := fmt.Sprintf("UPDATE %s SET count = count - 1 WHERE id = $1 and count > 0", ITEMS_TABLE)
	result, err := tx.Exec(updateQuery, id)
	if err != nil {
		return 0, err
	}

	// Check how many rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected == 0 {
		return 0, fmt.Errorf("not item with id %s was reserved", id)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return finalValue - 1, nil
}

func updateItem(db *sql.DB, id string, item Item) error {
	// Begin a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback the transaction if it's not committed

	// Select the item
	selectQuery := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", ITEMS_TABLE)
	var oldItem Item
	err = tx.QueryRow(selectQuery, id).Scan(&oldItem.Id, &oldItem.Name, &oldItem.Person, &oldItem.Link,
		&oldItem.Price, &oldItem.Count, &oldItem.Category)
	if err != nil {
		return err
	}

	if item.Name != NULL_STRING {
		oldItem.Name = item.Name
	}

	if item.Person != NULL_STRING {
		oldItem.Person = item.Person
	}

	if item.Link != NULL_STRING {
		oldItem.Link = item.Link
	}

	if item.Price != NULL_UINT {
		oldItem.Price = item.Price
	}

	if item.Count != NULL_UINT {
		oldItem.Count = item.Count
	}

	if item.Category != NULL_STRING {
		oldItem.Category = item.Category
	}

	// Update the amount
	updateQuery := fmt.Sprintf(
		"UPDATE %s SET name = ?, person = ?, link = ?, price = ?, count = ?, category = ? WHERE id = ?",
		ITEMS_TABLE)
	result, err := tx.Exec(updateQuery, oldItem.Name, oldItem.Person, oldItem.Link, oldItem.Price, oldItem.Count,
		oldItem.Category, oldItem.Id)
	if err != nil {
		return err
	}

	// Check how many rows were affected
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func isDevMode() bool {
	env := os.Getenv(DEV_MODE_ENV)
	return env == "1"
}
