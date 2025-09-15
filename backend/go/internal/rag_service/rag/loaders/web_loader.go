package loaders

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/net/html"
)

// WebLoader implements the Loader interface for fetching and parsing web pages.
type WebLoader struct{}

// NewWebLoader creates a new WebLoader.
func NewWebLoader() *WebLoader {
	return &WebLoader{}
}

// Load fetches content from a URL, extracts the text, and returns it as a single Document.
func (l *WebLoader) Load(ctx context.Context, url string) ([]*schema.Document, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Extract text from the HTML body.
	text, err := extractText(resp.Body)
	if err != nil {
		return nil, err
	}

	doc := &schema.Document{
		ID:   uuid.New().String(),
		Text: text,
		Metadata: map[string]interface{}{
			"source_url": url,
		},
	}

	return []*schema.Document{doc}, nil
}

// extractText parses an HTML document and extracts all human-readable text,
// stripping away tags and scripts.
func extractText(body io.Reader) (string, error) {
	z := html.NewTokenizer(body)
	var sb strings.Builder
	var inScript, inStyle bool

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return sb.String(), nil
			}
			return "", z.Err()
		case html.StartTagToken, html.EndTagToken:
			tn, _ := z.TagName()
			tag := string(tn)
			if tag == "script" {
				inScript = (tt == html.StartTagToken)
			} else if tag == "style" {
				inStyle = (tt == html.StartTagToken)
			}
		case html.TextToken:
			if !inScript && !inStyle {
				// Append text content, ensuring spaces between words.
				text := strings.TrimSpace(string(z.Text()))
				if len(text) > 0 {
					sb.WriteString(text)
					sb.WriteString(" ")
				}
			}
		}
	}
}

// compile-time check to ensure WebLoader implements the Loader interface
var _ interfaces.Loader = (*WebLoader)(nil)
