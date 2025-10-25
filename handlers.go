package main

import (
	"log"
	"net/http"

	"wishpage/templates"
)

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Landing page - load ALL items and family members at once
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	familyMembers, err := s.getAllFamilyMembers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if renderErr := templates.Error("Failed to fetch family members").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	// Load ALL items at once (no filtering)
	allItems, err := s.getAllItems(r.Context(), "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := templates.Error("Failed to fetch items").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}

	// Render everything - CSS will handle showing/hiding based on hash
	if err := templates.View(allItems, familyMembers, 0).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering template: %v", err)
	}
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
		if renderErr := templates.Error("Failed to fetch family members").Render(r.Context(), w); renderErr != nil {
			log.Printf("Error rendering template: %v", renderErr)
		}
		return
	}

	// Get ALL items (no filtering)
	items, err := s.getAllItems(r.Context(), "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := templates.Error("Failed to fetch items").Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
		return
	}

	if err := templates.Edit(items, familyMembers).Render(r.Context(), w); err != nil {
		log.Printf("Error rendering template: %v", err)
	}
}
