package embedding

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	ollama "github.com/ollama/ollama/api"
)

// OllamaModel 是一个用于 Ollama API 的 Embedding 模型客户端。
type OllamaModel struct {
	client *ollama.Client // Ollama 客户端实例。
	model  string         // 要使用的模型名称。
}

// NewOllamaModel 创建一个新的 OllamaModel 客户端。
//
// 参数:
//
//	model: 要使用的模型名称。
//	baseURL: Ollama 服务的基准 URL。如果为空，则默认为 "http://localhost:11434"。
//
// 返回值:
//
//	*OllamaModel: 新创建的 OllamaModel 客户端实例。
//	error: 如果基准 URL 无效，则返回错误。
func NewOllamaModel(model, baseURL string) (*OllamaModel, error) {
	// 如果 baseURL 为空，则使用默认地址。
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// 将字符串 URL 转换为 *url.URL。
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// 创建一个带有超时设置的 HTTP 客户端。
	hc := &http.Client{
		Timeout: 120 * time.Second,
	}

	// 创建 Ollama 客户端。
	client := ollama.NewClient(parsedURL, hc)

	return &OllamaModel{client: client, model: model}, nil
}

// Embed 为单个文本生成嵌入向量。
//
// 参数:
//
//	ctx: 上下文，用于控制操作的生命周期。
//	text: 要生成嵌入向量的文本。
//
// 返回值:
//
//	[]float32: 生成的嵌入向量。
//	error: 如果从 Ollama 获取嵌入向量失败，则返回错误。
func (m *OllamaModel) Embed(ctx context.Context, text string) ([]float32, error) {
	// 调用 Ollama 客户端的 Embed 方法生成嵌入向量。
	resp, err := m.client.Embed(ctx, &ollama.EmbedRequest{
		Model: m.model,
		Input: text,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get embeddings from ollama: %w", err)
	}

	// 返回第一个嵌入向量（单个文本输入）。
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return resp.Embeddings[0], nil
}

// EmbedBatch 为一批文本生成嵌入向量。
//
// 参数:
//
//	ctx: 上下文，用于控制操作的生命周期。
//	texts: 要生成嵌入向量的文本切片。
//
// 返回值:
//
//	[][]float32: 生成的嵌入向量切片。
//	error: 如果从 Ollama 获取批量嵌入向量失败，则返回错误。
func (m *OllamaModel) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {

	// 使用 Ollama 的批量嵌入功能。
	resp, err := m.client.Embed(ctx, &ollama.EmbedRequest{
		Model: m.model,
		Input: texts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get batch embeddings from ollama: %w", err)
	}

	return resp.Embeddings, nil
}
