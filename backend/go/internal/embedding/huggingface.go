package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// HuggingFaceModel 是一个用于 Hugging Face Inference API 的 Embedding 模型客户端。
type HuggingFaceModel struct {
	client  *http.Client // HTTP 客户端实例。
	model   string       // 要使用的模型名称。
	apiKey  string       // Hugging Face API 密钥。
	baseURL string       // Hugging Face Inference API 的基准 URL。
}

// NewHuggingFaceModel 创建一个新的 HuggingFaceModel 客户端。
//
// 参数:
//
//	apiKey: Hugging Face 的 API 密钥。
//	modelName: 要使用的模型名称。
//	baseURL: Hugging Face Inference API 的基准 URL。如果为空，则默认为 "https://api-inference.huggingface.co/pipeline/feature-extraction/"。
//
// 返回值:
//
//	*HuggingFaceModel: 新创建的 HuggingFaceModel 客户端实例。
//	error: 如果创建客户端失败，则返回错误。
func NewHuggingFaceModel(apiKey, modelName, baseURL string) (*HuggingFaceModel, error) {
	// 如果 baseURL 为空，则使用默认地址。
	if baseURL == "" {
		baseURL = "https://api-inference.huggingface.co/pipeline/feature-extraction/"
	}
	return &HuggingFaceModel{
		client:  &http.Client{}, // 初始化 HTTP 客户端。
		model:   modelName,
		apiKey:  apiKey,
		baseURL: baseURL,
	}, nil
}

// Embed 使用 Hugging Face Inference API 为单个文本生成嵌入向量。
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
func (m *HuggingFaceModel) Embed(ctx context.Context, text string) ([]float32, error) {
	// 调用 EmbedBatch 方法为单个文本生成嵌入向量。
	embeddings, err := m.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil // 返回第一个嵌入向量。
}

// EmbedBatch 使用 Hugging Face Inference API 为一批文本生成嵌入向量。
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
func (m *HuggingFaceModel) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {

	// 准备请求载荷。
	payload := map[string]interface{}{
		"inputs":  texts,
		"options": map[string]bool{"wait_for_model": true}, // 等待模型加载。
	}

	// 将载荷 Marshal 为 JSON。
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// 创建 HTTP 请求。
	req, err := http.NewRequestWithContext(ctx, "POST", m.baseURL+m.model, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头。
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求。
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() // 确保在函数退出时关闭响应体。

	// 解码响应。
	var embeddings [][]float32
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 检查是否返回了嵌入向量。
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embeddings, nil
}
