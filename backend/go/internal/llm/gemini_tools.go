package llm

import (
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/mark3labs/mcp-go/mcp"
)

// ConvertMCPToolsToGemini 将从 mcp_host 客户端获取的工具列表转换为 Gemini Go SDK 所需的 FunctionDeclaration 列表。
func ConvertMCPToolsToGemini(tools []*mcp.Tool) ([]*genai.FunctionDeclaration, error) {
	var declarations []*genai.FunctionDeclaration

	for _, tool := range tools {
		declaration := &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
		}

		params, required, err := convertMCPParamsToGeminiSchema(tool.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("error converting parameters for tool '%s': %w", tool.Name, err)
		}

		if params != nil {
			declaration.Parameters = params
			declaration.Parameters.Required = required
		}

		declarations = append(declarations, declaration)
	}

	return declarations, nil
}

// convertMCPParamsToGeminiSchema 辅助函数，其签名恢复到之前的版本，但函数体更健壮
func convertMCPParamsToGeminiSchema(mcpParams mcp.ToolInputSchema) (*genai.Schema, []string, error) {
	if len(mcpParams.Properties) == 0 {
		return nil, nil, nil
	}

	geminiSchema := &genai.Schema{
		Type:       genai.TypeObject,
		Properties: make(map[string]*genai.Schema),
	}

	for name, param := range mcpParams.Properties {
		// param is an interface{}. We need to assert it to a map[string]interface{}
		paramMap, ok := param.(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("invalid parameter format for %s", name)
		}

		propSchema := &genai.Schema{}
		if desc, ok := paramMap["description"].(string); ok {
			propSchema.Description = desc
		}

		if paramType, ok := paramMap["type"].(string); ok {
			switch paramType {
			case "string":
				propSchema.Type = genai.TypeString
			case "integer":
				propSchema.Type = genai.TypeInteger
			case "number":
				propSchema.Type = genai.TypeNumber
			case "boolean":
				propSchema.Type = genai.TypeBoolean
			case "array":
				propSchema.Type = genai.TypeArray
			case "object":
				propSchema.Type = genai.TypeObject
			default:
				return nil, nil, fmt.Errorf("unsupported parameter type: %s", paramType)
			}
		} else {
			return nil, nil, fmt.Errorf("parameter type not specified for %s", name)
		}

		geminiSchema.Properties[name] = propSchema
	}

	return geminiSchema, mcpParams.Required, nil
}