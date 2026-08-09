package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"wishpage/models"
	"wishpage/templates"
)

var errInvalidItemLink = errors.New("link must be a valid HTTP or HTTPS URL")

func (s *server) handleItems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		familyMemberID := r.URL.Query().Get("family_member_id")
		s.renderItemList(w, r, familyMemberID)
	case http.MethodPost:
		s.requireAuth(s.createItem)(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := templates.Error("Method not allowed").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
	}
}

func (s *server) handleItemsID(w http.ResponseWriter, r *http.Request) {
	// Handle special reservation endpoints
	if handled := s.handleReservationEndpoints(w, r); handled {
		return
	}

	// Handle regular item operations
	switch r.Method {
	case http.MethodPut:
		s.requireAuth(s.updateItem)(w, r)
	case http.MethodDelete:
		s.requireAuth(s.deleteItem)(w, r)
	default:
		s.renderMethodNotAllowed(w, r)
	}
}

// handleReservationEndpoints routes reservation-related endpoints and returns true if handled.
func (s *server) handleReservationEndpoints(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path

	// No longer need /reserve-confirm and /reserve-cancel endpoints - using CSS now!
	if strings.HasSuffix(path, "/reserve") {
		s.handleMethodOrError(w, r, http.MethodPut, s.reserveItem)
		return true
	}
	if strings.HasSuffix(path, "/unreserve") {
		handler := s.requireAuth(s.unreserveItem)
		s.handleMethodOrError(w, r, http.MethodPut, handler)
		return true
	}

	return false
}

// handleMethodOrError calls handler if method matches, otherwise renders method not allowed error.
func (s *server) handleMethodOrError(w http.ResponseWriter, r *http.Request, expectedMethod string, handler http.HandlerFunc) {
	if r.Method == expectedMethod {
		handler(w, r)
	} else {
		s.renderMethodNotAllowed(w, r)
	}
}

// renderMethodNotAllowed renders a "Method not allowed" error response.
func (s *server) renderMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	if err := templates.Error("Method not allowed").Render(r.Context(), w); err != nil {
		log.Printf("Error rendering template: %v", err)
	}
}

func (s *server) getAllItems(ctx context.Context, familyMemberID string) ([]models.Item, error) {
	query := `
        SELECT items.id, items.family_member_id, items.name, items.link, items.price, items.reserved, family_members.name as family_member_name 
        FROM items 
        JOIN family_members ON items.family_member_id = family_members.id
    `
	args := []any{}
	if familyMemberID != "" {
		query += " WHERE family_member_id = ?"
		args = append(args, familyMemberID)
	}
	query += " ORDER BY CASE WHEN items.price IS NULL THEN 0 ELSE 1 END, items.price ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query items for family member %s: %w", familyMemberID, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	items := []models.Item{}
	for rows.Next() {
		item, err := scanItemRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// scanItemRow scans a database row into an Item struct, handling nullable fields.
func scanItemRow(scanner interface {
	Scan(dest ...any) error
}) (models.Item, error) {
	var item models.Item
	var link sql.NullString
	var price sql.NullFloat64

	err := scanner.Scan(&item.ID, &item.FamilyMemberID, &item.Name, &link, &price, &item.Reserved, &item.FamilyMemberName)
	if err != nil {
		return models.Item{}, err
	}

	if link.Valid {
		v := link.String
		item.Link = &v
	}
	if price.Valid {
		v := price.Float64
		item.Price = &v
	}

	return item, nil
}

// renderItemList is a helper that fetches items and family members, then renders the appropriate ItemList template.
// It renders ItemListEdit if the user is authenticated, otherwise ItemListView.
func (s *server) renderItemList(w http.ResponseWriter, r *http.Request, familyMemberID string) {
	items, err := s.getAllItems(r.Context(), familyMemberID)
	if err != nil {
		log.Printf("Error fetching items for family member %s: %v", familyMemberID, err)
		w.WriteHeader(http.StatusInternalServerError)
		if renderErr := templates.Error("Failed to fetch items").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	members, err := s.getAllFamilyMembers(r.Context())
	if err != nil {
		log.Printf("Error fetching family members: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if renderErr := templates.Error("Failed to fetch family members").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	// Check if user is authenticated
	cookie, err := r.Cookie("session_token")
	isAuthenticated := err == nil && s.sessions.isValid(cookie.Value)

	if isAuthenticated {
		// In edit mode, render items for the member using the new template
		if err := templates.ItemsForMember(items).Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
	} else {
		if err := templates.ItemListView(items, members).Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
	}
}

// renderSingleItemCard renders just a single item card (for HTMX updates after reservation).
func (s *server) renderSingleItemCard(w http.ResponseWriter, r *http.Request, itemID int) {
	// Fetch the single item
	item, err := s.getItemByID(r.Context(), itemID)
	if err != nil {
		log.Printf("Error fetching item %d: %v", itemID, err)
		w.WriteHeader(http.StatusInternalServerError)
		if renderErr := templates.Error("Failed to fetch item").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	// Get all family members (needed for the template)
	members, err := s.getAllFamilyMembers(r.Context())
	if err != nil {
		log.Printf("Error fetching family members: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if renderErr := templates.Error("Failed to fetch family members").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	// Render just the single card
	if err := templates.ItemCardView(item, members).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering template: %v", err)
	}
}

func (s *server) createItem(w http.ResponseWriter, r *http.Request) {
	data, err := parseItemRequestBody(r)
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	validated, err := s.validateCreateItemData(r, data)
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.insertItem(r.Context(), validated)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "Failed to add item")
		return
	}

	s.notifyItemCreated(r.Context(), result)
	s.renderItemList(w, r, validated.FamilyMemberIDStr)
}

// renderError is a helper to render an error template with a status code.
func (s *server) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	w.WriteHeader(status)
	if err := templates.Error(message).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering template: %v", err)
	}
}

// validatedItemData holds validated data for creating an item.
type validatedItemData struct {
	Name              string
	FamilyMemberID    int
	FamilyMemberIDStr string
	Price             float64
	PriceValid        bool
	Link              string
}

// validateCreateItemData validates and normalizes all item creation data.
func (s *server) validateCreateItemData(r *http.Request, data *requestBodyData) (*validatedItemData, error) {
	name, err := validateAndNormalizeName(data.Name)
	if err != nil {
		return nil, err
	}
	if len(name) > 200 {
		return nil, fmt.Errorf("item name too long (max 200 characters)")
	}

	familyMemberID, familyMemberIDStr, err := s.resolveFamilyMemberID(r, data.FamilyMemberID)
	if err != nil {
		return nil, err
	}

	price, err := parsePrice(data.Price)
	if err != nil {
		return nil, err
	}
	link, err := validateItemLink(data.Link)
	if err != nil {
		return nil, err
	}

	exists, err := s.itemNameExists(r.Context(), name, familyMemberID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to check item name: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("an item with this name already exists for this family member")
	}

	return &validatedItemData{
		Name:              name,
		FamilyMemberID:    familyMemberID,
		FamilyMemberIDStr: familyMemberIDStr,
		Price:             price,
		PriceValid:        data.Price != "",
		Link:              link,
	}, nil
}

// insertItem inserts a validated item into the database.
func (s *server) insertItem(ctx context.Context, data *validatedItemData) (sql.Result, error) {
	linkParam := toNullString(data.Link)
	priceParam := sql.NullFloat64{Float64: data.Price, Valid: data.PriceValid}

	return s.db.ExecContext(ctx,
		"INSERT INTO items (family_member_id, name, link, price) VALUES (?, ?, ?, ?)",
		data.FamilyMemberID, data.Name, linkParam, priceParam)
}

// validateAndNormalizeName ensures the item name is valid, returning an error if empty.
func validateAndNormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("item name is required")
	}
	return name, nil
}

// itemNameExists checks if an item with the given name already exists for the family member.
// The excludeItemID parameter allows excluding a specific item (useful for updates).
func (s *server) itemNameExists(ctx context.Context, name string, familyMemberID, excludeItemID int) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM items WHERE family_member_id = ? AND name = ? AND id != ?"
	err := s.db.QueryRowContext(ctx, query, familyMemberID, name, excludeItemID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check item name uniqueness: %w", err)
	}
	return count > 0, nil
}

// resolveFamilyMemberID determines the family member ID from body, header, or defaults to first available.
func (s *server) resolveFamilyMemberID(r *http.Request, bodyID string) (id int, idStr string, err error) {
	// Try body ID first
	if bodyID != "" {
		return s.parseAndValidateFamilyMemberID(bodyID)
	}

	// Try header
	headerID := r.Header.Get("X-Family-Member-ID")
	if headerID != "" {
		return s.parseAndValidateFamilyMemberID(headerID)
	}

	// Default to first family member
	members, err := s.getAllFamilyMembers(r.Context())
	if err != nil || len(members) == 0 {
		return 0, "", fmt.Errorf("missing family member selection")
	}

	firstMemberIDStr := strconv.Itoa(members[0].ID)
	return members[0].ID, firstMemberIDStr, nil
}

// parseAndValidateFamilyMemberID converts and validates a family member ID string.
func (s *server) parseAndValidateFamilyMemberID(idStr string) (id int, idString string, err error) {
	id, err = strconv.Atoi(idStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid family member ID")
	}
	return id, idStr, nil
}

// parsePrice validates and parses the price string, returning 0.0 if empty.
func parsePrice(priceStr string) (float64, error) {
	if priceStr == "" {
		return 0.0, nil
	}

	priceVal, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0.0, fmt.Errorf("invalid price")
	}
	if priceVal < 0 {
		return 0.0, fmt.Errorf("price cannot be negative")
	}

	return priceVal, nil
}

func validateItemLink(rawLink string) (string, error) {
	link := strings.TrimSpace(rawLink)
	if link == "" {
		return "", nil
	}

	parsed, err := url.Parse(link)
	if err != nil {
		return "", errInvalidItemLink
	}
	isHTTP := strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https")
	if parsed.Host == "" || !isHTTP {
		return "", errInvalidItemLink
	}

	return link, nil
}

// toNullString converts a string to sql.NullString, with NULL for empty strings.
func toNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

// notifyItemCreated fetches the created item details and sends an email notification.
func (s *server) notifyItemCreated(ctx context.Context, result sql.Result) {
	itemID, err := result.LastInsertId()
	if err != nil {
		return
	}

	item, err := s.getItemByID(ctx, int(itemID))
	if err != nil {
		return
	}

	linkStr := ""
	if item.Link != nil {
		linkStr = *item.Link
	}
	s.emailService.notifyItemAdded(item.Name, item.FamilyMemberName, linkStr, item.Price)
}

// getItemByID fetches a single item with its family member name.
func (s *server) getItemByID(ctx context.Context, itemID int) (models.Item, error) {
	query := `SELECT items.id, items.family_member_id, items.name, items.link, items.price, items.reserved, family_members.name as family_member_name 
			  FROM items 
			  JOIN family_members ON items.family_member_id = family_members.id 
			  WHERE items.id = ?`

	row := s.db.QueryRowContext(ctx, query, itemID)
	return scanItemRow(row)
}

func (s *server) updateItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath("/api/items/", r.URL.Path)
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Invalid item ID")
		return
	}

	data, err := parseItemRequestBody(r)
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}
	link, err := validateItemLink(data.Link)
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	data.Link = link

	if err := s.validateItemNameUpdate(r.Context(), id, data); err != nil {
		s.renderError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.executeItemUpdate(r.Context(), id, data); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "Failed to update item")
		return
	}

	currentFam := getFamilyMemberIDForItem(s, r, id)
	s.renderItemList(w, r, currentFam)
}

// validateItemNameUpdate validates the item name update and checks for uniqueness if name is being changed.
func (s *server) validateItemNameUpdate(ctx context.Context, itemID int, data *requestBodyData) error {
	// If name is not being updated, skip validation
	if strings.TrimSpace(data.Name) == "" {
		return nil
	}

	normalizedName, err := validateAndNormalizeName(data.Name)
	if err != nil {
		return err
	}

	item, err := s.getItemByID(ctx, itemID)
	if err != nil {
		return fmt.Errorf("failed to fetch item: %w", err)
	}

	familyMemberID := item.FamilyMemberID
	if data.FamilyMemberID != "" {
		familyMemberID, err = strconv.Atoi(data.FamilyMemberID)
		if err != nil {
			return fmt.Errorf("invalid family member ID")
		}
	}

	exists, err := s.itemNameExists(ctx, normalizedName, familyMemberID, itemID)
	if err != nil {
		return fmt.Errorf("failed to check item name: %w", err)
	}
	if exists {
		return fmt.Errorf("an item with this name already exists for this family member")
	}

	return nil
}

// executeItemUpdate builds and executes the UPDATE query for an item.
func (s *server) executeItemUpdate(ctx context.Context, itemID int, data *requestBodyData) error {
	setParts, args, err := buildUpdateQuery(data)
	if err != nil {
		return err
	}

	if len(setParts) == 0 {
		return nil // No updates to perform
	}

	query := buildUpdateSQL(setParts)
	args = append(args, itemID)
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

// buildUpdateQuery constructs the SET clause and arguments for an UPDATE query based on provided data.
func buildUpdateQuery(data *requestBodyData) (setParts []string, args []any, err error) {
	setParts = []string{}
	args = []any{}

	if strings.TrimSpace(data.Name) != "" {
		setParts = append(setParts, "name = ?")
		args = append(args, data.Name)
	}

	if data.Link != "" {
		setParts = append(setParts, "link = ?")
		args = append(args, data.Link)
	} else {
		setParts = append(setParts, "link = NULL")
	}

	if strings.TrimSpace(data.Price) != "" {
		p, err := strconv.ParseFloat(data.Price, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid price")
		}
		setParts = append(setParts, "price = ?")
		args = append(args, p)
	} else if data.Price == "" {
		setParts = append(setParts, "price = NULL")
	}

	if strings.TrimSpace(data.FamilyMemberID) != "" {
		famID, err := strconv.Atoi(data.FamilyMemberID)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid family member ID")
		}
		setParts = append(setParts, "family_member_id = ?")
		args = append(args, famID)
	}

	return setParts, args, nil
}

// buildUpdateSQL constructs a parameterized UPDATE SQL statement.
// This is safe because setParts are constructed internally with trusted strings.
func buildUpdateSQL(setParts []string) string {
	return "UPDATE items SET " + strings.Join(setParts, ", ") + " WHERE id = ?"
}

// updateItemReservation is a helper function that handles reserving or unreserving an item.
func (s *server) updateItemReservation(w http.ResponseWriter, r *http.Request, reserved bool) {
	id, err := parseIDFromPath("/api/items/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if renderErr := templates.Error("Invalid item ID").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	itemName, familyMemberName := s.getItemDetailsForNotification(r.Context(), id)

	reservedVal := 0
	errMsg := "Failed to un-reserve item"
	if reserved {
		reservedVal = 1
		errMsg = "Failed to reserve item"
	}

	_, err = s.db.ExecContext(r.Context(), "UPDATE items SET reserved = ? WHERE id = ?", reservedVal, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if renderErr := templates.Error(errMsg).Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	if itemName != "" {
		s.emailService.notifyItemReserved(itemName, familyMemberName, reserved)
	}

	// Check if user is authenticated to determine what to render
	cookie, err := r.Cookie("session_token")
	isAuthenticated := err == nil && s.sessions.isValid(cookie.Value)

	if isAuthenticated {
		// In edit mode, render all items for the family member
		currentFam := getFamilyMemberIDForItem(s, r, id)
		s.renderItemList(w, r, currentFam)
	} else {
		// In view mode, return just the updated card HTML for the single item
		s.renderSingleItemCard(w, r, id)
	}
}

func (s *server) reserveItem(w http.ResponseWriter, r *http.Request) {
	s.updateItemReservation(w, r, true)
}

func (s *server) unreserveItem(w http.ResponseWriter, r *http.Request) {
	s.updateItemReservation(w, r, false)
}

func (s *server) deleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath("/api/items/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if renderErr := templates.Error("Invalid item ID").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	currentFam := getFamilyMemberIDForItem(s, r, id)
	itemName, familyMemberName := s.getItemDetailsForNotification(r.Context(), id)

	_, err = s.db.ExecContext(r.Context(), "DELETE FROM items WHERE id = ?", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := templates.Error("Failed to delete item").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}

	if itemName != "" {
		s.emailService.notifyItemDeleted(itemName, familyMemberName)
	}

	s.renderItemList(w, r, currentFam)
}

// getItemDetailsForNotification fetches the item name and family member name for email notifications.
func (s *server) getItemDetailsForNotification(ctx context.Context, itemID int) (itemName, familyMemberName string) {
	query := `SELECT items.name, family_members.name 
			  FROM items 
			  JOIN family_members ON items.family_member_id = family_members.id 
			  WHERE items.id = ?`

	err := s.db.QueryRowContext(ctx, query, itemID).Scan(&itemName, &familyMemberName)
	if err != nil {
		log.Printf("Error fetching item details for notification: %v", err)
	}

	return itemName, familyMemberName
}

// Removed reserveConfirm and reserveCancel - now using pure CSS with <details> element!

// getFamilyMemberIDForItem is a helper that retrieves the family member ID from the request header
// or derives it from the item's database record. This is used to preserve the current filter
// when rendering item lists after operations.
func getFamilyMemberIDForItem(s *server, r *http.Request, itemID int) string {
	currentFam := r.Header.Get("X-Family-Member-ID")
	if strings.TrimSpace(currentFam) == "" {
		// Derive from item if not provided
		var famID string
		err := s.db.QueryRowContext(r.Context(), "SELECT family_member_id FROM items WHERE id = ?", itemID).Scan(&famID)
		if err == nil {
			currentFam = famID
		}
	}
	return currentFam
}
