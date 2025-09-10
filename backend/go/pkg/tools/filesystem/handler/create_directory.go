package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleCreateDirectory 方法：处理创建目录的请求
func (fs *FilesystemHandler) HandleCreateDirectory(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 从请求参数中获取路径
	path, err := request.RequireString("path")
	if err != nil {
		return nil, err
	}

	// 如果路径是 "." 或 "./"，则转换为绝对路径（当前工作目录）
	if path == "." || path == "./" {
		// 获取当前工作目录
		cwd, err := os.Getwd()
		if err != nil {
			// 返回错误信息
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

	// 校验路径是否合法（防止目录穿越或非法访问）
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

	// 检查路径是否已经存在
	if info, err := os.Stat(validPath); err == nil {
		// 如果已存在且是目录
		if info.IsDir() {
			resourceURI := pathToResourceURI(validPath)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					// 返回提示：目录已存在
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Directory already exists: %s", path),
					},
					// 返回资源信息
					mcp.EmbeddedResource{
						Type: "resource",
						Resource: mcp.TextResourceContents{
							URI:      resourceURI,
							MIMEType: "text/plain",
							Text:     fmt.Sprintf("Directory: %s", validPath),
						},
					},
				},
			}, nil
		}
		// 如果路径存在但不是目录，返回错误
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Path exists but is not a directory: %s", path),
				},
			},
			IsError: true,
		}, nil
	}

	// 创建目录（包括父目录，权限 0755）
	if err := os.MkdirAll(validPath, 0755); err != nil {
		// 创建失败，返回错误
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error creating directory: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 创建成功，返回结果
	resourceURI := pathToResourceURI(validPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			// 提示信息
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Successfully created directory %s", path),
			},
			// 返回资源信息
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("Directory: %s", validPath),
				},
			},
		},
	}, nil
}
