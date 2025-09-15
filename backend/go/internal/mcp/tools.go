package mcp

import v1 "Jarvis_2.0/api/proto/v1"

// GetTools 返回所有本地可用的MCP（模型控制程序）工具的元数据。
// 返回值已修改为 []*v1.AgentMetadata 以便与服务发现的类型统一。
func GetTools() []*v1.AgentMetadata {
	return []*v1.AgentMetadata{
		{
			Name:             "execute_code",
			Capability:       "Executes a block of Python code and returns the result.",
			InputDescription: "A self-contained block of Python code to be executed.",
		},
	}
}