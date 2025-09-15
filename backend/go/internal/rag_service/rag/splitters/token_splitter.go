package splitters

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkoukk/tiktoken-go"
)

// TokenSplitter implements the Splitter interface to split documents based on token count.
type TokenSplitter struct {
	ChunkSize    int
	ChunkOverlap int
	tokenizer    *tiktoken.Tiktoken
}

// NewTokenSplitter creates a new TokenSplitter.
// It initializes a tokenizer for the specified model.
func NewTokenSplitter(chunkSize, chunkOverlap int) (*TokenSplitter, error) {
	// Using "cl100k_base" which is the tokenizer for gpt-4, gpt-3.5-turbo, and text-embedding-ada-002
	tke, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, fmt.Errorf("failed to get tiktoken encoding: %w", err)
	}

	return &TokenSplitter{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
		tokenizer:    tke,
	}, nil
}

// Split splits a list of documents into smaller chunks based on the token size.
func (s *TokenSplitter) Split(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
	var chunks []*schema.Document

	for _, doc := range docs {
		tokens := s.tokenizer.Encode(doc.Text, nil, nil)
		step := s.ChunkSize - s.ChunkOverlap

		for start := 0; start < len(tokens); start += step {
			end := start + s.ChunkSize
			if end > len(tokens) {
				end = len(tokens)
			}

			// Decode the chunk of tokens back to text
			chunkText := s.tokenizer.Decode(tokens[start:end])

			// Create a new document for the chunk
			newDoc := &schema.Document{
				ID:   uuid.New().String(),
				Text: chunkText,
				// Deep copy metadata to avoid sharing maps
				Metadata: s.copyMetadata(doc.Metadata),
			}

			// Add chunk-specific metadata
			newDoc.Metadata["original_doc_id"] = doc.ID
			newDoc.Metadata["chunk_number"] = (start / step) + 1

			chunks = append(chunks, newDoc)

			if end == len(tokens) {
				break
			}
		}
	}

	return chunks, nil
}

func (s *TokenSplitter) copyMetadata(md map[string]interface{}) map[string]interface{} {
	if md == nil {
		return make(map[string]interface{})
	}
	newMd := make(map[string]interface{}, len(md))
	for k, v := range md {
		newMd[k] = v
	}
	return newMd
}

// compile-time check to ensure TokenSplitter implements the Splitter interface
var _ interfaces.Splitter = (*TokenSplitter)(nil)
