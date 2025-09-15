package docstore

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"context"
	"fmt"
	"sync"
)

// InMemoryDocStore is a thread-safe, in-memory implementation of the DocStore interface.
// It uses a userID prefix on keys to simulate multi-tenancy.
type InMemoryDocStore struct {
	mu   sync.RWMutex
	docs map[string]*schema.Document
}

// NewInMemoryDocStore creates a new instance of InMemoryDocStore.
func NewInMemoryDocStore() *InMemoryDocStore {
	return &InMemoryDocStore{
		docs: make(map[string]*schema.Document),
	}
}

// tenantKey generates a key that is unique for a given user and document ID.
func (s *InMemoryDocStore) tenantKey(userID, docID string) string {
	return fmt.Sprintf("%s:%s", userID, docID)
}

// Add adds a map of documents to the store for a specific user.
func (s *InMemoryDocStore) Add(ctx context.Context, userID string, docs map[string]*schema.Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, doc := range docs {
		key := s.tenantKey(userID, id)
		s.docs[key] = doc
	}
	return nil
}

// Get retrieves a map of documents from the store by their IDs for a specific user.
func (s *InMemoryDocStore) Get(ctx context.Context, userID string, ids []string) (map[string]*schema.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*schema.Document)
	for _, id := range ids {
		key := s.tenantKey(userID, id)
		if doc, ok := s.docs[key]; ok {
			result[id] = doc // Return with the original ID, not the tenant key
		}
	}
	return result, nil
}

// Delete removes documents from the store by their IDs for a specific user.
func (s *InMemoryDocStore) Delete(ctx context.Context, userID string, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		key := s.tenantKey(userID, id)
		delete(s.docs, key)
	}
	return nil
}

// compile-time check to ensure InMemoryDocStore implements the DocStore interface
var _ interfaces.DocStore = (*InMemoryDocStore)(nil)
