package vectorstore

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"context"
	"fmt"
	"strings"

	"Jarvis_2.0/backend/go/internal/database/milvus"
	"Jarvis_2.0/backend/go/pkg/logger"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	// Schema fields for the Milvus collection that we want to filter on or output.
	FieldID        = "id"
	FieldEmbedding = "embedding"
	FieldDocType   = "doc_type"
	FieldFileName  = "file_name"
	FieldPageLabel = "page_label"
	FieldUserID    = "user_id"
	FieldFolderID  = "folder_id"
)

// MilvusStore is an adapter for the existing Milvus client to implement the VectorStore interface.
// It uses the underlying milvus-sdk-go client to leverage advanced features like metadata filtering.
type MilvusStore struct {
	log        logger.Logger
	client     client.Client // The raw client from the existing MilvusClient wrapper
	collection string
}

// NewMilvusStore creates a new MilvusStore adapter.
// It takes the project's existing MilvusClient wrapper and the name of the collection to use.
func NewMilvusStore(milvusClient *milvus.MilvusClient, collectionName string, log logger.Logger) (interfaces.VectorStore, error) {
	if milvusClient == nil || milvusClient.Client == nil {
		return nil, fmt.Errorf("milvus client is not initialized")
	}
	return &MilvusStore{
		log:        log,
		client:     milvusClient.Client,
		collection: collectionName,
	}, nil
}

// Add inserts a list of documents into the Milvus collection.
// It extracts embeddings and metadata from the documents and stores them in separate columns.
func (s *MilvusStore) Add(ctx context.Context, docs []*schema.Document) error {
	if len(docs) == 0 {
		return nil
	}

	// Prepare columns from the documents
	ids := make([]string, len(docs))
	embeddings := make([][]float32, len(docs))
	userIDs := make([]string, len(docs))
	folderIDs := make([]string, len(docs)) // Assuming folderID is stored as string

	dim := 0
	for i, doc := range docs {
		ids[i] = doc.ID
		embeddings[i] = doc.Embedding
		if len(doc.Embedding) > dim {
			dim = len(doc.Embedding)
		}

		if uid, ok := doc.Metadata[FieldUserID].(string); ok {
			userIDs[i] = uid
		}
		if fid, ok := doc.Metadata[FieldFolderID].(string); ok {
			folderIDs[i] = fid
		}
	}

	// Create Milvus columns
	idCol := entity.NewColumnVarChar(FieldID, ids)
	embeddingCol := entity.NewColumnFloatVector(FieldEmbedding, dim, embeddings)
	userIDCol := entity.NewColumnVarChar(FieldUserID, userIDs)
	folderIDCol := entity.NewColumnVarChar(FieldFolderID, folderIDs)

	s.log.Info(fmt.Sprintf("Inserting %d documents into Milvus collection: %s", len(docs), s.collection))
	_, err := s.client.Insert(ctx, s.collection, "" /* default partition */, idCol, embeddingCol, userIDCol, folderIDCol)
	if err != nil {
		s.log.Error(fmt.Sprintf("Failed to insert data into Milvus: %v", err))
		return fmt.Errorf("failed to insert data into Milvus: %w", err)
	}

	return nil
}

// Query performs a vector search in the Milvus collection with optional metadata filtering.
func (s *MilvusStore) Query(ctx context.Context, embedding []float32, topK int, filters map[string]interface{}) ([]*schema.Document, error) {
	filterExpr := s.buildFilterExpression(filters)

	searchParams, _ := entity.NewIndexIvfFlatSearchParam(10)
	outputFields := []string{FieldID, FieldUserID, FieldFolderID} // Adjust output fields as needed

	s.log.Info(fmt.Sprintf("Querying Milvus collection '%s' with filter: '%s'", s.collection, filterExpr))

	searchResults, err := s.client.Search(
		ctx, s.collection, []string{}, filterExpr, outputFields,
		[]entity.Vector{entity.FloatVector(embedding)},
		FieldEmbedding, entity.L2, topK, searchParams,
	)
	if err != nil {
		s.log.Error(fmt.Sprintf("Failed to search in Milvus: %v", err))
		return nil, fmt.Errorf("failed to search in Milvus: %w", err)
	}

	var results []*schema.Document
	for _, res := range searchResults {
		findColumn := func(name string) entity.Column {
			for _, field := range res.Fields {
				if field.Name() == name {
					return field
				}
			}
			return nil
		}

		idCol, ok := findColumn(FieldID).(*entity.ColumnVarChar)
		if !ok {
			s.log.Warn("Search result is missing ID field or has wrong type, skipping.")
			continue
		}
		idData := idCol.Data()

		var userIDData, folderIDData []string
		if userIDCol, ok := findColumn(FieldUserID).(*entity.ColumnVarChar); ok {
			userIDData = userIDCol.Data()
		}
		if folderIDCol, ok := findColumn(FieldFolderID).(*entity.ColumnVarChar); ok {
			folderIDData = folderIDCol.Data()
		}

		for i := 0; i < res.ResultCount; i++ {
			doc := &schema.Document{
				ID:       idData[i],
				Metadata: map[string]interface{}{"score": res.Scores[i]},
			}
			if userIDData != nil {
				doc.Metadata[FieldUserID] = userIDData[i]
			}
			if folderIDData != nil {
				doc.Metadata[FieldFolderID] = folderIDData[i]
			}
			results = append(results, doc)
		}
	}

	return results, nil
}

// buildFilterExpression creates a Milvus filter expression string from a map.
func (s *MilvusStore) buildFilterExpression(filters map[string]interface{}) string {
	if len(filters) == 0 {
		return ""
	}

	var conditions []string
	for key, value := range filters {
		if v, ok := value.(string); ok {
			conditions = append(conditions, fmt.Sprintf(`%s == "%s"`, key, v))
		}
	}
	return strings.Join(conditions, " and ")
}

// compile-time check to ensure MilvusStore implements the VectorStore interface
var _ interfaces.VectorStore = (*MilvusStore)(nil)
