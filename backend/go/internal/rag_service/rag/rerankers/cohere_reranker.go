package rerankers

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

const cohereRerankURL = "https://api.cohere.ai/v1/rerank"

// CohereReranker implements the Reranker interface using the Cohere Rerank API.
type CohereReranker struct {
	apiKey     string
	httpClient *http.Client
	model      string
	topN       int
}

// cohereRerankRequest defines the request body for the Cohere Rerank API.
type cohereRerankRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	TopN            int      `json:"top_n"`
	ReturnDocuments bool     `json:"return_documents"`
}

// cohereRerankResponse defines the structure of a result from the Cohere Rerank API.
type cohereRerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type cohereRerankResponse struct {
	Results []cohereRerankResult `json:"results"`
}

// NewCohereReranker creates a new CohereReranker.
func NewCohereReranker(apiKey, model string, topN int) *CohereReranker {
	return &CohereReranker{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		model:      model,
		topN:       topN,
	}
}

// Rerank re-orders a list of documents based on relevance scores from the Cohere API.
func (r *CohereReranker) Rerank(ctx context.Context, query string, docs []*schema.Document) ([]*schema.Document, error) {
	if len(docs) == 0 {
		return docs, nil
	}

	// 1. Prepare the request for Cohere's API
	docTexts := make([]string, len(docs))
	for i, doc := range docs {
		docTexts[i] = doc.Text
	}

	reqBody := cohereRerankRequest{
		Model:           r.model,
		Query:           query,
		Documents:       docTexts,
		TopN:            r.topN,
		ReturnDocuments: false,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cohere request: %w", err)
	}

	// 2. Make the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", cohereRerankURL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create cohere request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call cohere api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cohere api returned non-200 status: %s", resp.Status)
	}

	// 3. Parse the response and re-order the documents
	var cohereResp cohereRerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&cohereResp); err != nil {
		return nil, fmt.Errorf("failed to decode cohere response: %w", err)
	}

	// Create a new slice for the reranked documents
	rerankedDocs := make([]*schema.Document, 0, len(cohereResp.Results))
	for _, result := range cohereResp.Results {
		if result.Index < len(docs) {
			// Get the original document
			originalDoc := docs[result.Index]
			// Update its score
			originalDoc.Metadata["score"] = result.RelevanceScore
			rerankedDocs = append(rerankedDocs, originalDoc)
		}
	}

	// Sort by the new score in descending order
	sort.Slice(rerankedDocs, func(i, j int) bool {
		iScore, _ := rerankedDocs[i].Metadata["score"].(float64)
		jScore, _ := rerankedDocs[j].Metadata["score"].(float64)
		return iScore > jScore
	})

	return rerankedDocs, nil
}

// compile-time check to ensure CohereReranker implements the Reranker interface
var _ interfaces.Reranker = (*CohereReranker)(nil)
