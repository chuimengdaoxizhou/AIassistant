package store

import (
	"Jarvis_2.0/backend/go/internal/database/milvus"
	"Jarvis_2.0/backend/go/internal/embedding"
	"Jarvis_2.0/backend/go/internal/models"
	"context"
	"fmt"
	"time"
)

// MilvusStore is an implementation of the Store interface that uses Milvus as the backend.
type MilvusStore struct {
	client   *milvus.MilvusClient
	embedder embedding.Embedding
	collName string
}

// NewMilvusStore creates a new MilvusStore.
func NewMilvusStore(client *milvus.MilvusClient, embedder embedding.Embedding, collName string) *MilvusStore {
	return &MilvusStore{
		client:   client,
		embedder: embedder,
		collName: collName,
	}
}

// GetFacts retrieves facts from Milvus.
func (s *MilvusStore) GetFacts(ctx context.Context, userID string, query string) ([]*models.Fact, error) {
	// Generate embedding for the query
	queryVector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	// Search for similar vectors
	searchResult, err := s.client.Search(ctx, s.collName, 10, queryVector)
	if err != nil {
		return nil, fmt.Errorf("failed to search in Milvus: %w", err)
	}

	var facts []*models.Fact
	for _, result := range searchResult {
		for i := 0; i < result.ResultCount; i++ {
			id, _ := result.IDs.GetAsString(i)
			// Assuming the order of fields in the output is the same as requested
			userID, _ := result.Fields[1].GetAsString(i)
			content, _ := result.Fields[2].GetAsString(i)
			source, _ := result.Fields[3].GetAsString(i)
			startTimeInt, _ := result.Fields[4].GetAsInt64(i)
			endTimeInt, _ := result.Fields[5].GetAsInt64(i)

			startTime := time.Unix(startTimeInt, 0)
			var endTime *time.Time
			if endTimeInt != 0 {
				t := time.Unix(endTimeInt, 0)
				endTime = &t
			}

			facts = append(facts, &models.Fact{
				ID:        id,
				UserID:    userID,
				Content:   content,
				Source:    source,
				StartTime: startTime,
				EndTime:   endTime,
			})
		}
	}

	return facts, nil
}

// AddFact adds a fact to Milvus.
func (s *MilvusStore) AddFact(ctx context.Context, fact *models.Fact) error {
	// Generate embedding for the fact's content
	vector, err := s.embedder.Embed(ctx, fact.Content)
	if err != nil {
		return err
	}

	return s.client.Insert(ctx, s.collName, "", fact.ID, vector)
}

// UpdateFact updates a fact in Milvus.
func (s *MilvusStore) UpdateFact(ctx context.Context, fact *models.Fact) error {
	// For Milvus, update is basically a delete and insert.
	// First, delete the old fact.
	if err := s.DeleteFact(ctx, fact.ID); err != nil {
		return err
	}

	// Then, insert the new fact.
	return s.AddFact(ctx, fact)
}

// DeleteFact deletes a fact from Milvus.
func (s *MilvusStore) DeleteFact(ctx context.Context, factID string) error {
	return s.client.Delete(ctx, s.collName, "", factID)
}
