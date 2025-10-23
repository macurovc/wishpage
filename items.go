package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"wishpage/models"
	"wishpage/templates"
)

func (s *server) handleItems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		familyMemberID := r.URL.Query().Get("family_member_id")
		s.renderItemList(w, r, familyMemberID)
	case http.MethodPost:
		s.requireAuth(s.createItem)(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		templates.Error("Method not allowed").Render(r.Context(), w)
	}
}

func (s *server) handleItemsID(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/reserve-confirm") {
		if r.Method == http.MethodGet {
			s.reserveConfirm(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
			templates.Error("Method not allowed").Render(r.Context(), w)
		}
		return
	} else if strings.HasSuffix(r.URL.Path, "/reserve-cancel") {
		if r.Method == http.MethodGet {
			s.reserveCancel(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
			templates.Error("Method not allowed").Render(r.Context(), w)
		}
		return
	} else if strings.HasSuffix(r.URL.Path, "/reserve") {
		if r.Method == http.MethodPut {
			s.reserveItem(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
			templates.Error("Method not allowed").Render(r.Context(), w)
		}
		return
	} else if strings.HasSuffix(r.URL.Path, "/unreserve") {
		if r.Method == http.MethodPut {
			s.requireAuth(s.unreserveItem)(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
			templates.Error("Method not allowed").Render(r.Context(), w)
		}
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.requireAuth(s.updateItem)(w, r)
	case http.MethodDelete:
		s.requireAuth(s.deleteItem)(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		templates.Error("Method not allowed").Render(r.Context(), w)
	}
}

func (s *server) getAllItems(ctx context.Context, familyMemberID string) ([]models.Item, error) {
	query := `
        SELECT items.id, items.family_member_id, items.name, items.link, items.price, items.reserved, family_members.name as family_member_name 
        FROM items 
        JOIN family_members ON items.family_member_id = family_members.id
    `
	args := []interface{}{}
	if familyMemberID != "" {
		query += " WHERE family_member_id = ?"
		args = append(args, familyMemberID)
	}
	query += " ORDER BY CASE WHEN items.price IS NULL THEN 0 ELSE 1 END, items.price ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query items for family member %s: %w", familyMemberID, err)
	}
	defer rows.Close()

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
		templates.Error("Failed to fetch items").Render(r.Context(), w)
		return
	}

	members, err := s.getAllFamilyMembers(r.Context())
	if err != nil {
		log.Printf("Error fetching family members: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to fetch family members").Render(r.Context(), w)
		return
	}

	// Check if user is authenticated
	cookie, err := r.Cookie("session_token")
	isAuthenticated := err == nil && s.sessions.isValid(cookie.Value)

	if isAuthenticated {
		// In edit mode, render items for the member using the new template
		templates.ItemsForMember(items).Render(r.Context(), w)
	} else {
		templates.ItemListView(items, members).Render(r.Context(), w)
	}
}

func (s *server) createItem(w http.ResponseWriter, r *http.Request) {
	data, err := parseItemRequestBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error("Invalid request body").Render(r.Context(), w)
		return
	}

	name, err := validateAndNormalizeName(data.Name)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error(err.Error()).Render(r.Context(), w)
		return
	}
	if len(name) > 200 {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error("Item name too long (max 200 characters)").Render(r.Context(), w)
		return
	}

	familyMemberID, familyMemberIDStr, err := s.resolveFamilyMemberID(r, data.FamilyMemberID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error(err.Error()).Render(r.Context(), w)
		return
	}

	price, err := parsePrice(data.Price)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error(err.Error()).Render(r.Context(), w)
		return
	}

	link := strings.TrimSpace(data.Link)

	// Check if an item with this name already exists for this family member
	exists, err := s.itemNameExists(r.Context(), name, familyMemberID, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to check item name").Render(r.Context(), w)
		return
	}
	if exists {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error("An item with this name already exists for this family member").Render(r.Context(), w)
		return
	}

	// Convert to SQL NULL types: empty string becomes NULL for link, empty input becomes NULL for price
	linkParam := toNullString(link)
	priceParam := sql.NullFloat64{Float64: price, Valid: data.Price != ""}

	result, err := s.db.ExecContext(r.Context(),
		"INSERT INTO items (family_member_id, name, link, price) VALUES (?, ?, ?, ?)",
		familyMemberID, name, linkParam, priceParam)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to add item").Render(r.Context(), w)
		return
	}

	s.notifyItemCreated(r.Context(), result)
	s.renderItemList(w, r, familyMemberIDStr)
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
func (s *server) itemNameExists(ctx context.Context, name string, familyMemberID int, excludeItemID int) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM items WHERE family_member_id = ? AND name = ? AND id != ?"
	err := s.db.QueryRowContext(ctx, query, familyMemberID, name, excludeItemID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check item name uniqueness: %w", err)
	}
	return count > 0, nil
}

// resolveFamilyMemberID determines the family member ID from body, header, or defaults to first available.
func (s *server) resolveFamilyMemberID(r *http.Request, bodyID string) (int, string, error) {
	familyMemberIDStr := bodyID
	if familyMemberIDStr == "" {
		familyMemberIDStr = r.Header.Get("X-Family-Member-ID")
		if familyMemberIDStr == "" {
			members, err := s.getAllFamilyMembers(r.Context())
			if err == nil && len(members) > 0 {
				familyMemberIDStr = strconv.Itoa(members[0].ID)
			}
			if familyMemberIDStr == "" {
				return 0, "", fmt.Errorf("missing family member selection")
			}
		}
	}

	familyMemberID, err := strconv.Atoi(familyMemberIDStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid family member ID")
	}

	return familyMemberID, familyMemberIDStr, nil
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
		w.WriteHeader(http.StatusBadRequest)
		templates.Error("Invalid item ID").Render(r.Context(), w)
		return
	}

	data, err := parseItemRequestBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error("Invalid request body").Render(r.Context(), w)
		return
	}

	// If name is being updated, validate it and check uniqueness
	if strings.TrimSpace(data.Name) != "" {
		name, err := validateAndNormalizeName(data.Name)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			templates.Error(err.Error()).Render(r.Context(), w)
			return
		}

		// Get current item to determine family member ID
		item, err := s.getItemByID(r.Context(), id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			templates.Error("Failed to fetch item").Render(r.Context(), w)
			return
		}

		// Use the new family member ID if it's being changed, otherwise use the current one
		familyMemberID := item.FamilyMemberID
		if data.FamilyMemberID != "" {
			familyMemberID, err = strconv.Atoi(data.FamilyMemberID)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				templates.Error("invalid family member ID").Render(r.Context(), w)
				return
			}
		}

		// Check uniqueness
		exists, err := s.itemNameExists(r.Context(), name, familyMemberID, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			templates.Error("Failed to check item name").Render(r.Context(), w)
			return
		}
		if exists {
			w.WriteHeader(http.StatusBadRequest)
			templates.Error("An item with this name already exists for this family member").Render(r.Context(), w)
			return
		}
	}

	setParts, args, err := buildUpdateQuery(data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error(err.Error()).Render(r.Context(), w)
		return
	}

	if len(setParts) > 0 {
		query := "UPDATE items SET " + strings.Join(setParts, ", ") + " WHERE id = ?"
		args = append(args, id)
		if _, err := s.db.ExecContext(r.Context(), query, args...); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			templates.Error("Failed to update item").Render(r.Context(), w)
			return
		}
	}

	currentFam := getFamilyMemberIDForItem(s, r, id)
	s.renderItemList(w, r, currentFam)
}

// buildUpdateQuery constructs the SET clause and arguments for an UPDATE query based on provided data.
func buildUpdateQuery(data *requestBodyData) ([]string, []any, error) {
	setParts := []string{}
	args := []any{}

	if strings.TrimSpace(data.Name) != "" {
		setParts = append(setParts, "name = ?")
		args = append(args, data.Name)
	}

	if strings.TrimSpace(data.Link) != "" {
		setParts = append(setParts, "link = ?")
		args = append(args, data.Link)
	} else if data.Link == "" {
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

func (s *server) reserveItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath("/api/items/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error("Invalid item ID").Render(r.Context(), w)
		return
	}

	itemName, familyMemberName := s.getItemDetailsForNotification(r.Context(), id)

	_, err = s.db.ExecContext(r.Context(), "UPDATE items SET reserved = 1 WHERE id = ?", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to reserve item").Render(r.Context(), w)
		return
	}

	if itemName != "" {
		s.emailService.notifyItemReserved(itemName, familyMemberName, true)
	}

	currentFam := getFamilyMemberIDForItem(s, r, id)
	s.renderItemList(w, r, currentFam)
}

func (s *server) unreserveItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath("/api/items/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error("Invalid item ID").Render(r.Context(), w)
		return
	}

	itemName, familyMemberName := s.getItemDetailsForNotification(r.Context(), id)

	_, err = s.db.ExecContext(r.Context(), "UPDATE items SET reserved = 0 WHERE id = ?", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to un-reserve item").Render(r.Context(), w)
		return
	}

	if itemName != "" {
		s.emailService.notifyItemReserved(itemName, familyMemberName, false)
	}

	currentFam := getFamilyMemberIDForItem(s, r, id)
	s.renderItemList(w, r, currentFam)
}

func (s *server) deleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath("/api/items/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error("Invalid item ID").Render(r.Context(), w)
		return
	}

	currentFam := getFamilyMemberIDForItem(s, r, id)
	itemName, familyMemberName := s.getItemDetailsForNotification(r.Context(), id)

	_, err = s.db.ExecContext(r.Context(), "DELETE FROM items WHERE id = ?", id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to delete item").Render(r.Context(), w)
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

func (s *server) reserveConfirm(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath("/api/items/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error("Invalid item ID").Render(r.Context(), w)
		return
	}

	templates.ReserveConfirmation(id).Render(r.Context(), w)
}

func (s *server) reserveCancel(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath("/api/items/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		templates.Error("Invalid item ID").Render(r.Context(), w)
		return
	}

	templates.ReserveButton(id).Render(r.Context(), w)
}

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
