
package models

// AgentMetadata 包含了描述一个 Agent 能力所需的所有信息。
type AgentMetadata struct {
	Name        string  `json:"name"`        // Agent 的唯一名称，用作标识符
	Description string  `json:"description"` // 对 Agent 能力的总体描述
	InputSchema *Schema `json:"inputSchema"` // Agent 输入参数的 Schema 定义
}

// Schema 定义了工具或Agent的输入参数结构，兼容OpenAPI 3.0.3规范。
type Schema struct {
	Type       string             `json:"type"`                 // 参数类型 (e.g., "object", "string", "number")
	Properties map[string]*Schema `json:"properties,omitempty"` // 如果类型是 "object", 定义其属性
	Required   []string           `json:"required,omitempty"`   // "object" 类型中的必需属性列表
	Description string            `json:"description,omitempty"` // 参数的描述
	Enum       []string           `json:"enum,omitempty"`       // 如果类型是 "string", 可选的枚举值
	Items      *Schema            `json:"items,omitempty"`      // 如果类型是 "array", 定义数组元素的类型
}
