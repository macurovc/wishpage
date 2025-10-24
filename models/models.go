// Package models defines the data structures used in the wishlist application.
package models

// FamilyMember represents a member of a family with their own wishlist.
type FamilyMember struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

// Item represents a wishlist item associated with a family member.
type Item struct {
	Name             string   `json:"name"`
	FamilyMemberName string   `json:"family_member_name"`
	Link             *string  `json:"link"`
	Price            *float64 `json:"price"`
	ID               int      `json:"id"`
	FamilyMemberID   int      `json:"family_member_id"`
	Reserved         bool     `json:"reserved"`
}
