package store

import (
	"Jarvis_2.0/backend/go/internal/database/neo4j"
	"Jarvis_2.0/backend/go/internal/models"
	"context"
	"fmt"
)

// GraphStore defines the interface for storing and retrieving graph data.
type GraphStore interface {
	AddRelations(ctx context.Context, relations []*models.Relation) error
	GetRelations(ctx context.Context, userID string) ([]*models.Relation, error)
}

// Neo4jStore is an implementation of the GraphStore interface that uses Neo4j as the backend.
type Neo4jStore struct {
	client *neo4j.Neo4jClient
}

// NewNeo4jStore creates a new Neo4jStore.
func NewNeo4jStore(client *neo4j.Neo4jClient) *Neo4jStore {
	return &Neo4jStore{client: client}
}

// AddRelations adds a list of relations to Neo4j.
func (s *Neo4jStore) AddRelations(ctx context.Context, relations []*models.Relation) error {
	for _, rel := range relations {
		query := `
		MERGE (source {name: $source_name, user_id: $user_id})
		MERGE (target {name: $target_name, user_id: $user_id})
		MERGE (source)-[:` + rel.Type + `]->(target)
		`
		params := map[string]interface{}{
			"source_name": rel.Source,
			"target_name": rel.Target,
			"user_id":     rel.UserID,
		}
		_, err := s.client.RunCypherQuery(ctx, query, params)
		if err != nil {
			return fmt.Errorf("failed to add relation to neo4j: %w", err)
		}
	}
	return nil
}

// GetRelations retrieves all relations for a given user from Neo4j.
func (s *Neo4jStore) GetRelations(ctx context.Context, userID string) ([]*models.Relation, error) {
	query := `
	MATCH (source {user_id: $user_id})-[r]->(target {user_id: $user_id})
	RETURN source.name AS source, type(r) AS type, target.name AS target
	`
	params := map[string]interface{}{
		"user_id": userID,
	}
	result, err := s.client.ReadCypherQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get relations from neo4j: %w", err)
	}

	var relations []*models.Relation
	for result.Next(ctx) {
		record := result.Record()
		source, _ := record.Get("source")
		target, _ := record.Get("target")
		relType, _ := record.Get("type")

		relations = append(relations, &models.Relation{
			Source: source.(string),
			Target: target.(string),
			Type:   relType.(string),
			UserID: userID,
		})
	}

	return relations, nil
}