package models

import "time"

// Fact represents a piece of information with its metadata.
type Fact struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Vector    []float32 `json:"vector"`
	Source    string    `json:"source"`
	StartTime time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}
