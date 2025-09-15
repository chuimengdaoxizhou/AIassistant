package loaders

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"context"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

// XlsxLoader implements the Loader interface for reading Excel (.xlsx) files.
type XlsxLoader struct{}

// NewXlsxLoader creates a new XlsxLoader.
func NewXlsxLoader() *XlsxLoader {
	return &XlsxLoader{}
}

// Load reads an .xlsx file, converting each sheet to a Markdown table
// and extracting images. It returns a Document for each sheet.
func (l *XlsxLoader) Load(ctx context.Context, path string) ([]*schema.Document, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var documents []*schema.Document
	sheetList := f.GetSheetList()

	for _, sheetName := range sheetList {

		rows, err := f.GetRows(sheetName)
		if err != nil {
			// Skip sheet if rows can't be read
			continue
		}

		// Convert sheet data to Markdown table
		var mdBuilder strings.Builder
		if len(rows) > 0 {
			// Header
			mdBuilder.WriteString("| " + strings.Join(rows[0], " | ") + " |\n")
			// Separator
			mdBuilder.WriteString("|" + strings.Repeat("---", len(rows[0])) + "\n")
			// Body
			for _, row := range rows[1:] {
				mdBuilder.WriteString("| " + strings.Join(row, " | ") + " |\n")
			}
		}

		// Extract pictures from the sheet
		var imagesData [][]byte
		pictures, err := f.GetPictures(sheetName, "")
		if err == nil {
			for _, pic := range pictures {
				imagesData = append(imagesData, pic.File)
			}
		}

		doc := &schema.Document{
			ID:   uuid.New().String(),
			Text: mdBuilder.String(),
			Metadata: map[string]interface{}{
				schema.MetadataKeyFileName: filepath.Base(path),
				"sheet_name":               sheetName,
			},
		}

		if len(imagesData) > 0 {
			doc.Metadata[schema.MetadataKeyImage] = imagesData
		}

		documents = append(documents, doc)
	}

	return documents, nil
}

// compile-time check to ensure XlsxLoader implements the Loader interface
var _ interfaces.Loader = (*XlsxLoader)(nil)
