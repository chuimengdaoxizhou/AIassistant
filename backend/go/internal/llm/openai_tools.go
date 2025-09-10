package llm

import (
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/meguminnnnnnnnn/go-openai"
)

// ConvertMCPToolsToOpenAI converts a list of tools from the mcp_host client to the FunctionDefinition list required by the OpenAI Go SDK.
func ConvertMCPToolsToOpenAI(tools []*mcp.Tool) ([]openai.Tool, error) {
	var openAITools []openai.Tool

	for _, tool := range tools {
		params, err := convertMCPParamsToOpenAISchema(tool.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("error converting parameters for tool '%s': %w", tool.Name, err)
		}

		openAITool := openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			},
		}
		openAITools = append(openAITools, openAITool)
	}

	return openAITools, nil
}

// convertMCPParamsToOpenAISchema is a helper function that converts mcp.ToolInputSchema to the format required by OpenAI.
func convertMCPParamsToOpenAISchema(mcpParams mcp.ToolInputSchema) (map[string]interface{}, error) {
	if len(mcpParams.Properties) == 0 {
		return nil, nil
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": mcpParams.Properties,
		"required":   mcpParams.Required,
	}

	return schema, nil
}
