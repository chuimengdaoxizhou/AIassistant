package extractor

import (
	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/internal/models"
	"context"
	"encoding/json"
	"fmt"
)

const extractRelationsPrompt = `
You are an advanced algorithm designed to extract structured information from text to construct knowledge graphs. Your goal is to capture comprehensive and accurate information. Follow these key principles:

1. Extract only explicitly stated information from the text.
2. Establish relationships among the entities provided.
3. Use "USER_ID" as the source entity for any self-references (e.g., "I," "me," "my," etc.) in user messages.

Relationships:
    - Use consistent, general, and timeless relationship types.
    - Example: Prefer "professor" over "became_professor."
    - Relationships should only be established among the entities explicitly mentioned in the user message.

Entity Consistency:
    - Ensure that relationships are coherent and logically align with the context of the message.
    - Maintain consistent naming for entities across the extracted data.

Strive to construct a coherent and easily understandable knowledge graph by eshtablishing all the relationships among the entities and adherence to the user’s context.

Adhere strictly to these guidelines to ensure high-quality knowledge graph extraction.`

// GraphExtractor is an implementation of the Extractor interface that uses an LLM to extract relations.
type GraphExtractor struct {
	llm llm.LLM
}

// NewGraphExtractor creates a new GraphExtractor.
func NewGraphExtractor(llm llm.LLM) *GraphExtractor {
	return &GraphExtractor{llm: llm}
}

// Extract extracts relations from HistoryContent using an LLM.
func (e *GraphExtractor) Extract(ctx context.Context, content *models.HistoryContent) ([]*models.Relation, error) {
	// 1. Construct the prompt from the HistoryContent.
	var conversation string
	for _, part := range content.Content.Parts {
		conversation += fmt.Sprintf("%s: %s\n", content.Content.Role, part.Text)
	}

	prompt := fmt.Sprintf("%s\n\nConversation:\n%s", extractRelationsPrompt, conversation)

	// 2. Call the LLM to get the extracted relations.
	llmReq := &models.GenerateContentRequest{
		Content: []models.Content{
			{
				Parts: []*models.Part{
					{Text: prompt},
				},
			},
		},
	}
	llmResp, err := e.llm.GenerateContent(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// 3. Parse the LLM response.
	var response struct {
		Relations []*models.Relation `json:"relations"`
	}

	// Assuming the response is in the first part of the first content
	jsonString := llmResp.Content[0].Parts[0].Text
	if err := json.Unmarshal([]byte(jsonString), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 4. Set the UserID for the relations.
	for _, rel := range response.Relations {
		rel.UserID = content.User
	}

	return response.Relations, nil
}
