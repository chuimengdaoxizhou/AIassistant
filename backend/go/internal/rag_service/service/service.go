package service

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/dal"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/embeddings"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/llms"
	loaders2 "Jarvis_2.0/backend/go/internal/rag_service/rag/loaders"
	pipeline2 "Jarvis_2.0/backend/go/internal/rag_service/rag/pipeline"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/rerankers"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/splitters"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/storages/docstore"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/storages/vectorstore"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	ragv1 "Jarvis_2.0/api/proto/v1/rag"
	"Jarvis_2.0/backend/go/internal/database/milvus"
	"Jarvis_2.0/backend/go/internal/embedding"
	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the RagServiceServer interface generated from the proto.
type Server struct {
	ragv1.UnimplementedRagServiceServer

	log                   logger.Logger
	folderDal             *dal.FolderDAL
	milvusClient          *milvus.MilvusClient
	geminiEmbeddingClient *embedding.GoogleModel
	geminiLLMClient       *llm.Gemini
	collectionName        string
	cohereAPIKey          string // Assuming cohere API key is passed in config
}

// NewServer creates a new gRPC server for the RAG service.
func NewServer(
	log logger.Logger,
	folderDal *dal.FolderDAL,
	milvusClient *milvus.MilvusClient,
	geminiEmbeddingClient *embedding.GoogleModel,
	geminiLLMClient *llm.Gemini,
	collectionName string,
	cohereAPIKey string,
) *Server {
	return &Server{
		log:                   log,
		folderDal:             folderDal,
		milvusClient:          milvusClient,
		geminiEmbeddingClient: geminiEmbeddingClient,
		geminiLLMClient:       geminiLLMClient,
		collectionName:        collectionName,
		cohereAPIKey:          cohereAPIKey,
	}
}

// Query handles a user's query by running the full retrieval and QA pipeline.
func (s *Server) Query(ctx context.Context, req *ragv1.QueryRequest) (*ragv1.QueryResponse, error) {
	s.log.Info(fmt.Sprintf("Received Query request for user %s", req.GetUserId()))

	docStore := docstore.NewInMemoryDocStore()
	vectorStore, err := vectorstore.NewMilvusStore(s.milvusClient, s.collectionName, s.log)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create vector store: %v", err)
	}
	embeddingAdapter := embeddings.NewGenaiAdapter(s.geminiEmbeddingClient)
	llmAdapter := llms.NewGeminiAdapter(s.geminiLLMClient)
	reranker := rerankers.NewCohereReranker(s.cohereAPIKey, "rerank-english-v2.0", 10)

	retrievalPipeline := pipeline2.NewRetrievalPipeline(embeddingAdapter, vectorStore, docStore, reranker, s.log)
	qaPipeline := pipeline2.NewQAPipeline(llmAdapter, s.log)

	retrievedDocs, err := retrievalPipeline.Run(ctx, req.GetQuery(), req.GetUserId(), req.GetFolderIds(), 10)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "retrieval pipeline failed: %v", err)
	}

	answer, err := qaPipeline.Run(ctx, req.GetQuery(), retrievedDocs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "QA pipeline failed: %v", err)
	}

	resp := &ragv1.QueryResponse{Answer: answer}
	for _, doc := range retrievedDocs {
		metadata := make(map[string]string)
		for k, v := range doc.Metadata {
			metadata[k] = fmt.Sprintf("%v", v)
		}
		resp.Sources = append(resp.Sources, &ragv1.RetrievedDocument{
			Id: doc.ID, Text: doc.Text, Metadata: metadata,
		})
	}

	return resp, nil
}

// Index handles the document indexing process.
func (s *Server) Index(req *ragv1.IndexRequest, stream ragv1.RagService_IndexServer) error {
	ctx := stream.Context()
	s.log.Info(fmt.Sprintf("Received Index request for user %s, folder %s", req.GetUserId(), req.GetFolderId()))

	docStore := docstore.NewInMemoryDocStore()
	vectorStore, err := vectorstore.NewMilvusStore(s.milvusClient, s.collectionName, s.log)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create vector store: %v", err)
	}
	splitter, err := splitters.NewTokenSplitter(1024, 256)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create splitter: %v", err)
	}
	embeddingAdapter := embeddings.NewGenaiAdapter(s.geminiEmbeddingClient)

	indexingPipeline := pipeline2.NewIndexingPipeline(splitter, embeddingAdapter, docStore, vectorStore, s.log)

	for _, path := range req.GetPaths() {
		var loader interfaces.Loader
		ext := strings.ToLower(filepath.Ext(path))

		switch {
		case strings.HasPrefix(path, "http"):
			loader = loaders2.NewWebLoader()
		case ext == ".pdf":
			loader = loaders2.NewPdfLoader()
		case ext == ".docx":
			loader = loaders2.NewDocxLoader()
		case ext == ".xlsx":
			loader = loaders2.NewXlsxLoader()
		case ext == ".md":
			loader = loaders2.NewMarkdownLoader()
		case ext == ".txt":
			loader = loaders2.NewTxtLoader()
		default:
			s.log.Warn(fmt.Sprintf("Unsupported file type '%s' for path %s. Defaulting to text loader.", ext, path))
			loader = loaders2.NewTxtLoader()
		}

		progressChan := make(chan *ragv1.IndexResponse)
		go func() {
			if err := indexingPipeline.Run(ctx, loader, path, req.GetUserId(), req.GetFolderId(), progressChan); err != nil {
				s.log.Error(fmt.Sprintf("Indexing pipeline failed for path %s: %v", path, err))
			}
		}()

		for progress := range progressChan {
			if err := stream.Send(progress); err != nil {
				s.log.Error(fmt.Sprintf("Failed to send progress update to client: %v", err))
				return err
			}
		}
	}

	s.log.Info("Finished processing all paths for Index request.")
	return nil
}

// CreateFolder creates a new folder for a user.
func (s *Server) CreateFolder(ctx context.Context, req *ragv1.CreateFolderRequest) (*ragv1.FolderResponse, error) {
	s.log.Info(fmt.Sprintf("Received CreateFolder request for user %s, folder %s", req.GetUserId(), req.GetFolderName()))

	folder, err := s.folderDal.CreateFolder(ctx, req.GetUserId(), req.GetFolderName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create folder: %v", err)
	}

	return &ragv1.FolderResponse{
		Folder: &ragv1.Folder{
			Id:        strconv.FormatUint(uint64(folder.ID), 10),
			Name:      folder.Name,
			CreatedAt: folder.CreatedAt.String(),
		},
	}, nil
}

// ListFolders lists all folders for a user.
func (s *Server) ListFolders(ctx context.Context, req *ragv1.ListFoldersRequest) (*ragv1.ListFoldersResponse, error) {
	s.log.Info(fmt.Sprintf("Received ListFolders request for user %s", req.GetUserId()))

	folders, err := s.folderDal.ListFoldersByUser(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list folders: %v", err)
	}

	resp := &ragv1.ListFoldersResponse{
		Folders: make([]*ragv1.Folder, 0, len(folders)),
	}

	for _, folder := range folders {
		resp.Folders = append(resp.Folders, &ragv1.Folder{
			Id:        strconv.FormatUint(uint64(folder.ID), 10),
			Name:      folder.Name,
			CreatedAt: folder.CreatedAt.String(),
		})
	}

	return resp, nil
}
