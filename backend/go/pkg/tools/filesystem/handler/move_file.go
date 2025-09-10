package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleMoveFile 用于处理“移动文件”的请求
func (fs *FilesystemHandler) HandleMoveFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 获取请求中的 source 参数（源文件路径）
	source, err := request.RequireString("source")
	if err != nil {
		return nil, err
	}
	// 获取请求中的 destination 参数（目标文件路径）
	destination, err := request.RequireString("destination")
	if err != nil {
		return nil, err
	}

	// 处理源路径为空或相对路径（"." 或 "./"）的情况
	if source == "." || source == "./" {
		// 获取当前工作目录
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("获取当前目录失败: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		source = cwd
	}

	// 处理目标路径为空或相对路径（"." 或 "./"）的情况
	if destination == "." || destination == "./" {
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("获取当前目录失败: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		destination = cwd
	}

	// 验证源路径是否合法（是否在允许访问的目录中）
	validSource, err := fs.validatePath(source)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("源路径错误: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 检查源文件是否存在
	if _, err := os.Stat(validSource); os.IsNotExist(err) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("错误: 源文件不存在: %s", source),
				},
			},
			IsError: true,
		}, nil
	}

	// 获取目标路径的父目录，并验证合法性
	destDir := filepath.Dir(destination)
	validDestDir, err := fs.validatePath(destDir)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("目标目录路径错误: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 如果目标父目录不存在，则自动创建
	if err := os.MkdirAll(validDestDir, 0755); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("创建目标目录失败: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 验证完整的目标路径是否合法
	validDest, err := fs.validatePath(destination)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("目标路径错误: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 执行文件移动操作
	if err := os.Rename(validSource, validDest); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("移动文件失败: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 生成资源 URI，返回结果
	resourceURI := pathToResourceURI(validDest)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf(
					"文件已成功从 %s 移动到 %s",
					source,
					destination,
				),
			},
			// 返回一个嵌入式资源，包含目标路径的文本信息
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("已移动文件: %s", validDest),
				},
			},
		},
	}, nil
}
