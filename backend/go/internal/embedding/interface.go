package embedding

import "context"

// Embedding 定义了所有 embedding 模型需要实现的接口。
type Embedding interface {
	// Embed 为单个文本生成嵌入向量。
	//
	// 参数:
	//   ctx: 上下文，用于控制操作的生命周期。
	//   text: 要生成嵌入向量的文本。
	//
	// 返回值:
	//   []float32: 生成的嵌入向量。
	//   error: 如果生成嵌入向量失败，则返回错误。
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch 为一批文本生成嵌入向量。
	//
	// 参数:
	//   ctx: 上下文，用于控制操作的生命周期。
	//   texts: 要生成嵌入向量的文本切片。
	//
	// 返回值:
	//   [][]float32: 生成的嵌入向量切片。
	//   error: 如果生成嵌入向量失败，则返回错误。
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// ModelType 是一个枚举类型，用于表示不同的模型厂商。
type ModelType string

const (
	OpenAI      ModelType = "openai"      // OpenAI 模型类型。
	Google      ModelType = "google"      // Google 模型类型。
	Ollama      ModelType = "ollama"      // Ollama 模型类型。
	HuggingFace ModelType = "huggingface" // HuggingFace 模型类型。
)
