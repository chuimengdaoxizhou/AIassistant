package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleWriteFile 处理写入文件请求
// 根据传入的 path 和 content，将内容写入指定文件
func (fs *FilesystemHandler) HandleWriteFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 获取 path 参数
	path, err := request.RequireString("path")
	if err != nil {
		return nil, err
	}

	// 获取 content 参数
	content, err := request.RequireString("content")
	if err != nil {
		return nil, err
	}

	// 处理相对路径或 "." 这种情况，转换为绝对路径
	if path == "." || path == "./" {
		// 获取当前工作目录
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error resolving current directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		path = cwd
	}

	// 验证路径是否合法（在允许的目录范围内）
	validPath, err := fs.validatePath(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 检查路径是否是目录，如果是则无法写入文件
	if info, err := os.Stat(validPath); err == nil && info.IsDir() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Cannot write to a directory",
				},
			},
			IsError: true,
		}, nil
	}

	// 如果父目录不存在，则创建父目录
	parentDir := filepath.Dir(validPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error creating parent directories: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 写入文件内容，文件权限设置为 0644
	if err := os.WriteFile(validPath, []byte(content), 0644); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error writing file: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 获取文件信息，用于返回文件大小等信息
	info, err := os.Stat(validPath)
	if err != nil {
		// 文件已成功写入，但无法获取文件信息
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Successfully wrote to %s", path),
				},
			},
		}, nil
	}

	// 生成文件的资源 URI
	resourceURI := pathToResourceURI(validPath)

	// 返回写入成功结果，包括文件大小和资源信息
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			// 文本信息
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Successfully wrote %d bytes to %s", info.Size(), path),
			},
			// 嵌入资源信息
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("File: %s (%d bytes)", validPath, info.Size()),
				},
			},
		},
	}, nil
}
