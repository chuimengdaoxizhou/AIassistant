package embeddings

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"context"

	"Jarvis_2.0/backend/go/internal/embedding"
)

// GenaiAdapter adapts the project's specific GoogleModel to the generic EmbeddingModel interface.
type GenaiAdapter struct {
	client *embedding.GoogleModel
}

// NewGenaiAdapter creates a new adapter for the GoogleModel.
func NewGenaiAdapter(client *embedding.GoogleModel) *GenaiAdapter {
	return &GenaiAdapter{client: client}
}

// Embed calls the underlying client's EmbedBatch method to satisfy the EmbeddingModel interface.
func (a *GenaiAdapter) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return a.client.EmbedBatch(ctx, texts)
}

// compile-time check to ensure GenaiAdapter implements the EmbeddingModel interface
var _ interfaces.EmbeddingModel = (*GenaiAdapter)(nil)
