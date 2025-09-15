package interfaces

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"context"
)

// Loader is the interface for loading data from a source (e.g., file, URL)
// and converting it into a list of Document objects.
type Loader interface {
	Load(ctx context.Context, path string) ([]*schema.Document, error)
}

// Splitter is the interface for splitting a list of Documents into smaller chunks.
type Splitter interface {
	Split(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error)
}

// DocStore is the interface for storing and retrieving document chunks by their ID.
type DocStore interface {
	Add(ctx context.Context, userID string, docs map[string]*schema.Document) error
	Get(ctx context.Context, userID string, ids []string) (map[string]*schema.Document, error)
	Delete(ctx context.Context, userID string, ids []string) error
}

// VectorStore is the interface for storing and querying document vectors.
type VectorStore interface {
	Add(ctx context.Context, docs []*schema.Document) error // Add is multi-tenant via metadata in docs
	Query(ctx context.Context, embedding []float32, topK int, filters map[string]interface{}) ([]*schema.Document, error)
}

// Reranker is the interface for re-ordering a list of retrieved documents to improve relevance.
type Reranker interface {
	Rerank(ctx context.Context, query string, docs []*schema.Document) ([]*schema.Document, error)
}

// EmbeddingModel is the interface for a text embedding model.
type EmbeddingModel interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// LLM is the interface for a large language model that can generate text.
type LLM interface {
	Generate(ctx context.Context, prompt string) (string, error)
}
