package models

import "time"

// RagFolder represents a user-defined folder to categorize RAG documents.
// The combination of UserID and Name should be unique.
type RagFolder struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    string    `gorm:"index:idx_user_folder,unique;not null;size:255"` // Indexed for fast lookups by user
	Name      string    `gorm:"index:idx_user_folder,unique;not null;size:255"` // Folder name, unique per user
	CreatedAt time.Time
	UpdatedAt time.Time
}
