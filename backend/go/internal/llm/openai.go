package llm

import (
	"Jarvis_2.0/backend/go/internal/models"
	"context"
	"fmt"
	openai "github.com/meguminnnnnnnnn/go-openai"
)

// OpenAI 是一个用于 OpenAI API 的 LLM 客户端。
type OpenAI struct {
	client *openai.Client // OpenAI 客户端实例。
	model  string         // 要使用的模型名称。
	tools  []openai.Tool  // 为该客户端配置的工具列表
}

// NewOpenAI 创建一个新的 OpenAI 客户端。
func NewOpenAI(model string, apiKey string, tools []openai.Tool) (*OpenAI, error) {
	config := openai.DefaultConfig(apiKey)
	client := openai.NewClientWithConfig(config)
	return &OpenAI{
		client: client,
		model:  model,
		tools:  tools,
	}, nil
}

// GenerateContent 使用 OpenAI API 生成内容。
func (o *OpenAI) GenerateContent(ctx context.Context, req *models.GenerateContentRequest) (*models.GenerateContentResponse, error) {
	openaiReq := o.toOpenAIRequest(req)

	// 如果配置了工具，则添加到请求中
	if len(o.tools) > 0 {
		openaiReq.Tools = o.tools
	}

	resp, err := o.client.CreateChatCompletion(ctx, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	return o.toGenerateContentResponse(&resp), nil
}

// GenerateContentStream 使用 OpenAI API 以流式方式生成内容。
func (o *OpenAI) GenerateContentStream(ctx context.Context, req *models.GenerateContentRequest) (<-chan *models.GenerateContentResponse, error) {
	openaiReq := o.toOpenAIRequest(req)

	// 如果配置了工具，则添加到请求中
	if len(o.tools) > 0 {
		openaiReq.Tools = o.tools
	}

	stream, err := o.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion stream: %w", err)
	}

	respChan := make(chan *models.GenerateContentResponse)

go func() {
		defer close(respChan)
		defer stream.Close()

		for {
			resp, err := stream.Recv()
			if err != nil {
				return
			}
			respChan <- o.toGenerateContentResponseStream(&resp)
		}
	}()

	return respChan, nil
}

// toOpenAIRequest 将我们的内部请求格式转换为 OpenAI 格式。
func (o *OpenAI) toOpenAIRequest(req *models.GenerateContentRequest) openai.ChatCompletionRequest {
	var messages []openai.ChatCompletionMessage
	for _, content := range req.Content {
		// TODO: 这里需要更复杂的逻辑来处理多模态内容和 FunctionResponse
		for _, part := range content.Parts {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    string(content.Role),
				Content: part.Text,
			})
		}
	}

	return openai.ChatCompletionRequest{
		Model:    o.model,
		Messages: messages,
	}
}

// toGenerateContentResponse 将 OpenAI 响应转换为我们的内部格式。
func (o *OpenAI) toGenerateContentResponse(resp *openai.ChatCompletionResponse) *models.GenerateContentResponse {
	var content []models.Content
	for _, choice := range resp.Choices {
		// TODO: 这里需要处理 FunctionCall 的返回
		content = append(content, models.Content{
			Parts: []*models.Part{
				{Text: choice.Message.Content},
			},
			Role: models.SpeakerModel,
		})
	}

	return &models.GenerateContentResponse{
		Content:      content,
		ResponseID:   resp.ID,
		ModelVersion: resp.Model,
	}
}

// toGenerateContentResponseStream 将 OpenAI 流式响应转换为我们的内部格式。
func (o *OpenAI) toGenerateContentResponseStream(resp *openai.ChatCompletionStreamResponse) *models.GenerateContentResponse {
	var content []models.Content
	for _, choice := range resp.Choices {
		// TODO: 这里需要处理 FunctionCall 的返回
		content = append(content, models.Content{
			Parts: []*models.Part{
				{Text: choice.Delta.Content},
			},
			Role: models.SpeakerModel,
		})
	}

	return &models.GenerateContentResponse{
		Content:      content,
		ResponseID:   resp.ID,
		ModelVersion: resp.Model,
	}
}