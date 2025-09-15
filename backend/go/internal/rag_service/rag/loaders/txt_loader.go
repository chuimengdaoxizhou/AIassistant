package loaders

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"context"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// TxtLoader implements the Loader interface for reading plain text files.
type TxtLoader struct{}

// NewTxtLoader creates a new TxtLoader.
func NewTxtLoader() *TxtLoader {
	return &TxtLoader{}
}

// Load reads a text file from the given path and returns it as a single Document.
func (l *TxtLoader) Load(ctx context.Context, path string) ([]*schema.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	doc := &schema.Document{
		ID:   uuid.New().String(),
		Text: string(content),
		Metadata: map[string]interface{}{
			"file_name": filepath.Base(path),
		},
	}

	return []*schema.Document{doc}, nil
}

// compile-time check to ensure TxtLoader implements the Loader interface
var _ interfaces.Loader = (*TxtLoader)(nil)
