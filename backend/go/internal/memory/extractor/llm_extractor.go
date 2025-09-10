package extractor

import (
	"Jarvis_2.0/backend/go/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// LlmExtractor is an implementation of the Extractor interface that uses the LangExtract Python script to extract facts.
type LlmExtractor struct {
	pythonScriptPath string
}

// NewLlmExtractor creates a new LlmExtractor.
func NewLlmExtractor(pythonScriptPath string) *LlmExtractor {
	return &LlmExtractor{pythonScriptPath: pythonScriptPath}
}

// Extract extracts facts from HistoryContent using the LangExtract Python script.
func (e *LlmExtractor) Extract(ctx context.Context, content *models.HistoryContent) ([]*models.Fact, error) {
	// 1. Construct the conversation text from the HistoryContent.
	var conversation string
	for _, part := range content.Content.Parts {
		conversation += fmt.Sprintf("%s: %s\n", content.Content.Role, part.Text)
	}

	// 2. Call the Python script to get the extracted facts.
	cmd := exec.CommandContext(ctx, "python3", e.pythonScriptPath, conversation)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run python script: %w, stderr: %s", err, errOut.String())
	}

	// 3. Parse the JSON output from the script.
	var response struct {
		Facts []map[string]string `json:"facts"`
	}

	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response from python script: %w", err)
	}

	// 4. Convert the extracted facts to Fact objects.
	var facts []*models.Fact
	for _, factMap := range response.Facts {
		for _, factContent := range factMap {
			facts = append(facts, &models.Fact{
				UserID:  content.User,
				Content: factContent,
				Source:  "HistoryContent",
			})
		}
	}

	return facts, nil
}
