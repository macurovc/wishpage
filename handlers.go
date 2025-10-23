package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"

	"wishpage/models"
	"wishpage/templates"
)

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Landing page - show view with no family member selected
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	familyMembers, err := s.getAllFamilyMembers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to fetch family members").Render(r.Context(), w)
		return
	}

	// Show landing page with no member selected (selectedFamilyID = 0)
	templates.View([]models.Item{}, familyMembers, 0).Render(r.Context(), w)
}

func (s *server) handleFamilyView(w http.ResponseWriter, r *http.Request) {
	// Public view of a specific family member's items
	// Extract name from URL path: /family/alice -> "alice"
	name := strings.TrimPrefix(r.URL.Path, "/family/")
	if name == "" || strings.Contains(name, "/") {
		http.NotFound(w, r)
		return
	}

	// Get family member by name
	var familyMemberID int
	err := s.db.QueryRowContext(r.Context(), "SELECT id FROM family_members WHERE name = ?", name).Scan(&familyMemberID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			log.Printf("Error fetching family member %s: %v", name, err)
			w.WriteHeader(http.StatusInternalServerError)
			templates.Error("Failed to fetch family member").Render(r.Context(), w)
		}
		return
	}

	items, err := s.getAllItems(r.Context(), strconv.Itoa(familyMemberID))
	if err != nil {
		log.Printf("Error fetching items for family member %s: %v", name, err)
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to fetch items").Render(r.Context(), w)
		return
	}

	familyMembers, err := s.getAllFamilyMembers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to fetch family members").Render(r.Context(), w)
		return
	}

	templates.View(items, familyMembers, familyMemberID).Render(r.Context(), w)
}

func (s *server) handleEdit(w http.ResponseWriter, r *http.Request) {
	// Single-page edit view showing all family members and their items
	if r.URL.Path != "/edit" {
		http.NotFound(w, r)
		return
	}

	familyMembers, err := s.getAllFamilyMembers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to fetch family members").Render(r.Context(), w)
		return
	}

	// Get ALL items (no filtering)
	items, err := s.getAllItems(r.Context(), "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		templates.Error("Failed to fetch items").Render(r.Context(), w)
		return
	}

	templates.Edit(items, familyMembers).Render(r.Context(), w)
}
