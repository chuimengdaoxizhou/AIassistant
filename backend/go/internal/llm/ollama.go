package llm

import (
	"Jarvis_2.0/backend/go/internal/models"

	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	olla "github.com/ollama/ollama/api"
)

// Ollama 是一个用于 Ollama API 的 LLM 客户端。
type Ollama struct {
	client *olla.Client // Ollama 客户端实例。
	model  string       // 要使用的模型名称。
}

// NewOllama 创建一个新的 Ollama 客户端。
//
// 参数:
//
//	model: 要使用的模型名称。
//	baseURL: Ollama 服务的基准 URL。如果为空，则默认为 "http://localhost:11434"。
//
// 返回值:
//
//	*Ollama: 新创建的 Ollama 客户端实例。
//	error: 如果基准 URL 无效，则返回错误。
func NewOllama(model, baseURL string) (*Ollama, error) {
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
	client := olla.NewClient(parsedURL, hc)

	return &Ollama{client: client, model: model}, nil
}

// GenerateContent 使用 Ollama API 生成内容。
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
func (o *Ollama) GenerateContent(ctx context.Context, req *models.GenerateContentRequest) (*models.GenerateContentResponse, error) {
	// 将内部请求转换为 Ollama 提示格式。
	prompt := o.toOllamaPrompt(req)

	var result *olla.GenerateResponse // 用于存储生成结果。
	var genErr error                  // 用于存储生成过程中的错误。

	// 调用 Ollama 客户端的 Generate 方法生成内容。
	err := o.client.Generate(ctx, &olla.GenerateRequest{
		Model:  o.model,
		Prompt: prompt,
		Stream: &[]bool{false}[0], // 设置为非流式传输。
	}, func(resp olla.GenerateResponse) error {
		result = &resp // 存储响应。
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate content with ollama: %w", err)
	}

	if genErr != nil {
		return nil, genErr
	}

	return o.toGenerateContentResponse(result), nil // 将 Ollama 响应转换为内部响应格式。
}

// GenerateContentStream 使用 Ollama API 以流式方式生成内容。
//
// 参数:
//
//	ctx: 上下文，用于控制请求的生命周期。
//	req: 生成内容请求。
//
// 返回值:
//
//	<-chan *GenerateContentResponse: 接收流式响应的通道。
//	error: 如果无法启动流式发送，则返回错误。
func (o *Ollama) GenerateContentStream(ctx context.Context, req *models.GenerateContentRequest) (<-chan *models.GenerateContentResponse, error) {
	// 将内部请求转换为 Ollama 提示格式。
	prompt := o.toOllamaPrompt(req)
	respChan := make(chan *models.GenerateContentResponse) // 创建用于发送响应的通道。

	// 启动一个 goroutine 来处理流式响应。
	go func() {
		defer close(respChan) // 确保在 goroutine 退出时关闭通道。

		// 调用 Ollama 客户端的 Generate 方法生成内容，并设置为流式传输。
		err := o.client.Generate(ctx, &olla.GenerateRequest{
			Model:  o.model,
			Prompt: prompt,
			Stream: &[]bool{true}[0], // 设置为流式传输。
		}, func(resp olla.GenerateResponse) error {
			respChan <- o.toGenerateContentResponse(&resp) // 将 Ollama 响应转换为内部响应格式并发送到通道。
			return nil
		})

		if err != nil {
			return
		}
	}()

	return respChan, nil
}

// toOllamaPrompt 将内部 GenerateContentRequest 转换为 Ollama 提示字符串。
//
// 参数:
//
//	req: 内部 GenerateContentRequest 实例。
//
// 返回值:
//
//	string: 转换后的 Ollama 提示字符串。
func (o *Ollama) toOllamaPrompt(req *models.GenerateContentRequest) string {
	var sb strings.Builder
	// 遍历请求内容，将所有文本部分拼接成一个字符串。
	for _, content := range req.Content {
		for _, part := range content.Parts {
			sb.WriteString(part.Text)
		}
	}
	return sb.String() // 返回拼接后的字符串。
}

// toGenerateContentResponse 将 Ollama GenerateResponse 转换为内部 GenerateContentResponse 结构体。
//
// 参数:
//
//	resp: Ollama GenerateResponse 实例。
//
// 返回值:
//
//	*GenerateContentResponse: 转换后的内部 GenerateContentResponse 实例。
func (o *Ollama) toGenerateContentResponse(resp *olla.GenerateResponse) *models.GenerateContentResponse {
	return &models.GenerateContentResponse{
		Content: []models.Content{
			{
				Parts: []*models.Part{{Text: resp.Response}}, // 将 Ollama 响应文本作为部分。
				Role:  models.SpeakerModel,                   // 设置角色为助手。
			},
		},
		ModelVersion: resp.Model, // 设置模型版本。
	}
}
