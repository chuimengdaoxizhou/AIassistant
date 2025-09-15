package llms

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"context"
	"fmt"

	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/internal/models"
)

// GeminiAdapter adapts the existing project-specific Gemini client to the generic LLM interface.
type GeminiAdapter struct {
	client *llm.Gemini
}

// NewGeminiAdapter creates a new adapter.
func NewGeminiAdapter(client *llm.Gemini) *GeminiAdapter {
	return &GeminiAdapter{client: client}
}

// Generate takes a simple string prompt, wraps it into the complex request structure
// required by the existing Gemini client, calls the client, and unwraps the response
// to return a simple string answer.
func (a *GeminiAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	// 1. Wrap the simple string prompt into the complex request object.
	req := &models.GenerateContentRequest{
		Content: []models.Content{
			{
				Parts: []*models.Part{
					{Text: prompt},
				},
			},
		},
	}

	// 2. Call the existing client's method.
	resp, err := a.client.GenerateContent(ctx, req)
	if err != nil {
		return "", fmt.Errorf("gemini client failed to generate content: %w", err)
	}

	// 3. Unwrap the complex response object to extract the plain text answer.
	// We'll take the text from the first part of the first content of the first candidate.
	if len(resp.Content) > 0 && len(resp.Content[0].Parts) > 0 {
		return resp.Content[0].Parts[0].Text, nil
	}

	return "", fmt.Errorf("gemini response was empty or in an unexpected format")
}

// compile-time check to ensure GeminiAdapter implements the LLM interface
var _ interfaces.LLM = (*GeminiAdapter)(nil)
