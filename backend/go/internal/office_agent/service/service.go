
package service

import (
	"Jarvis_2.0/api/proto/v1"
	"Jarvis_2.0/backend/go/internal/agent"
	"Jarvis_2.0/backend/go/internal/config"
	"Jarvis_2.0/backend/go/internal/database/minio"
	"Jarvis_2.0/backend/go/internal/llm"
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"Jarvis_2.0/backend/go/pkg/mcp_host"
	"context"
	"encoding/json"
	"fmt"
	minioGo "github.com/minio/minio-go/v7"
	"os"
	"strings"
)

const maxIterations = 10 // ReAct 循环的最大次数
const minioScheme = "minio://"

// OfficeAgentService 实现了 Agent 核心逻辑，并集成了 MinIO。
type OfficeAgentService struct {
	llmClient   llm.LLM
	mcpHost     *mcp_host.Host
	minioClient *minioGo.Client
	minioBucket string
}

// LLMConfig 封装了创建 LLM 客户端所需的配置
type LLMConfig struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
}

// NewOfficeAgentService 创建一个新的 OfficeAgentService 实例。
func NewOfficeAgentService(ctx context.Context, llmConfig LLMConfig, minioConfig config.MinIOConfig) (*OfficeAgentService, error) {
	appLogger := logger.New("office_agent_service_init", "", "")
	appLogger.Info("Initializing OfficeAgentService with MinIO integration...")

	// 1. 初始化 MinIO Client
	minioClient, err := minio.GetClient(&minioConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}
	appLogger.Info("MinIO client initialized.")

	// 2. 初始化 MCP Host
	host := mcp_host.NewHost()
	mcpServers := []mcp_host.ConnectOptions{
		{ServerName: "excel-editor", TransportType: "stdio", Command: "go", Args: []string{"run", "backend/go/pkg/tools/edit_office/excel_mcp_server.go"}},
		{ServerName: "word-editor", TransportType: "stdio", Command: "go", Args: []string{"run", "backend/go/pkg/tools/edit_office/word_mcp_server.go"}},
		{ServerName: "ppt-editor", TransportType: "stdio", Command: "go", Args: []string{"run", "backend/go/pkg/tools/edit_office/ppt_mcp_server.go"}},
	}

	for _, serverOpts := range mcpServers {
		if err := host.Connect(ctx, serverOpts); err != nil {
			host.CloseAll()
			return nil, fmt.Errorf("failed to connect to MCP server '%s': %w", serverOpts.ServerName, err)
		}
	}
	appLogger.Info("All MCP servers connected.")

	// 3. 获取工具并初始化 LLM
	officeTools, errors := host.GetAllTools(ctx)
	if len(errors) > 0 {
		appLogger.Warn(fmt.Sprintf("Encountered errors while getting tools: %v", errors))
	}
	if len(officeTools) == 0 {
		host.CloseAll()
		return nil, fmt.Errorf("no tools found from any mcp_host server")
	}

	llmClient, err := llm.NewLLM(llmConfig.Provider, llmConfig.Model, llmConfig.APIKey, llmConfig.BaseURL, officeTools)
	if err != nil {
		host.CloseAll()
		return nil, fmt.Errorf("failed to create LLM client with office tools: %w", err)
	}
	appLogger.Info("LLM client initialized with specific office tools.")

	return &OfficeAgentService{
		llmClient:   llmClient,
		mcpHost:     host,
		minioClient: minioClient,
		minioBucket: minioConfig.Bucket,
	}, nil
}

func (s *OfficeAgentService) Close() error {
	return s.mcpHost.CloseAll()
}

func (s *OfficeAgentService) Metadata() agent.AgentMetadata {
	return agent.AgentMetadata{
		Name: "office_agent",
		Capability: "Understands complex, multi-step tasks related to Office documents (Word, Excel, PowerPoint). " +
			"It can create and edit files, saving them to and loading them from MinIO object storage.",
		InputDescription: "A natural language string describing the entire task. File paths should be specified as MinIO URIs (e.g., 'minio://bucket/path/to/file.docx') when referring to existing files.",
		OutputDescription: "A string summarizing the final result, including the MinIO path of any created or modified files.",
	}
}

func (s *OfficeAgentService) Execute(ctx context.Context, task *v1.AgentTask) (*v1.AgentTask, error) {
	appLogger := logger.New("office_agent_service", task.TaskId, task.CorrelationId)
	appLogger.Info("Office agent service received task")

	history := models.ConvertProtoToModelsContent(task.Content)

	for i := 0; i < maxIterations; i++ {
		appLogger.Info(fmt.Sprintf("ReAct loop iteration: %d", i+1))

		llmReq := &models.GenerateContentRequest{Content: history}

		llmResp, err := s.llmClient.GenerateContent(ctx, llmReq)
		if err != nil {
			appLogger.Error(fmt.Sprintf("LLM GenerateContent failed: %v", err))
			return nil, fmt.Errorf("llm generation failed: %w", err)
		}

		hasFunctionCall := false
		if len(llmResp.Content) > 0 && len(llmResp.Content[0].Parts) > 0 {
			for _, part := range llmResp.Content[0].Parts {
				if part.FunctionCall != nil {
					hasFunctionCall = true
					break
				}
			}
		}

		if hasFunctionCall {
			appLogger.Info("LLM decided to call a function.")
			history = append(history, llmResp.Content...)

			observationContent, err := s.executeFunctionCall(ctx, llmResp.Content[0].Parts)
			if err != nil {
				appLogger.Error(fmt.Sprintf("Function call execution failed: %v", err))
				history = append(history, models.Content{
					Role:  models.SpeakerTool,
					Parts: []*models.Part{{Text: fmt.Sprintf("Error executing tool: %v", err)}},
				})
			} else {
				history = append(history, observationContent)
			}
			continue
		} else {
			finalAnswer := "Task finished."
			if len(llmResp.Content) > 0 && len(llmResp.Content[0].Parts) > 0 {
				finalAnswer = llmResp.Content[0].Parts[0].Text
			}
			appLogger.Info(fmt.Sprintf("LLM provided final answer: %s", finalAnswer))
			return models.ConvertModelsToProtoTask(task, models.Content{Parts: []*models.Part{{Text: finalAnswer}}}, task.SourceAgentId, "Final Result")
		}
	}

	appLogger.Error("Reached max iterations, stopping.")
	return nil, fmt.Errorf("reached max iterations")
}

// executeFunctionCall handles the entire lifecycle of a tool call, including MinIO operations.
func (s *OfficeAgentService) executeFunctionCall(ctx context.Context, parts []*models.Part) (models.Content, error) {
	for _, part := range parts {
		if part.FunctionCall == nil {
			continue
		}
		fc := part.FunctionCall
		toolName := fc.Name
		args := fc.Args

		appLogger := logger.New("office_tool_executor", "", "")
		appLogger.Info(fmt.Sprintf("Executing tool '%s' with MinIO integration", toolName))

		var tempFile *os.File
		var err error
		var originalMinioPath string
		var objectName string

		filePathArg, pathOK := args["file_path"].(string)
		if !pathOK {
			return models.Content{}, fmt.Errorf("'file_path' argument is missing or not a string for tool %s", toolName)
		}

		isCreate := isCreationTool(toolName)
		isEdit := !isCreate

		tempFile, err = os.CreateTemp("", "office-agent-*")
		if err != nil {
			return models.Content{}, fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tempFile.Name())

		if isEdit {
			if !strings.HasPrefix(filePathArg, minioScheme) {
				return models.Content{}, fmt.Errorf("invalid file path for editing: must be a MinIO path (e.g., minio://...)")
			}
			originalMinioPath = filePathArg
			objectName = strings.TrimPrefix(originalMinioPath, minioScheme+s.minioBucket+"/")
			appLogger.Info(fmt.Sprintf("Downloading '%s' from MinIO to '%s'", objectName, tempFile.Name()))
			err = s.minioClient.FGetObject(ctx, s.minioBucket, objectName, tempFile.Name(), minioGo.GetObjectOptions{})
			if err != nil {
				return models.Content{}, fmt.Errorf("failed to download file from MinIO: %w", err)
			}
		}

		args["file_path"] = tempFile.Name()
		tempFile.Close()

		appLogger.Info(fmt.Sprintf("Invoking MCP tool '%s' on temp file '%s'", toolName, args["file_path"]))
		result, errors := s.mcpHost.InvokeTool(ctx, toolName, args)
		if len(errors) > 0 {
			return models.Content{}, fmt.Errorf("mcp tool invocation failed: %v", errors)
		}
		if result == nil {
			return models.Content{}, fmt.Errorf("tool '%s' not found by mcp_host", toolName)
		}

		var finalMinioPath string
		if isCreate {
			finalMinioPath = filePathArg
			objectName = strings.TrimPrefix(finalMinioPath, minioScheme+s.minioBucket+"/")
		} else { // isEdit
			finalMinioPath = originalMinioPath
			// objectName is already set from the download step
		}

		appLogger.Info(fmt.Sprintf("Uploading '%s' to MinIO as '%s'", args["file_path"], objectName))
		_, err = s.minioClient.FPutObject(ctx, s.minioBucket, objectName, args["file_path"].(string), minioGo.PutObjectOptions{})
		if err != nil {
			return models.Content{}, fmt.Errorf("failed to upload file to MinIO: %w", err)
		}

		var observation string
		if isCreate {
			observation = fmt.Sprintf("Successfully created file and saved it to MinIO at: %s", finalMinioPath)
		} else {
			resultBytes, _ := json.Marshal(result.Result)
			observation = fmt.Sprintf("Successfully edited file %s. Details: %s", finalMinioPath, string(resultBytes))
		}
		appLogger.Info(fmt.Sprintf("Observation for tool '%s': %s", toolName, observation))

		return models.Content{
			Role: models.SpeakerTool,
			Parts: []*models.Part{{
				FunctionResponse: &models.FunctionResponse{
					Name:     toolName,
					Response: map[string]any{"output": observation},
				},
			}},
		}, nil
	}
	return models.Content{}, fmt.Errorf("no function call found in model response")
}

// isCreationTool is a helper to identify tools that create new files.
func isCreationTool(toolName string) bool {
	return strings.HasSuffix(toolName, "_new_workbook") ||
		strings.HasSuffix(toolName, "_new_document") ||
		strings.HasSuffix(toolName, "_new_presentation")
}
