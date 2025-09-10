package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleDeleteFile 方法：处理删除文件或目录的请求
func (fs *FilesystemHandler) HandleDeleteFile(
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
			// 如果获取失败，返回错误结果
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

	// 校验路径是否合法（防止目录穿越或非法路径）
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

	// 检查路径是否存在
	info, err := os.Stat(validPath)
	if os.IsNotExist(err) {
		// 路径不存在
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Path does not exist: %s", path),
				},
			},
			IsError: true,
		}, nil
	} else if err != nil {
		// 其他错误（如权限问题）
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error accessing path: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 读取可选参数 recursive（是否递归删除目录，默认 false）
	recursive := false
	if recursiveParam, err := request.RequireBool("recursive"); err == nil {
		recursive = recursiveParam
	}

	// 如果是目录
	if info.IsDir() {
		// 如果没有设置 recursive=true，则禁止删除目录
		if !recursive {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error: %s is a directory. Use recursive=true to delete directories.", path),
					},
				},
				IsError: true,
			}, nil
		}

		// recursive=true 时，递归删除目录
		if err := os.RemoveAll(validPath); err != nil {
			// 删除失败
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error deleting directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		// 删除成功，返回结果
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Successfully deleted directory %s", path),
				},
			},
		}, nil
	}

	// 如果是文件，直接删除
	if err := os.Remove(validPath); err != nil {
		// 删除失败
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error deleting file: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 删除文件成功
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Successfully deleted file %s", path),
			},
		},
	}, nil
}
