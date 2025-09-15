package pipeline

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"context"
	"fmt"
	"strings"

	"Jarvis_2.0/backend/go/pkg/logger"
)

// QAPipeline is responsible for generating an answer based on a query and retrieved documents.
type QAPipeline struct {
	llm interfaces.LLM
	log logger.Logger
}

// NewQAPipeline creates a new QAPipeline.
func NewQAPipeline(llm interfaces.LLM, log logger.Logger) *QAPipeline {
	return &QAPipeline{
		llm: llm,
		log: log,
	}
}

// Run takes a query and a list of documents, builds a prompt, and calls the LLM to generate an answer.
func (p *QAPipeline) Run(ctx context.Context, query string, documents []*schema.Document) (string, error) {
	p.log.Info(fmt.Sprintf("Building prompt for query: '%s' with %d documents", query, len(documents)))

	// 1. Build the prompt
	prompt := p.buildPrompt(query, documents)

	// 2. Call the LLM to generate the answer
	p.log.Info("Sending prompt to LLM to generate answer...")
	answer, err := p.llm.Generate(ctx, prompt)
	if err != nil {
		p.log.Error(fmt.Sprintf("LLM failed to generate answer: %v", err))
		return "", err
	}

	p.log.Info("Successfully generated answer from LLM.")
	return answer, nil
}

// buildPrompt constructs a prompt string from a query and a list of context documents.
func (p *QAPipeline) buildPrompt(query string, documents []*schema.Document) string {
	var sb strings.Builder

	sb.WriteString("Based on the following context, please answer the question.\n\nContext:\n")

	for i, doc := range documents {
		sb.WriteString("---\n")
		sb.WriteString(fmt.Sprintf("Context %d:\n%s\n", i+1, doc.Text))
	}

	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("Question: %s", query))

	return sb.String()
}
