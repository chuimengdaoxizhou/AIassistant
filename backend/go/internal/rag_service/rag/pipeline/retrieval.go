package pipeline

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/storages/vectorstore"
	"context"
	"fmt"

	"Jarvis_2.0/backend/go/pkg/logger"
)

// RetrievalPipeline orchestrates the process of retrieving relevant documents for a given query.
type RetrievalPipeline struct {
	embedder    interfaces.EmbeddingModel
	vectorStore interfaces.VectorStore
	docStore    interfaces.DocStore
	reranker    interfaces.Reranker // Optional component to rerank results
	log         logger.Logger
}

// NewRetrievalPipeline creates a new RetrievalPipeline.
// The reranker is optional and can be nil.
func NewRetrievalPipeline(
	embedder interfaces.EmbeddingModel,
	vectorStore interfaces.VectorStore,
	docStore interfaces.DocStore,
	reranker interfaces.Reranker,
	log logger.Logger,
) *RetrievalPipeline {
	return &RetrievalPipeline{
		embedder:    embedder,
		vectorStore: vectorStore,
		docStore:    docStore,
		reranker:    reranker,
		log:         log,
	}
}

// Run executes the retrieval pipeline with multi-tenancy filters.
func (p *RetrievalPipeline) Run(ctx context.Context, query, userID string, folderIDs []string, topK int) ([]*schema.Document, error) {
	p.log.Info(fmt.Sprintf("Starting retrieval for query: '%s' for user: %s", query, userID))

	// 1. Embed the query
	queryEmbeddings, err := p.embedder.Embed(ctx, []string{query})
	if err != nil || len(queryEmbeddings) == 0 {
		p.log.Error(fmt.Sprintf("Failed to embed query: %v", err))
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	p.log.Info("Successfully embedded query")

	// 2. Construct filters for multi-tenancy
	filters := map[string]interface{}{
		vectorstore.FieldUserID: userID,
	}
	if len(folderIDs) > 0 {
		// This requires the buildFilterExpression to handle IN clauses
		// For simplicity in this example, we will filter by the first folder ID.
		// A production implementation should handle multiple folders.
		filters[vectorstore.FieldFolderID] = folderIDs[0]
	}

	// 3. Query the VectorStore to get document IDs and preliminary metadata
	retrievedDocs, err := p.vectorStore.Query(ctx, queryEmbeddings[0], topK, filters)
	if err != nil {
		p.log.Error(fmt.Sprintf("Failed to query vector store: %v", err))
		return nil, err
	}
	if len(retrievedDocs) == 0 {
		p.log.Info("No documents found in vector store for the given query.")
		return []*schema.Document{}, nil
	}
	p.log.Info(fmt.Sprintf("Retrieved %d document candidates from vector store", len(retrievedDocs)))

	// 4. Enrich the results with full text from the DocStore
	ids := make([]string, len(retrievedDocs))
	for i, doc := range retrievedDocs {
		ids[i] = doc.ID
	}

	fullDocsMap, err := p.docStore.Get(ctx, userID, ids)
	if err != nil {
		p.log.Error(fmt.Sprintf("Failed to get full documents from doc store: %v", err))
		return nil, err
	}

	// 5. Combine information into a final list
	finalDocs := make([]*schema.Document, 0, len(retrievedDocs))
	for _, retrievedDoc := range retrievedDocs {
		if fullDoc, ok := fullDocsMap[retrievedDoc.ID]; ok {
			fullDoc.Metadata = retrievedDoc.Metadata
			finalDocs = append(finalDocs, fullDoc)
		} else {
			p.log.Warn(fmt.Sprintf("Could not find full document for ID: %s in doc store for user %s", retrievedDoc.ID, userID))
		}
	}

	// 6. Rerank the results if a reranker is configured
	if p.reranker != nil {
		p.log.Info("Reranking documents...")
		rerankedDocs, err := p.reranker.Rerank(ctx, query, finalDocs)
		if err != nil {
			p.log.Warn(fmt.Sprintf("Reranker failed: %v. Returning documents without reranking.", err))
		} else {
			finalDocs = rerankedDocs
			p.log.Info("Successfully reranked documents.")
		}
	}

	p.log.Info(fmt.Sprintf("Successfully retrieved and enriched %d documents", len(finalDocs)))
	return finalDocs, nil
}
