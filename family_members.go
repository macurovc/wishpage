package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"wishpage/models"
	"wishpage/templates"
)

func (s *server) handleFamilyMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getFamilyMembers(w, r)
	case http.MethodPost:
		s.requireAuth(s.createFamilyMember)(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := templates.Error("Method not allowed").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
	}
}

func (s *server) handleFamilyMembersID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		s.requireAuth(s.updateFamilyMember)(w, r)
	case http.MethodDelete:
		s.requireAuth(s.deleteFamilyMember)(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := templates.Error("Method not allowed").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
	}
}

func (s *server) getAllFamilyMembers(ctx context.Context) ([]models.FamilyMember, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name FROM family_members ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query family members: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Error closing rows: %v", err)
		}
	}()

	members := []models.FamilyMember{}
	for rows.Next() {
		var member models.FamilyMember
		if err := rows.Scan(&member.ID, &member.Name); err != nil {
			return nil, fmt.Errorf("failed to scan family member: %w", err)
		}
		members = append(members, member)
	}
	return members, nil
}

func (s *server) getFamilyMembers(w http.ResponseWriter, r *http.Request) {
	familyMembers, err := s.getAllFamilyMembers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := templates.Error("Failed to fetch family members").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(familyMembers); err != nil {
		log.Printf("Error encoding family members response: %v", err)
	}
}

func (s *server) createFamilyMember(w http.ResponseWriter, r *http.Request) {
	data, err := parseRequestBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if renderErr := templates.Error("Invalid request body").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	name := strings.TrimSpace(data["name"])
	if name == "" {
		name = "New family member"
	}
	if len(name) > 200 {
		w.WriteHeader(http.StatusBadRequest)
		if renderErr := templates.Error("Name too long (max 200 characters)").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	result, err := s.db.ExecContext(r.Context(), "INSERT INTO family_members (name) VALUES (?)", name)
	if err != nil {
		s.handleFamilyMemberInsertError(w, r, err)
		return
	}

	// Get the newly created family member
	newID, err := result.LastInsertId()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := templates.Error("Failed to get new member ID").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}

	newMember := models.FamilyMember{
		Name: name,
		ID:   int(newID),
	}

	// Send email notification
	s.emailService.notifyFamilyMemberAdded(name)

	// Render just the new family section
	// The hx-swap="beforebegin" on the form will insert it before the add form
	w.Header().Set("Content-Type", "text/html")
	if err := templates.FamilyMemberSection(newMember, []models.Item{}).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering template: %v", err)
	}
}

// handleFamilyMemberInsertError handles database errors when inserting a family member.
func (s *server) handleFamilyMemberInsertError(w http.ResponseWriter, r *http.Request, err error) {
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		w.WriteHeader(http.StatusBadRequest)
		if renderErr := templates.Error("A family member with this name already exists").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	log.Printf("Error creating family member: %v", err)
	w.WriteHeader(http.StatusInternalServerError)
	if renderErr := templates.Error("Failed to add family member").Render(r.Context(), w); renderErr != nil {
		log.Printf("Error rendering template: %v", renderErr)
	}
}

func (s *server) deleteFamilyMember(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath("/api/family_members/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if renderErr := templates.Error("Invalid family member ID").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	// Get family member details before deletion for notification
	var name string
	err = s.db.QueryRowContext(r.Context(), "SELECT name FROM family_members WHERE id = ?", id).Scan(&name)
	if err != nil {
		log.Printf("Error fetching family member details for notification: %v", err)
	}

	_, err = s.db.ExecContext(r.Context(), "DELETE FROM family_members WHERE id = ?", id)
	if err != nil {
		slog.Error("Error deleting family member", "family_member_id", id, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := templates.Error("Failed to delete family member").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}

	// Send email notification
	if name != "" {
		s.emailService.notifyFamilyMemberDeleted(name)
	}

	// Return empty response - HTMX will remove the element with hx-swap="outerHTML"
	w.WriteHeader(http.StatusOK)
}

func (s *server) updateFamilyMember(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath("/api/family_members/", r.URL.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if renderErr := templates.Error("Invalid family member ID").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	data, err := parseRequestBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if renderErr := templates.Error("Invalid request body").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	name := data["name"]
	if strings.TrimSpace(name) == "" {
		w.WriteHeader(http.StatusBadRequest)
		if renderErr := templates.Error("Name is required").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	if _, err := s.db.ExecContext(r.Context(), "UPDATE family_members SET name = ? WHERE id = ?", name, id); err != nil {
		slog.Error("Error updating family member", "family_member_id", id, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		if err := templates.Error("Failed to update family member").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}

	// Return OK - this feature is not used in the current UI
	w.WriteHeader(http.StatusOK)
}
