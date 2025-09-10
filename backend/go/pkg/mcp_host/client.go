package mcp_host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Host 是一个 MCP 客户端主机
// 它可以连接并管理多个 MCP 服务端，聚合所有工具，并提供统一的调用入口。
type Host struct {
	servers map[string]client.MCPClient // 使用正确的客户端接口
	closers []io.Closer
	mu      sync.RWMutex
}

// ConnectOptions 定义了连接到 MCP 服务端的配置项
type ConnectOptions struct {
	ServerName    string
	TransportType string // "stdio" or "http-sse"
	Command       string
	Args          []string
	URL           string
	Env           []string // 添加环境变量支持
}

// NewHost 创建一个新的 Host 实例
func NewHost() *Host {
	return &Host{
		servers: make(map[string]client.MCPClient),
	}
}

// Connect 根据提供的选项，连接到一个新的 MCP 服务端
func (h *Host) Connect(ctx context.Context, opts ConnectOptions) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.servers[opts.ServerName]; exists {
		return fmt.Errorf("server with name '%s' already connected", opts.ServerName)
	}

	var mcpClient client.MCPClient
	var err error

	switch opts.TransportType {
	case "stdio":
		// 使用正确的stdio客户端创建方法
		mcpClient, err = client.NewStdioMCPClient(opts.Command, opts.Env, opts.Args...)
		if err != nil {
			return fmt.Errorf("failed to create stdio client: %w", err)
		}
	case "http-sse":
		// 使用SSE客户端
		mcpClient, _ = client.NewSSEMCPClient(opts.URL)
	default:
		return fmt.Errorf("unsupported transport type: '%s'", opts.TransportType)
	}

	// 初始化客户端连接
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "mcp-host",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	}

	_, err = mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		mcpClient.Close()
		return fmt.Errorf("failed to initialize client: %w", err)
	}

	h.servers[opts.ServerName] = mcpClient
	return nil
}

// GetAllTools 聚合并返回所有已连接服务端提供的工具列表。
// GetAllTools方法修正 - 支持部分失败
func (h *Host) GetAllTools(ctx context.Context) ([]*mcp.Tool, map[string]error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var allTools []*mcp.Tool
	errors := make(map[string]error)

	for serverName, client := range h.servers {
		toolsResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			errors[serverName] = err
			continue // 跳过失败的服务器，继续处理其他服务器
		}

		// 转换类型：[]Tool -> []*mcp.Tool
		for i := range toolsResult.Tools {
			allTools = append(allTools, &toolsResult.Tools[i])
		}
	}

	return allTools, errors
}

// InvokeTool 在所有连接的服务端中查找并调用指定的工具。
// InvokeTool方法 - 支持部分失败
func (h *Host) InvokeTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, map[string]error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	errors := make(map[string]error)

	for serverName, client := range h.servers {
		toolsResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			errors[serverName] = fmt.Errorf("failed to list tools: %w", err)
			continue
		}

		for _, tool := range toolsResult.Tools {
			if tool.Name == toolName {
				result, err := client.CallTool(ctx, mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Name:      toolName,
						Arguments: args,
					},
				})
				if err != nil {
					errors[serverName] = fmt.Errorf("failed to call tool: %w", err)
					continue
				}
				return result, errors // 成功找到并执行工具
			}
		}
	}

	return nil, errors
}

// CloseAll 关闭所有到服务端的连接并清理资源
func (h *Host) CloseAll() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var errs []error
	for _, client := range h.servers {
		if err := client.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	h.servers = make(map[string]client.MCPClient) // 清空服务器映射
	return errors.Join(errs...)
}
