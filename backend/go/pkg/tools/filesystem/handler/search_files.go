package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleSearchFiles 处理文件搜索请求
func (fs *FilesystemHandler) HandleSearchFiles(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 从请求中获取搜索路径参数
	path, err := request.RequireString("path")
	if err != nil {
		return nil, err
	}
	// 获取匹配模式（如 *.go）
	pattern, err := request.RequireString("pattern")
	if err != nil {
		return nil, err
	}

	// 如果路径为 "." 或 "./"，则转换为当前工作目录的绝对路径
	if path == "." || path == "./" {
		cwd, err := os.Getwd()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("解析当前目录出错: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		path = cwd
	}

	// 验证路径是否合法（是否在允许的目录范围内）
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

	// 判断路径是否存在并且是目录
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
	if !info.IsDir() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "错误: 搜索路径必须是目录",
				},
			},
			IsError: true,
		}, nil
	}

	// 调用 searchFiles 执行文件搜索
	results, err := searchFiles(validPath, pattern, fs)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("搜索文件出错: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 如果没有找到任何文件
	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("未找到符合模式 '%s' 的文件 (路径: %s)", pattern, path),
				},
			},
		}, nil
	}

	// 格式化搜索结果，带上资源 URI
	var formattedResults strings.Builder
	formattedResults.WriteString(fmt.Sprintf("找到 %d 个结果:\n\n", len(results)))

	for _, result := range results {
		resourceURI := pathToResourceURI(result)
		info, err := os.Stat(result)
		if err == nil {
			if info.IsDir() {
				// 目录
				formattedResults.WriteString(fmt.Sprintf("[DIR]  %s (%s)\n", result, resourceURI))
			} else {
				// 文件，显示大小
				formattedResults.WriteString(fmt.Sprintf("[FILE] %s (%s) - %d bytes\n",
					result, resourceURI, info.Size()))
			}
		} else {
			// 如果无法获取信息，只显示路径
			formattedResults.WriteString(fmt.Sprintf("%s (%s)\n", result, resourceURI))
		}
	}

	// 返回搜索结果
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: formattedResults.String(),
			},
		},
	}, nil
}

// searchFiles 在指定目录下递归搜索符合模式的文件
func searchFiles(rootPath, pattern string, fs *FilesystemHandler) ([]string, error) {
	var results []string
	// 编译匹配模式，例如 *.go、*.txt
	globPattern := glob.MustCompile(pattern)

	// 遍历目录树
	err := filepath.Walk(
		rootPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // 出错则跳过该文件/目录，继续搜索
			}

			// 验证路径是否合法（是否在允许的目录范围内）
			if _, err := fs.validatePath(path); err != nil {
				return nil // 非法路径直接跳过
			}

			// 如果文件/目录名符合模式，则加入结果
			if globPattern.Match(info.Name()) {
				results = append(results, path)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return results, nil
}
