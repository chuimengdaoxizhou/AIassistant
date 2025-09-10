package embedding

import (
	"context"
	"github.com/google/generative-ai-go/genai" // 正确的导入路径
	"google.golang.org/api/option"
)

// GoogleModel 是一个用于 Google GenAI Embedding API 的客户端。
type GoogleModel struct {
	// 结构体现在持有一个 EmbeddingModel 实例。
	model *genai.EmbeddingModel
}

// NewGoogleModel 使用正确的库进行初始化，创建并返回一个新的 GoogleModel 客户端实例。
//
// 参数:
//
//	apiKey: Google GenAI 的 API 密钥。
//	modelName: 要使用的 Embedding 模型名称。
//
// 返回值:
//
//	*GoogleModel: 新创建的 GoogleModel 客户端实例。
//	error: 如果无法创建 GenAI 客户端，则返回错误。
func NewGoogleModel(apiKey string, modelName string) (*GoogleModel, error) {
	ctx := context.Background() // 创建一个背景上下文。
	// 1. 使用 genai.NewClient 初始化客户端。
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	// 2. 获取指定的 embedding 模型。
	return &GoogleModel{
		model: client.EmbeddingModel(modelName),
	}, nil
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
//	error: 如果生成嵌入向量失败，则返回错误。
func (m *GoogleModel) Embed(ctx context.Context, text string) ([]float32, error) {
	// 调用模型的 EmbedContent 方法生成嵌入向量。
	res, err := m.model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}
	return res.Embedding.Values, nil // 返回嵌入向量的值。
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
//	error: 如果生成嵌入向量失败，则返回错误。
func (m *GoogleModel) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {

	// 创建一个新的批量嵌入请求。
	batch := m.model.NewBatch()
	// 将所有文本添加到批量请求中。
	for _, text := range texts {
		batch.AddContent(genai.Text(text))
	}

	// 调用模型的 BatchEmbedContents 方法生成批量嵌入向量。
	res, err := m.model.BatchEmbedContents(ctx, batch)
	if err != nil {
		return nil, err
	}

	// 将结果转换为 [][]float32 格式。
	embeddings := make([][]float32, 0, len(res.Embeddings))
	for _, emb := range res.Embeddings {
		embeddings = append(embeddings, emb.Values)
	}

	return embeddings, nil
}
