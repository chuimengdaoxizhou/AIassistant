package llm

import (
	"Jarvis_2.0/backend/go/internal/models"
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// LLM 是大型语言模型的接口。
type LLM interface {
	GenerateContent(ctx context.Context, req *models.GenerateContentRequest) (*models.GenerateContentResponse, error)
	GenerateContentStream(ctx context.Context, req *models.GenerateContentRequest) (<-chan *models.GenerateContentResponse, error)
}

// NewLLM 根据提供商创建新的 LLM 客户端。
// 它接受一个可选的 []*mcp.Tool 配置，并会根据 provider 自动转换为相应的格式。
func NewLLM(provider, model, apiKey, baseURL string, mcpTools []*mcp.Tool) (LLM, error) {
	switch provider {
	case "gemini":
		geminiTools, err := ConvertMCPToolsToGemini(mcpTools)
		if err != nil {
			return nil, fmt.Errorf("failed to convert tools for gemini: %w", err)
		}
		return NewGemini(context.Background(), model, apiKey, geminiTools)
	case "openai":
		openaiTools, err := ConvertMCPToolsToOpenAI(mcpTools)
		if err != nil {
			return nil, fmt.Errorf("failed to convert tools for openai: %w", err)
		}
		return NewOpenAI(model, apiKey, openaiTools)
	case "huggingface":
		return NewHuggingFace(model, apiKey, baseURL)
	case "ollama":
		return NewOllama(model, baseURL)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
