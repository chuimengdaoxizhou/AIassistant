package service

import (
	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/internal/memory/extractor"
	"Jarvis_2.0/backend/go/internal/memory/store"
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const defaultUpdateMemoryPrompt = `You are a smart memory manager which controls the memory of a system.
You can perform four operations: (1) add into the memory, (2) update the memory, (3) delete from the memory, and (4) no change.

Based on the above four operations, the memory will change.

Compare newly retrieved facts with the existing memory. For each new fact, decide whether to:
- ADD: Add it to the memory as a new element
- UPDATE: Update an existing memory element
- DELETE: Delete an existing memory element
- NONE: Make no change (if the fact is already present or irrelevant)

There are specific guidelines to select which operation to perform:

1. **Add**: If the retrieved facts contain new information not present in the memory, then you have to add it by generating a new ID in the id field.

2. **Update**: If the retrieved facts contain information that is already present in the memory but the information is totally different, then you have to update it. 
If the retrieved fact contains information that conveys the same thing as the elements present in the memory, then you have to keep the fact which has the most information. 
Example (a) -- if the memory contains "User likes to play cricket" and the retrieved fact is "Loves to play cricket with friends", then update the memory with the retrieved facts.
Example (b) -- if the memory contains "Likes cheese pizza" and the retrieved fact is "Loves cheese pizza", then you do not need to update it because they convey the same information.
If the direction is to update the memory, then you have to update it.
Please keep in mind while updating you have to keep the same ID.
Please note to return the IDs in the output from the input IDs only and do not generate any new ID.

3. **Delete**: If the retrieved facts contain information that contradicts the information present in the memory, then you have to delete it. Or if the direction is to delete the memory, then you have to delete it.
Please note to return the IDs in the output from the input IDs only and do not generate any new ID.

4. **No Change**: If the retrieved facts contain information that is already present in the memory, then you do not need to make any changes.
`

// MemoryAction represents an action to be taken on a memory.
type MemoryAction struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Event     string `json:"event"`
	OldMemory string `json:"old_memory,omitempty"`
}

// MemoryUpdateResponse represents the response from the LLM for a memory update.
type MemoryUpdateResponse struct {
	Memory []MemoryAction `json:"memory"`
}

// MemoryService provides the core memory functionality.
type MemoryService struct {
	factExtractor  extractor.Extractor
	graphExtractor *extractor.GraphExtractor
	vecStore       store.Store
	graphStore     store.GraphStore
	llm            llm.LLM
	logger         *logger.Logger
}

// NewMemoryService creates a new MemoryService.
func NewMemoryService(factExtractor extractor.Extractor, graphExtractor *extractor.GraphExtractor, vecStore store.Store, graphStore store.GraphStore, llm llm.LLM, logger *logger.Logger) *MemoryService {
	return &MemoryService{
		factExtractor:  factExtractor,
		graphExtractor: graphExtractor,
		vecStore:       vecStore,
		graphStore:     graphStore,
		llm:            llm,
		logger:         logger,
	}
}

// AddMemory adds a new memory from HistoryContent.
func (s *MemoryService) AddMemory(ctx context.Context, content *models.HistoryContent) error {
	// 1. Extract facts and relations from the content in parallel.
	var newFacts []*models.Fact
	var newRelations []*models.Relation
	var errFacts, errRelations error

	go func() {
		newFacts, errFacts = s.factExtractor.Extract(ctx, content)
	}()

	go func() {
		newRelations, errRelations = s.graphExtractor.Extract(ctx, content)
	}()

	if errFacts != nil {
		s.logger.WithError(models.ErrorInfo{Message: errFacts.Error()}).Error("failed to extract facts")
		return fmt.Errorf("failed to extract facts: %w", errFacts)
	}
	if errRelations != nil {
		s.logger.WithError(models.ErrorInfo{Message: errRelations.Error()}).Error("failed to extract relations")
		return fmt.Errorf("failed to extract relations: %w", errRelations)
	}

	// 2. Get existing memories and relations in parallel.
	var existingFacts []*models.Fact
	var existingRelations []*models.Relation
	var errExistingFacts, errExistingRelations error

	go func() {
		existingFacts, errExistingFacts = s.vecStore.GetFacts(ctx, content.User, "")
	}()

	go func() {
		existingRelations, errExistingRelations = s.graphStore.GetRelations(ctx, content.User)
	}()

	if errExistingFacts != nil {
		s.logger.WithError(models.ErrorInfo{Message: errExistingFacts.Error()}).Error("failed to get existing facts")
		return fmt.Errorf("failed to get existing facts: %w", errExistingFacts)
	}
	if errExistingRelations != nil {
		s.logger.WithError(models.ErrorInfo{Message: errExistingRelations.Error()}).Error("failed to get existing relations")
		return fmt.Errorf("failed to get existing relations: %w", errExistingRelations)
	}

	// 3. Process facts and relations.
	s.processFacts(ctx, content.User, existingFacts, newFacts)
	s.processRelations(ctx, existingRelations, newRelations)

	return nil
}

func (s *MemoryService) processFacts(ctx context.Context, userID string, existingFacts, newFacts []*models.Fact) {
	// Construct the prompt for the LLM.
	prompt, err := s.constructUpdatePrompt(existingFacts, newFacts)
	if err != nil {
		s.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to construct update prompt")
		return
	}

	// Call the LLM to get the memory actions.
	llmReq := &models.GenerateContentRequest{
		Content: []models.Content{
			{
				Parts: []*models.Part{
					{Text: prompt},
				},
			},
		},
	}
	llmResp, err := s.llm.GenerateContent(ctx, llmReq)
	if err != nil {
		s.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to generate content")
		return
	}

	// Parse the LLM response and execute the actions.
	var updateResp MemoryUpdateResponse
	jsonString := llmResp.Content[0].Parts[0].Text
	if err := json.Unmarshal([]byte(jsonString), &updateResp); err != nil {
		s.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to unmarshal response")
		return
	}

	for _, action := range updateResp.Memory {
		switch action.Event {
		case "ADD":
			newFact := &models.Fact{
				ID:        uuid.New().String(),
				UserID:    userID,
				Content:   action.Text,
				Source:    "HistoryContent",
				StartTime: time.Now(),
			}
			if err := s.vecStore.AddFact(ctx, newFact); err != nil {
				s.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to add fact")
			}
		case "UPDATE":
			var oldFact *models.Fact
			for _, f := range existingFacts {
				if f.ID == action.ID {
					oldFact = f
					break
				}
			}
			if oldFact != nil {
				now := time.Now()
				oldFact.EndTime = &now
				if err := s.vecStore.UpdateFact(ctx, oldFact); err != nil {
					s.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to update fact")
				}

				newFact := &models.Fact{
					ID:        uuid.New().String(),
					UserID:    userID,
					Content:   action.Text,
					Source:    "HistoryContent",
					StartTime: now,
				}
				if err := s.vecStore.AddFact(ctx, newFact); err != nil {
					s.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to add new fact after update")
				}
			}
		case "DELETE":
			if err := s.vecStore.DeleteFact(ctx, action.ID); err != nil {
				s.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("failed to delete fact")
			}
		case "NONE":
		}
	}
}

func (s *MemoryService) processRelations(ctx context.Context, existingRelations, newRelations []*models.Relation) {
	// TODO: Implement the logic for processing relations, similar to processFacts.
	// This will involve creating prompts for updating and deleting relations,
	// calling the LLM, and then executing the actions on the graphStore.
	s.graphStore.AddRelations(ctx, newRelations)
}

func (s *MemoryService) constructUpdatePrompt(existingFacts, newFacts []*models.Fact) (string, error) {
	existingFactsJSON, err := json.Marshal(existingFacts)
	if err != nil {
		return "", err
	}

	var newFactContents []string
	for _, f := range newFacts {
		newFactContents = append(newFactContents, f.Content)
	}

	newFactsJSON, err := json.Marshal(newFactContents)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf("%s\n\nExisting Memory:\n%s\n\nNew Facts:\n%s", defaultUpdateMemoryPrompt, string(existingFactsJSON), string(newFactsJSON))
	return prompt, nil
}
