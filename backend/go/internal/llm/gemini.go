package llm

import (
	"Jarvis_2.0/backend/go/internal/models"
	"context"
	"errors"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Gemini 是一个实现了 LLM 接口的结构体，用于与 Gemini API 交互。
type Gemini struct {
	model       *genai.GenerativeModel // Gemini 生成模型实例。
	chatSession *genai.ChatSession     // Gemini 聊天会话实例。
}

// NewGemini 创建一个新的 Gemini 客户端。
//
// 参数:
//
//	ctx: 上下文，用于控制客户端的生命周期。
//	model: 要使用的 Gemini 模型名称。
//	apiKey: Gemini API 密钥。
//
// 返回值:
//
//	*Gemini: 新创建的 Gemini 客户端实例。
//	error: 如果无法创建 GenAI 客户端，则返回错误。
func NewGemini(ctx context.Context, model, apiKey string, tools []*genai.FunctionDeclaration) (*Gemini, error) {
	// 使用 API 密钥创建 GenAI 客户端。
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	// 获取生成模型
	generativeModel := client.GenerativeModel(model)

	// 如果提供了工具，则进行配置
	if len(tools) > 0 {
		geminiTool := &genai.Tool{
			FunctionDeclarations: tools,
		}
		generativeModel.Tools = []*genai.Tool{geminiTool}
	}

	// 启动聊天会话
	chatSession := generativeModel.StartChat()

	return &Gemini{
		model:       generativeModel,
		chatSession: chatSession,
	}, nil
}

// GenerateContent 向 Gemini API 发送请求并返回响应。
//
// 参数:
//
//	ctx: 上下文，用于控制请求的生命周期。
//	req: 生成内容请求。
//
// 返回值:
//
//	*GenerateContentResponse: 生成内容的响应。
//	error: 如果发送消息失败，则返回错误。
func (g *Gemini) GenerateContent(ctx context.Context, req *models.GenerateContentRequest) (*models.GenerateContentResponse, error) {
	// 将内部内容格式转换为 GenAI 部分，并发送消息。
	resp, err := g.chatSession.SendMessage(ctx, toGenaiParts(req.Content)...)
	if err != nil {
		return nil, err
	}

	return fromGenaiResponse(resp), nil // 将 GenAI 响应转换为内部响应格式。
}

// GenerateContentStream 向 Gemini API 发送请求并返回响应通道。
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
func (g *Gemini) GenerateContentStream(ctx context.Context, req *models.GenerateContentRequest) (<-chan *models.GenerateContentResponse, error) {

	ch := make(chan *models.GenerateContentResponse) // 创建用于发送响应的通道。
	// 将内部内容格式转换为 GenAI 部分，并启动流式发送。
	iter := g.chatSession.SendMessageStream(ctx, toGenaiParts(req.Content)...)

	// 启动一个 goroutine 来处理流式响应。
	go func() {
		defer close(ch) // 确保在 goroutine 退出时关闭通道。
		for {
			// 获取下一个流式响应。
			resp, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				return // 流结束。
			}
			if err != nil {
				return
			}
			ch <- fromGenaiResponse(resp) // 将 GenAI 响应转换为内部响应格式并发送到通道。
		}
	}()

	return ch, nil
}

// GetHistory 返回聊天历史记录。
//
// 返回值:
//
//	[]*genai.Content: 聊天历史记录的 GenAI 内容切片。
func (g *Gemini) GetHistory() []*genai.Content {
	return g.chatSession.History // 返回聊天会话的历史记录。
}

// toGenaiParts 将内部 Content 结构体转换为 GenAI Part 切片。
//
// 参数:
//
//	content: 内部 Content 结构体切片。
//
// 返回值:
//
//	[]genai.Part: 转换后的 GenAI Part 切片。
func toGenaiParts(content []models.Content) []genai.Part {
	var parts []genai.Part
	// 遍历内部 Content，将其中的部分转换为对应的 GenAI Part。
	for _, c := range content {
		for _, p := range c.Parts {
			if p.Text != "" {
				parts = append(parts, genai.Text(p.Text))
			} else if p.InlineData != nil {
				parts = append(parts, genai.Blob{
					MIMEType: p.InlineData.MIMEType,
					Data:     p.InlineData.Data,
				})
			} else if p.FileData != nil {
				parts = append(parts, genai.FileData{
					MIMEType: p.FileData.MIMEType,
					URI:      p.FileData.FileURI,
				})
			} else if p.FunctionResponse != nil {
				parts = append(parts, genai.FunctionResponse{
					Name:     p.FunctionResponse.Name,
					Response: p.FunctionResponse.Response,
				})
			}
			// 注意: FunctionCall, CodeExecutionResult, ExecutableCode, VideoMetadata 等
			// 通常是从模型接收的，而不是由客户端在 GenerateContent 请求中发送的。
			// FunctionResponse 是一个例外，它是客户端为了响应模型的 FunctionCall 而发送的。
		}
	}
	return parts
}

// fromGenaiResponse 将 GenAI GenerateContentResponse 转换为内部 GenerateContentResponse 结构体。
//
// 参数:
//
//	resp: GenAI GenerateContentResponse 实例。
//
// 返回值:
//
//	*GenerateContentResponse: 转换后的内部 GenerateContentResponse 实例。
func fromGenaiResponse(resp *genai.GenerateContentResponse) *models.GenerateContentResponse {
	// 如果 GenAI 响应为 nil，则返回 nil。
	if resp == nil {
		return nil
	}
	var content []models.Content
	// 遍历 GenAI 响应中的候选者，并将其内容转换为内部 Content 结构体。
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			content = append(content, fromGenaiContent(cand.Content))
		}
	}
	return &models.GenerateContentResponse{
		Content: content,
	}
}

// fromGenaiContent 将 GenAI Content 结构体转换为内部 Content 结构体。
//
// 参数:
//
//	content: GenAI Content 实例。
//
// 返回值:
//
//	Content: 转换后的内部 Content 结构体。
func fromGenaiContent(content *genai.Content) models.Content {
	var parts []*models.Part
	// 遍历 GenAI Content 中的部分，并将其转换为内部 Part 结构体。
	for _, p := range content.Parts {
		parts = append(parts, fromGenaiPart(p))
	}
	return models.Content{
		Parts: parts,
		Role:  models.SpeakerRole(content.Role),
	}
}

// fromGenaiPart 将 GenAI Part 接口转换为内部 Part 结构体。
//
// 参数:
//
//	part: GenAI Part 接口实例。
//
// 返回值:
//
//	*Part: 转换后的内部 Part 结构体。
func fromGenaiPart(part genai.Part) *models.Part {
	// 根据 GenAI Part 的具体类型进行转换。
	switch v := part.(type) {
	case genai.Text:
		return &models.Part{Text: string(v)}
	case genai.Blob:
		return &models.Part{
			InlineData: &models.Blob{
				MIMEType: v.MIMEType,
				Data:     v.Data,
			},
		}
	//case genai.VideoMetadata:
	//	return &models.Part{
	//		VideoMetadata: &models.VideoMetadata{
	//			FPS:         v.FPS,
	//			EndOffset:   v.EndOffset,
	//			StartOffset: v.StartOffset,
	//		},
	//	}
	case genai.FileData:
		return &models.Part{
			FileData: &models.FileData{
				FileURI:  v.URI,
				MIMEType: v.MIMEType,
			},
		}
	case genai.CodeExecutionResult:
		return &models.Part{
			CodeExecutionResult: &models.CodeExecutionResult{
				Outcome: models.Outcome(v.Outcome),
				Output:  v.Output,
			},
		}
	case genai.ExecutableCode:
		return &models.Part{
			ExecutableCode: &models.ExecutableCode{
				Code:     v.Code,
				Language: models.Language(v.Language),
			},
		}
	case genai.FunctionCall:
		return &models.Part{
			FunctionCall: &models.FunctionCall{
				Name: v.Name,
				Args: v.Args,
			},
		}
	case genai.FunctionResponse:
		return &models.Part{
			FunctionResponse: &models.FunctionResponse{
				Name:     v.Name,
				Response: v.Response,
			},
		}
	default:
		return &models.Part{Text: fmt.Sprintf("%v", v)}
	}
}
