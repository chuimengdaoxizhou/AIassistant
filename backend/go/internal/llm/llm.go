package llm

import (
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/internal/models"
	"context"
	"fmt"
	"github.com/google/generative-ai-go/genai"
)

// LLM 定义了所有大型语言模型客户端必须实现的通用接口。
type LLM interface {
	GenerateContent(ctx context.Context, req *models.GenerateContentRequest) (*models.GenerateContentResponse, error)
	GenerateContentStream(ctx context.Context, req *models.GenerateContentRequest) (<-chan *models.GenerateContentResponse, error)
}

// NewClient 是一个工厂函数，根据提供的配置创建并返回一个实现了 LLM 接口的客户端。
// 它现在接收一个工具声明列表，并将其注入到LLM客户端中，使模型能够感知并调用这些工具。
func NewClient(cfg config.LLMConfig, tools []*genai.FunctionDeclaration) (LLM, error) {
	switch cfg.Provider {
	case "gemini":
		// 假设配置文件中的第一个模型是我们要使用的模型。
		if len(cfg.Models) == 0 {
			return nil, fmt.Errorf("no model configured for gemini provider")
		}
		modelName := cfg.Models[0].Name
		apiKey := cfg.Models[0].APIKey
		return NewGemini(context.Background(), modelName, apiKey, tools)
	// case "openai":
	//     return NewOpenAI(...)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}