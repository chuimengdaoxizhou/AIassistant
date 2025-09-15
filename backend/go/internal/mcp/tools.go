
package mcp

import "Jarvis_2.0/backend/go/internal/models"

// GetTools 返回所有本地可用的MCP（模型控制程序）工具的元数据。
// 在实际应用中，这些可以从配置文件或插件目录中加载。
func GetTools() []models.AgentMetadata {
	return []models.AgentMetadata{
		{
			Name:        "execute_code",
			Description: "Execute a block of Python code and return the result. The code should be self-contained and not rely on external state.",
			InputSchema: &models.Schema{
				Type: "object",
				Properties: map[string]*models.Schema{
					"code": {
						Type:        "string",
						Description: "The Python code to execute.",
					},
				},
				Required: []string{"code"},
			},
		},
	}
}
