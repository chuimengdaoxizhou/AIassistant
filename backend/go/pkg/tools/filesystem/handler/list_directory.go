package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleListDirectory 用于处理“列出目录内容”的请求
// 输入参数为一个路径（path），输出为目录下的文件和文件夹列表
func (fs *FilesystemHandler) HandleListDirectory(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 从请求中获取参数 "path"
	path, err := request.RequireString("path")
	if err != nil {
		return nil, err
	}

	// 处理 "." 或 "./" 这种相对路径，转为当前工作目录的绝对路径
	if path == "." || path == "./" {
		cwd, err := os.Getwd() // 获取当前工作目录
		if err != nil {
			// 返回错误结果
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("解析当前目录失败: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		path = cwd
	}

	// 校验路径是否在允许访问的目录范围内
	validPath, err := fs.validatePath(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("错误: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 判断路径是否存在，以及是否为目录
	info, err := os.Stat(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("错误: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 如果路径不是目录，直接报错
	if !info.IsDir() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "错误: 该路径不是一个目录",
				},
			},
			IsError: true,
		}, nil
	}

	// 读取目录内容
	entries, err := os.ReadDir(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("读取目录失败: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 构建输出结果
	var result strings.Builder
	result.WriteString(fmt.Sprintf("目录内容列表: %s\n\n", validPath))

	// 遍历目录项
	for _, entry := range entries {
		entryPath := filepath.Join(validPath, entry.Name()) // 拼接完整路径
		resourceURI := pathToResourceURI(entryPath)         // 转换为资源URI

		if entry.IsDir() {
			// 文件夹
			result.WriteString(fmt.Sprintf("[DIR]  %s (%s)\n", entry.Name(), resourceURI))
		} else {
			// 文件（输出大小）
			info, err := entry.Info()
			if err == nil {
				result.WriteString(fmt.Sprintf("[FILE] %s (%s) - %d bytes\n",
					entry.Name(), resourceURI, info.Size()))
			} else {
				result.WriteString(fmt.Sprintf("[FILE] %s (%s)\n", entry.Name(), resourceURI))
			}
		}
	}

	// 返回目录列表的文本信息 + 目录本身作为一个资源对象
	resourceURI := pathToResourceURI(validPath)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			// 文本形式的目录清单
			mcp.TextContent{
				Type: "text",
				Text: result.String(),
			},
			// 嵌入资源（可供后续引用）
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("目录: %s", validPath),
				},
			},
		},
	}, nil
}
