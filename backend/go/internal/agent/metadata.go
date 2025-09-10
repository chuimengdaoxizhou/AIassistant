package agent

// AgentMetadata 包含了描述一个 Agent 能力所需的所有信息。
type AgentMetadata struct {
	Name                string // Agent 的唯一名称，用作标识符
	Capability          string // 对 Agent 能力的总体描述
	InputDescription    string // 对 Agent 所需输入内容的详细描述
	OutputDescription   string // 对 Agent 输出内容的详细描述
}
