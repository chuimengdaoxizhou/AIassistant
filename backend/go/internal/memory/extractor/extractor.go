package extractor

import (
	"Jarvis_2.0/backend/go/internal/models"
	"context"
)

// Extractor defines the interface for extracting facts from content.
type Extractor interface {
	Extract(ctx context.Context, content *models.HistoryContent) ([]*models.Fact, error)
}
