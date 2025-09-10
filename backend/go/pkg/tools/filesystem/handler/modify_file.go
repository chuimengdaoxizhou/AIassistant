package handler

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleModifyFile 处理“修改文件”的请求
// 支持字符串替换和正则替换，可选择替换单个或全部匹配项
func (fs *FilesystemHandler) HandleModifyFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 提取必要参数
	path, err := request.RequireString("path") // 文件路径
	if err != nil {
		return nil, err
	}

	find, err := request.RequireString("find") // 查找内容
	if err != nil {
		return nil, err
	}

	replace, err := request.RequireString("replace") // 替换内容
	if err != nil {
		return nil, err
	}

	// 提取可选参数，带默认值
	allOccurrences := true // 默认替换所有匹配
	if val, err := request.RequireBool("all_occurrences"); err == nil {
		allOccurrences = val
	}

	useRegex := false // 默认不使用正则
	if val, err := request.RequireBool("regex"); err == nil {
		useRegex = val
	}

	// 处理 "." 或 "./" 这种路径，将其转为绝对路径
	if path == "." || path == "./" {
		cwd, err := os.Getwd() // 获取当前工作目录
		if err != nil {
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

	// 校验路径是否在允许访问范围内
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

	// 判断路径是否是目录（不能修改目录）
	if info, err := os.Stat(validPath); err == nil && info.IsDir() {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "错误: 不能修改目录",
				},
			},
			IsError: true,
		}, nil
	}

	// 判断文件是否存在
	if _, err := os.Stat(validPath); os.IsNotExist(err) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("错误: 文件不存在: %s", path),
				},
			},
			IsError: true,
		}, nil
	}

	// 读取文件内容
	content, err := os.ReadFile(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("读取文件失败: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	originalContent := string(content) // 原始文件内容
	modifiedContent := ""              // 修改后的内容
	replacementCount := 0              // 替换次数

	// 执行替换逻辑
	if useRegex {
		// 使用正则替换
		re, err := regexp.Compile(find)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("错误: 无效的正则表达式: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		if allOccurrences {
			// 替换所有匹配
			modifiedContent = re.ReplaceAllString(originalContent, replace)
			replacementCount = len(re.FindAllString(originalContent, -1))
		} else {
			// 只替换第一个匹配
			matched := re.FindStringIndex(originalContent)
			if matched != nil {
				replacementCount = 1
				modifiedContent = originalContent[:matched[0]] + replace + originalContent[matched[1]:]
			} else {
				modifiedContent = originalContent
				replacementCount = 0
			}
		}
	} else {
		// 普通字符串替换
		if allOccurrences {
			// 替换所有
			replacementCount = strings.Count(originalContent, find)
			modifiedContent = strings.ReplaceAll(originalContent, find, replace)
		} else {
			// 只替换第一个
			if index := strings.Index(originalContent, find); index != -1 {
				replacementCount = 1
				modifiedContent = originalContent[:index] + replace + originalContent[index+len(find):]
			} else {
				modifiedContent = originalContent
				replacementCount = 0
			}
		}
	}

	// 将修改后的内容写回文件
	if err := os.WriteFile(validPath, []byte(modifiedContent), 0644); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("写入文件失败: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 创建响应
	resourceURI := pathToResourceURI(validPath)

	// 获取文件信息，用于返回文件大小等信息
	info, err := os.Stat(validPath)
	if err != nil {
		// 文件已写入成功，但获取信息失败
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("文件修改成功，共进行了 %d 次替换。", replacementCount),
				},
			},
		}, nil
	}

	// 正常返回结果：替换次数、文件大小、资源URI
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("文件修改成功，共进行了 %d 次替换，文件路径: %s (大小: %d bytes)",
					replacementCount, path, info.Size()),
			},
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("修改后的文件: %s (%d bytes)", validPath, info.Size()),
				},
			},
		},
	}, nil
}
