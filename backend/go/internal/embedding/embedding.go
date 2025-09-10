package embedding

import (
	"fmt"
)

// NewEmdModel 根据指定的提供商、模型、API 密钥和基础 URL 创建并返回一个新的 Embedding 模型实例。
//
// 参数:
//
//	provider: Embedding 模型的提供商 (例如: "gemini", "openai", "huggingface", "ollama")。
//	model: 要使用的模型名称。
//	apiKey: 模型的 API 密钥。
//	baseURL: 模型的服务基础 URL (可选，某些提供商可能不需要)。
//
// 返回值:
//
//	EmbeddingModel: 新创建的 Embedding 模型实例。
//	error: 如果提供商不支持或模型初始化失败，则返回错误。
func NewEmdModel(provider, model, apiKey, baseURL string) (Embedding, error) {
	// 根据提供商类型创建相应的 Embedding 模型实例。
	switch provider {
	case "gemini":
		return NewGoogleModel(model, apiKey)
	case "openai":
		return NewOpenAIModel(model, apiKey)
	case "huggingface":
		return NewHuggingFaceModel(model, apiKey, baseURL)
	case "ollama":
		return NewOllamaModel(model, baseURL)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider) // 如果提供商不支持，返回错误。
	}
}
