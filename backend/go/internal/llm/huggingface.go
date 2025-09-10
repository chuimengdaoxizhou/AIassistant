package llm

import (
	"Jarvis_2.0/backend/go/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// HuggingFace 是一个用于 Hugging Face Inference API 的 LLM 客户端。
type HuggingFace struct {
	client  *http.Client // HTTP 客户端实例。
	model   string       // 要使用的模型名称。
	apiKey  string       // Hugging Face API 密钥。
	baseURL string       // Hugging Face Inference API 的基准 URL。
}

// NewHuggingFace 创建一个新的 HuggingFace 客户端。
//
// 参数:
//
//	model: 要使用的模型名称。
//	apiKey: Hugging Face API 密钥。
//	baseURL: Hugging Face Inference API 的基准 URL。如果为空，则默认为 "https://api-inference.huggingface.co/models/"。
//
// 返回值:
//
//	*HuggingFace: 新创建的 HuggingFace 客户端实例。
//	error: 如果创建客户端失败，则返回错误。
func NewHuggingFace(model, apiKey, baseURL string) (*HuggingFace, error) {
	// 如果 baseURL 为空，则使用默认地址。
	if baseURL == "" {
		baseURL = "https://api-inference.huggingface.co/models/"
	}
	return &HuggingFace{
		client:  &http.Client{}, // 初始化 HTTP 客户端。
		model:   model,
		apiKey:  apiKey,
		baseURL: baseURL,
	}, nil
}

// GenerateContent 使用 Hugging Face Inference API 生成内容。
//
// 参数:
//
//	ctx: 上下文，用于控制请求的生命周期。
//	req: 生成内容请求。
//
// 返回值:
//
//	*GenerateContentResponse: 生成内容的响应。
//	error: 如果生成内容失败，则返回错误。
func (h *HuggingFace) GenerateContent(ctx context.Context, req *models.GenerateContentRequest) (*models.GenerateContentResponse, error) {

	// 将内部请求转换为 Hugging Face 格式。
	hfReq := h.toHuggingFaceRequest(req)

	// 将载荷 Marshal 为 JSON。
	jsonReq, err := json.Marshal(hfReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求。
	httpReq, err := http.NewRequestWithContext(ctx, "POST", h.baseURL+h.model, bytes.NewBuffer(jsonReq))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头。
	httpReq.Header.Set("Authorization", "Bearer "+h.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求。
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() // 确保在函数退出时关闭响应体。

	// 解码响应。
	var hfResp []struct {
		GeneratedText string `json:"generated_text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hfResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 检查是否返回了生成文本。
	if len(hfResp) == 0 {
		return nil, fmt.Errorf("no generated text returned")
	}

	// 将响应转换回我们的格式。
	return h.toGenerateContentResponse(hfResp), nil
}

// GenerateContentStream 尚未为 Hugging Face 实现。
//
// 参数:
//
//	ctx: 上下文，用于控制请求的生命周期。
//	req: 生成内容请求。
//
// 返回值:
//
//	<-chan *GenerateContentResponse: 接收流式响应的通道。
//	error: 始终返回错误，因为流式传输尚未实现。
func (h *HuggingFace) GenerateContentStream(ctx context.Context, req *models.GenerateContentRequest) (<-chan *models.GenerateContentResponse, error) {
	// 开始一个新的 span，用于跟踪流式生成内容操作。
	return nil, fmt.Errorf("streaming not yet implemented for Hugging Face") // 返回错误。
}

// toHuggingFaceRequest 将我们的内部请求格式转换为 Hugging Face 格式。
//
// 参数:
//
//	req: 内部 GenerateContentRequest 实例。
//
// 返回值:
//
//	map[string]interface{}: 转换后的 Hugging Face 请求映射。
func (h *HuggingFace) toHuggingFaceRequest(req *models.GenerateContentRequest) map[string]interface{} {
	var inputs string
	// 遍历请求内容，将所有文本部分拼接成一个字符串。
	for _, content := range req.Content {
		for _, part := range content.Parts {
			inputs += part.Text
		}
	}

	return map[string]interface{}{
		"inputs": inputs, // 设置输入文本。
	}
}

// toGenerateContentResponse 将 Hugging Face 响应转换为我们的内部格式。
//
// 参数:
//
//	resp: Hugging Face 响应结构体切片。
//
// 返回值:
//
//	*GenerateContentResponse: 转换后的内部 GenerateContentResponse 实例。
func (h *HuggingFace) toGenerateContentResponse(resp []struct {
	GeneratedText string `json:"generated_text"`
}) *models.GenerateContentResponse {
	var content []models.Content
	// 遍历 Hugging Face 响应，将其生成文本转换为内部 Content 结构体。
	for _, item := range resp {
		content = append(content, models.Content{
			Parts: []*models.Part{
				{Text: item.GeneratedText},
			},
			Role: models.SpeakerModel,
		})
	}

	return &models.GenerateContentResponse{
		Content: content,
	}
}
