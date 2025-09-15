package pipeline

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/storages/vectorstore"
	"context"
	"fmt"

	ragv1 "Jarvis_2.0/api/proto/v1/rag"
	"Jarvis_2.0/backend/go/pkg/logger"
	"golang.org/x/sync/errgroup"
)

// IndexingPipeline orchestrates the process of loading, splitting, embedding, and storing documents.
type IndexingPipeline struct {
	splitter    interfaces.Splitter
	embedder    interfaces.EmbeddingModel
	docStore    interfaces.DocStore
	vectorStore interfaces.VectorStore
	log         logger.Logger
}

// NewIndexingPipeline creates a new IndexingPipeline.
func NewIndexingPipeline(
	splitter interfaces.Splitter,
	embedder interfaces.EmbeddingModel,
	docStore interfaces.DocStore,
	vectorStore interfaces.VectorStore,
	log logger.Logger,
) *IndexingPipeline {
	return &IndexingPipeline{
		splitter:    splitter,
		embedder:    embedder,
		docStore:    docStore,
		vectorStore: vectorStore,
		log:         log,
	}
}

// Run executes the entire indexing pipeline for a given data source and streams progress updates.
func (p *IndexingPipeline) Run(ctx context.Context, loader interfaces.Loader, path, userID, folderID string, progressChan chan<- *ragv1.IndexResponse) error {
	defer close(progressChan)

	p.log.Info(fmt.Sprintf("Starting indexing for path: %s, user: %s, folder: %s", path, userID, folderID))
	progressChan <- &ragv1.IndexResponse{Message: fmt.Sprintf("Starting indexing for: %s", path)}

	// 1. Load the data
	initialDocs, err := loader.Load(ctx, path)
	if err != nil {
		p.log.Error(fmt.Sprintf("Failed to load data: %v", err))
		return err
	}
	progressChan <- &ragv1.IndexResponse{Message: fmt.Sprintf("Loaded %d initial documents", len(initialDocs)), Progress: 10}

	// 2. Split documents into chunks
	chunks, err := p.splitter.Split(ctx, initialDocs)
	if err != nil {
		p.log.Error(fmt.Sprintf("Failed to split documents: %v", err))
		return err
	}
	progressChan <- &ragv1.IndexResponse{Message: fmt.Sprintf("Split into %d chunks", len(chunks)), Progress: 25}

	// 3. Add multi-tenancy metadata to each chunk
	for _, chunk := range chunks {
		if chunk.Metadata == nil {
			chunk.Metadata = make(map[string]interface{})
		}
		chunk.Metadata[vectorstore.FieldUserID] = userID
		chunk.Metadata[vectorstore.FieldFolderID] = folderID
	}

	// 4. Embed the chunks
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Text
	}
	embeddings, err := p.embedder.Embed(ctx, texts)
	if err != nil {
		p.log.Error(fmt.Sprintf("Failed to embed chunks: %v", err))
		return err
	}
	for i, chunk := range chunks {
		chunk.Embedding = embeddings[i]
	}
	progressChan <- &ragv1.IndexResponse{Message: "Successfully embedded all chunks", Progress: 60}

	// 5. Store the chunks concurrently
	eg, gCtx := errgroup.WithContext(ctx)

	// Goroutine for DocStore
	eg.Go(func() error {
		chunkMap := make(map[string]*schema.Document, len(chunks))
		for _, chunk := range chunks {
			chunkMap[chunk.ID] = chunk
		}
		p.log.Info("Adding chunks to DocStore...")
		if err := p.docStore.Add(gCtx, userID, chunkMap); err != nil {
			p.log.Error(fmt.Sprintf("Failed to add chunks to DocStore: %v", err))
			return err
		}
		progressChan <- &ragv1.IndexResponse{Message: "Successfully added chunks to DocStore", Progress: 80}
		return nil
	})

	// Goroutine for VectorStore
	eg.Go(func() error {
		p.log.Info("Adding chunks to VectorStore...")
		if err := p.vectorStore.Add(gCtx, chunks); err != nil {
			p.log.Error(fmt.Sprintf("Failed to add chunks to VectorStore: %v", err))
			return err
		}
		progressChan <- &ragv1.IndexResponse{Message: "Successfully added chunks to VectorStore", Progress: 95}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	p.log.Info(fmt.Sprintf("Successfully finished indexing for: %s", path))
	progressChan <- &ragv1.IndexResponse{Message: fmt.Sprintf("Successfully finished indexing for: %s", path), Progress: 100}
	return nil
}
