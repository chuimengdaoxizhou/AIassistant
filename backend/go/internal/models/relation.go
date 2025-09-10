package models

// Relation represents a relationship between two entities.
type Relation struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Type     string `json:"type"`
	UserID   string `json:"user_id"`
}
