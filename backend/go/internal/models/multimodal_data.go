package models

import "time"

// SpeakerRole 定义了消息发送者的角色。
type SpeakerRole string

const (
	SpeakerUser      SpeakerRole = "user"      // 用户角色。
	SpeakerAssistant SpeakerRole = "assistant" // 助手角色。
	SpeakerTool      SpeakerRole = "tool"      // 工具角色。
	SpeakerModel     SpeakerRole = "model"     // 模型角色。
)

// HistoryContent 包含了历史消息的内容和元数据。
type HistoryContent struct {
	// 可选。模型返回的响应变体。
	Content *Content `json:"content,omitempty"`
	// 可选。向服务器发出请求时的时间戳。
	CreateTime time.Time `json:"createTime,omitempty"`
	// 用户ID。
	User string `json:"user,omitempty"`
}

// Content 包含了构成单个消息的多个部分。
type Content struct {
	// 可选。构成单个消息的部分列表。每个部分可能具有不同的 IANA MIME 类型。
	Parts []*Part `json:"parts,omitempty"`
	// 可选。内容的生产者。必须是 'user' 或 'model'。
	Role SpeakerRole `json:"role,omitempty"`
}

// GenerateContentRequest 定义了生成内容的请求结构。
type GenerateContentRequest struct {
	Content []Content   `json:"content,omitempty"` // 请求的内容列表。
	Role    SpeakerRole // 请求发送者的角色。
}

// GenerateContentResponse 定义了生成内容的响应结构。
type GenerateContentResponse struct {
	Content      []Content `json:"content,omitempty"`      // 响应的内容列表。
	CreateTime   time.Time `json:"createTime,omitempty"`   // 响应创建时间。
	ResponseID   string    `json:"respinseId,omitempty"`   // 响应ID。
	ModelVersion string    `json:"modelVersion,omitempty"` // 模型版本。
}

// Part 定义了消息的单个部分，可以包含文本、内联数据、文件数据等。
type Part struct {
	// 可选。给定视频的元数据。
	VideoMetadata *VideoMetadata `json:"videoMetadata,omitempty"`
	// 可选。指示该部分是否来自模型的思考。
	Thought bool `json:"thought,omitempty"`
	// 可选。内联字节数据。
	InlineData *Blob `json:"inlineData,omitempty"`
	// 可选。基于 URI 的数据。
	FileData *FileData `json:"fileData,omitempty"`
	// 可选。思考的不透明签名，以便在后续请求中重用。
	ThoughtSignature []byte `json:"thoughtSignature,omitempty"`
	// 可选。执行 [ExecutableCode] 的结果。
	CodeExecutionResult *CodeExecutionResult `json:"codeExecutionResult,omitempty"`
	// 可选。模型生成的用于执行的代码。
	ExecutableCode *ExecutableCode `json:"executableCode,omitempty"`
	// 可选。从模型返回的预测 [FunctionCall]，包含表示 [FunctionDeclaration.Name] 的字符串以及参数及其值。
	FunctionCall *FunctionCall `json:"functionCall,omitempty"`
	// 可选。 [FunctionCall] 的结果输出，包含表示 [FunctionDeclaration.Name] 的字符串和包含函数调用任何输出的结构化 JSON 对象。
	// 它用作模型的上下文。
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
	// 可选。文本部分（可以是代码）。
	Text string `json:"text,omitempty"`
}

// VideoMetadata 包含了视频的元数据。
type VideoMetadata struct {
	// 可选。发送到模型的视频帧率。如果未指定，默认值为 1.0。FPS 范围为 (0.0, 24.0]。
	FPS *float64 `json:"fps,omitempty"`
	// 可选。视频的结束偏移量。
	EndOffset time.Duration `json:"endOffset,omitempty"`
	// 可选。视频的开始偏移量。
	StartOffset time.Duration `json:"startOffset,omitempty"`
}

// Blob 包含了内联的二进制数据。
type Blob struct {
	// 可选。Blob 的显示名称。用于提供标签或文件名以区分 Blob。此字段目前未在 Gemini GenerateContent 调用中使用。
	DisplayName string `json:"displayName,omitempty"`
	// 必填。原始字节数据。
	Data []byte `json:"data,omitempty"`
	// 必填。源数据的 IANA 标准 MIME 类型。
	MIMEType string `json:"mimeType,omitempty"`
}

// FileData 包含了文件数据。
type FileData struct {
	// 可选。文件数据的显示名称。用于提供标签或文件名以区分文件数据。目前未在 Gemini GenerateContent 调用中使用。
	DisplayName string `json:"displayName,omitempty"`
	// 可选。必填。URI。
	FileURI string `json:"fileUri,omitempty"`
	// 可选。必填。源数据的 IANA 标准 MIME 类型。
	MIMEType string `json:"mimeType,omitempty"`
}

// Outcome 定义了代码执行的结果。
type Outcome string

// CodeExecutionResult 包含了代码执行的结果。
type CodeExecutionResult struct {
	// 必填。代码执行的结果。
	Outcome Outcome `json:"outcome,omitempty"`
	// 可选。代码执行成功时包含 stdout，否则包含 stderr 或其他描述。
	Output string `json:"output,omitempty"`
}

// Language 定义了编程语言。
type Language string

// ExecutableCode 包含了模型生成的用于执行的代码。
type ExecutableCode struct {
	// 必填。要执行的代码。
	Code string `json:"code,omitempty"`
	// 必填。`code` 的编程语言。
	Language Language `json:"language,omitempty"`
}

// FunctionCall 包含了模型预测的函数调用信息。
type FunctionCall struct {
	// 可选。函数调用的唯一 ID。如果已填充，客户端将执行 `function_call` 并返回具有匹配 `id` 的响应。
	ID string `json:"id,omitempty"`
	// 可选。JSON 对象格式的函数参数和值。有关参数详细信息，请参阅 [FunctionDeclaration.parameters]。
	Args map[string]any `json:"args,omitempty"`
	// 必填。要调用的函数名称。与 [FunctionDeclaration.Name] 匹配。
	Name string `json:"name,omitempty"`
}

// FunctionResponseScheduling 定义了函数响应的调度方式。
type FunctionResponseScheduling string

// FunctionResponse 包含了函数调用的结果输出。
type FunctionResponse struct {
	// 可选。表示函数调用继续，并将返回更多响应，将函数调用变为生成器。仅适用于 NON_BLOCKING 函数调用（有关详细信息，请参阅 FunctionDeclaration.behavior），否则将被忽略。
	// 如果为 false（默认值），则不会考虑未来的响应。仅适用于 NON_BLOCKING 函数调用，否则将被忽略。如果设置为 false，则不会考虑未来的响应。
	// 允许返回空的 `response` 和 `will_continue=False` 以表示函数调用已完成。
	WillContinue *bool `json:"willContinue,omitempty"`
	// 可选。指定响应在对话中应如何调度。仅适用于 NON_BLOCKING 函数调用，否则将被忽略。默认为 WHEN_IDLE。
	Scheduling FunctionResponseScheduling `json:"scheduling,omitempty"`
	// 可选。此响应对应的函数调用 ID。由客户端填充以匹配相应的函数调用 `id`。
	ID string `json:"id,omitempty"`
	// 必填。要调用的函数名称。与 [FunctionDeclaration.name] 和 [FunctionCall.name] 匹配。
	Name string `json:"name,omitempty"`
	// 必填。JSON 对象格式的函数响应。使用 "output" 键指定函数输出，使用 "error" 键指定错误详细信息（如果有）。
	// 如果未指定 "output" 和 "error" 键，则整个 "response" 被视为函数输出。
	Response map[string]any `json:"response,omitempty"`
}
