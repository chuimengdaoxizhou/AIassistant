package store

import (
	"Jarvis_2.0/backend/go/internal/models"
	"context"
)

// Store defines the interface for storing and retrieving facts.
type Store interface {
	GetFacts(ctx context.Context, userID string, query string) ([]*models.Fact, error)
	AddFact(ctx context.Context, fact *models.Fact) error
	UpdateFact(ctx context.Context, fact *models.Fact) error
	DeleteFact(ctx context.Context, factID string) error
}
