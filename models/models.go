package models

type FamilyMember struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Item struct {
	ID               int      `json:"id"`
	FamilyMemberID   int      `json:"family_member_id"`
	FamilyMemberName string   `json:"family_member_name"`
	Name             string   `json:"name"`
	Link             *string  `json:"link"`
	Price            *float64 `json:"price"`
	Reserved         bool     `json:"reserved"`
}
