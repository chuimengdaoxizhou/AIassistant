package embedding

import (
	"context"
	"fmt"
	openai "github.com/meguminnnnnnnnn/go-openai"
)

// OpenAIModel 是一个用于 OpenAI API 的 Embedding 模型客户端。
type OpenAIModel struct {
	client *openai.Client // OpenAI 客户端实例。
	model  string         // 要使用的模型名称。
}

// NewOpenAIModel 创建一个新的 OpenAIModel 客户端。
//
// 参数:
//
//	apiKey: OpenAI 的 API 密钥。
//	modelName: 要使用的模型名称。
//
// 返回值:
//
//	*OpenAIModel: 新创建的 OpenAIModel 客户端实例。
//	error: 如果创建客户端失败，则返回错误。
func NewOpenAIModel(apiKey, modelName string) (*OpenAIModel, error) {
	// 使用 API 密钥创建默认配置。
	config := openai.DefaultConfig(apiKey)
	// 使用配置创建新的 OpenAI 客户端。
	client := openai.NewClientWithConfig(config)
	return &OpenAIModel{client: client, model: modelName}, nil
}

// Embed 使用 OpenAI API 为单个文本生成嵌入向量。
//
// 参数:
//
//	ctx: 上下文，用于控制操作的生命周期。
//	text: 要生成嵌入向量的文本。
//
// 返回值:
//
//	[]float32: 生成的嵌入向量。
//	error: 如果生成嵌入向量失败，则返回错误。
func (m *OpenAIModel) Embed(ctx context.Context, text string) ([]float32, error) {
	// 调用 EmbedBatch 方法为单个文本生成嵌入向量。
	embeddings, err := m.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil // 返回第一个嵌入向量。
}

// EmbedBatch 使用 OpenAI API 为一批文本生成嵌入向量。
//
// 参数:
//
//	ctx: 上下文，用于控制操作的生命周期。
//	texts: 要生成嵌入向量的文本切片。
//
// 返回值:
//
//	[][]float32: 生成的嵌入向量切片。
//	error: 如果生成嵌入向量失败，则返回错误。
func (m *OpenAIModel) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {

	// 构建 OpenAI Embedding 请求。
	req := openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(m.model),
	}

	// 调用 OpenAI API 创建嵌入向量。
	resp, err := m.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// 检查是否返回了嵌入向量。
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// 将结果转换为 [][]float32 格式。
	embeddings := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		embeddings[i] = d.Embedding
	}

	return embeddings, nil
}
