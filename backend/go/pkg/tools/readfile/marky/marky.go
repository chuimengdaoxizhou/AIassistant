package marky

import (
	"Jarvis_2.0/backend/go/pkg/tools/readfile/converters"
	"fmt"
	"slices"

	"github.com/gabriel-vasile/mimetype"
)

// Marky manages document converters and provides conversion functionality.
type Marky struct {
	Converters []converters.Converter
}

// New creates a new Marky instance with all the available converters registered.
func New() *Marky {
	m := &Marky{}

	// Register all the available converters
	m.RegisterConverter(converters.NewCsvConverter())
	m.RegisterConverter(converters.NewDocxConverter())
	m.RegisterConverter(converters.NewEpubConverter())
	m.RegisterConverter(converters.NewExcelConverter())
	m.RegisterConverter(converters.NewHTMLConverter())
	m.RegisterConverter(converters.NewIpynbConverter())
	m.RegisterConverter(converters.NewPdfConverter())
	m.RegisterConverter(converters.NewPptxConverter())

	return m
}

type IMarky interface {
	Convert(path string) (string, error)
}

// RegisterConverter adds a new document converter to the available converters.
func (m *Marky) RegisterConverter(converter converters.Converter) {
	m.Converters = append(m.Converters, converter)
}

// Convert processes a document file and converts it to markdown format.
// Returns the markdown content and an error if the conversion fails.
func (m *Marky) Convert(path string) (string, error) {
	// Detect MIME type from file content - this is mandatory
	mtype, err := mimetype.DetectFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to detect MIME type: %w", err)
	}

	// Find a converter that can handle this MIME type
	for _, converter := range m.Converters {
		if accepts(mtype, converter.AcceptedExtensions(), converter.AcceptedMimeTypes()) {
			return converter.Load(path)
		}
	}

	return "", fmt.Errorf("no converter found for MIME type: %s", mtype.String())
}

func accepts(mtype *mimetype.MIME, extensions, mtypes []string) bool {
	if slices.Contains(extensions, mtype.Extension()) {
		return true
	}

	return slices.ContainsFunc(mtypes, mtype.Is)
}
