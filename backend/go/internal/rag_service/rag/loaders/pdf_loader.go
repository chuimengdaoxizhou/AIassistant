package loaders

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"bytes"
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

// PdfLoader implements the Loader interface for reading PDF files.
type PdfLoader struct{}

// NewPdfLoader creates a new PdfLoader.
func NewPdfLoader() *PdfLoader {
	return &PdfLoader{}
}

// Load reads a PDF file, extracts text and images from each page,
// and returns a Document for each page.
func (l *PdfLoader) Load(ctx context.Context, path string) ([]*schema.Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	pdfReader, err := model.NewPdfReader(f)
	if err != nil {
		return nil, err
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return nil, err
	}

	var documents []*schema.Document
	for i := 1; i <= numPages; i++ {
		page, err := pdfReader.GetPage(i)
		if err != nil {
			return nil, err
		}

		ex, err := extractor.New(page)
		if err != nil {
			return nil, err
		}

		text, err := ex.ExtractText()
		if err != nil {
			return nil, err
		}

		pageImages, err := ex.ExtractPageImages(nil)
		if err != nil {
			// Log or handle the error if images are not critical
			// For now, we'll just continue without images for this page
		}

		var imagesData [][]byte
		if pageImages != nil {
			for _, pImg := range pageImages.Images {
				goImg, err := pImg.Image.ToGoImage()
				if err != nil {
					continue
				}

				var buf bytes.Buffer
				if err := png.Encode(&buf, goImg); err != nil {
					continue
				}
				imagesData = append(imagesData, buf.Bytes())
			}
		}

		doc := &schema.Document{
			ID:   uuid.New().String(),
			Text: text,
			Metadata: map[string]interface{}{
				schema.MetadataKeyFileName:  filepath.Base(path),
				schema.MetadataKeyPageLabel: fmt.Sprintf("%d", i),
			},
		}

		if len(imagesData) > 0 {
			doc.Metadata[schema.MetadataKeyImage] = imagesData
		}

		documents = append(documents, doc)
	}

	return documents, nil
}

// compile-time check to ensure PdfLoader implements the Loader interface
var _ interfaces.Loader = (*PdfLoader)(nil)
