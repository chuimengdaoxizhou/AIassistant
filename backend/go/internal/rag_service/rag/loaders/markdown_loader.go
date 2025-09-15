package loaders

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"context"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/uuid"
)

// MarkdownLoader implements the Loader interface for reading Markdown (.md) files.
type MarkdownLoader struct{}

// NewMarkdownLoader creates a new MarkdownLoader.
func NewMarkdownLoader() *MarkdownLoader {
	return &MarkdownLoader{}
}

// imageRegex is used to find Markdown image syntax (e.g., ![alt text](path/to/image.jpg))
var imageRegex = regexp.MustCompile(`!\[.*?\]\((.*?)\)`)

// Load reads a Markdown file, its text content, and any referenced local images.
func (l *MarkdownLoader) Load(ctx context.Context, path string) ([]*schema.Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	textContent := string(content)

	var imagesData [][]byte
	matches := imageRegex.FindAllStringSubmatch(textContent, -1)
	baseDir := filepath.Dir(path)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		imagePath := match[1]

		// We only handle local file paths, not URLs
		if !filepath.IsAbs(imagePath) {
			imagePath = filepath.Join(baseDir, imagePath)
		}

		imageData, err := os.ReadFile(imagePath)
		if err == nil { // Silently ignore images that can't be read
			imagesData = append(imagesData, imageData)
		}
	}

	doc := &schema.Document{
		ID:   uuid.New().String(),
		Text: textContent,
		Metadata: map[string]interface{}{
			schema.MetadataKeyFileName: filepath.Base(path),
		},
	}

	if len(imagesData) > 0 {
		doc.Metadata[schema.MetadataKeyImage] = imagesData
	}

	return []*schema.Document{doc}, nil
}

// compile-time check to ensure MarkdownLoader implements the Loader interface
var _ interfaces.Loader = (*MarkdownLoader)(nil)
