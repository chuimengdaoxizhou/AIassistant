package handler

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleSearchWithinFiles 在文件内容中搜索指定子串
func (fs *FilesystemHandler) HandleSearchWithinFiles(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 获取搜索路径参数
	path, err := request.RequireString("path")
	if err != nil {
		return nil, err
	}
	// 获取要搜索的子串
	substring, err := request.RequireString("substring")
	if err != nil {
		return nil, err
	}
	// 子串不能为空
	if substring == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "错误: 子串不能为空",
				},
			},
			IsError: true,
		}, nil
	}

	// 可选参数: 搜索深度，默认 0 表示无限制
	maxDepth := 0
	if depthArg, err := request.RequireFloat("depth"); err == nil {
		maxDepth = int(depthArg)
		if maxDepth < 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "错误: 深度不能为负数",
					},
				},
				IsError: true,
			}, nil
		}
	}

	// 可选参数: 最大搜索结果数量，默认 MAX_SEARCH_RESULTS
	maxResults := MAX_SEARCH_RESULTS
	if maxResultsArg, err := request.RequireFloat("max_results"); err == nil {
		maxResults = int(maxResultsArg)
		if maxResults <= 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "错误: max_results 必须为正数",
					},
				},
				IsError: true,
			}, nil
		}
	}

	// 将相对路径 "." 或 "./" 转换为绝对路径
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

	// 验证路径是否合法
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

	// 调用 searchWithinFiles 执行搜索
	results, err := searchWithinFiles(validPath, substring, maxDepth, maxResults, fs)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("搜索文件内容出错: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 如果没有找到任何结果
	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("在 %s 下未找到 '%s' 的匹配内容", path, substring),
				},
			},
		}, nil
	}

	// 格式化搜索结果
	var formattedResults strings.Builder
	formattedResults.WriteString(fmt.Sprintf("找到 %d 个 '%s' 的匹配结果:\n\n", len(results), substring))

	// 将结果按文件分组，便于阅读
	fileResultsMap := make(map[string][]SearchResult)
	for _, result := range results {
		fileResultsMap[result.FilePath] = append(fileResultsMap[result.FilePath], result)
	}

	// 遍历每个文件的结果
	for filePath, fileResults := range fileResultsMap {
		resourceURI := pathToResourceURI(filePath)
		formattedResults.WriteString(fmt.Sprintf("文件: %s (%s)\n", filePath, resourceURI))

		for _, result := range fileResults {
			lineContent := result.LineContent
			// 如果行内容太长，截取匹配子串周围部分
			if len(lineContent) > 100 {
				substrPos := strings.Index(strings.ToLower(lineContent), strings.ToLower(substring))
				contextStart := max(0, substrPos-30)
				contextEnd := min(len(lineContent), substrPos+len(substring)+30)

				if contextStart > 0 {
					lineContent = "..." + lineContent[contextStart:contextEnd]
				} else {
					lineContent = lineContent[:contextEnd]
				}
				if contextEnd < len(result.LineContent) {
					lineContent += "..."
				}
			}
			formattedResults.WriteString(fmt.Sprintf("  行 %d: %s\n", result.LineNumber, lineContent))
		}
		formattedResults.WriteString("\n")
	}

	// 如果结果被限制数量，说明可能还有更多匹配
	if len(results) >= maxResults {
		formattedResults.WriteString(fmt.Sprintf("\n注意: 搜索结果被限制为 %d 条，可能存在更多匹配。", maxResults))
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

// searchWithinFiles 在文件内容中搜索子串
func searchWithinFiles(
	rootPath, substring string, maxDepth int, maxResults int, fs *FilesystemHandler,
) ([]SearchResult, error) {
	var results []SearchResult
	resultCount := 0
	currentDepth := 0

	// 遍历目录树
	err := filepath.Walk(
		rootPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // 出错则跳过
			}

			// 如果已经达到最大结果数量，跳过当前目录
			if resultCount >= maxResults {
				return filepath.SkipDir
			}

			// 验证路径是否合法
			validPath, err := fs.validatePath(path)
			if err != nil {
				return nil
			}

			// 跳过目录，只搜索文件
			if info.IsDir() {
				relPath, err := filepath.Rel(rootPath, path)
				if err != nil {
					return nil
				}
				if relPath == "" || relPath == "." {
					currentDepth = 0
				} else {
					currentDepth = strings.Count(relPath, string(filepath.Separator)) + 1
				}
				// 超过最大深度则跳过
				if maxDepth > 0 && currentDepth >= maxDepth {
					return filepath.SkipDir
				}
				return nil
			}

			// 跳过过大的文件
			if info.Size() > MAX_SEARCHABLE_SIZE {
				return nil
			}

			// 检测 MIME 类型，只搜索文本文件
			mimeType := detectMimeType(validPath)
			if !isTextFile(mimeType) {
				return nil
			}

			// 打开文件并逐行搜索
			file, err := os.Open(validPath)
			if err != nil {
				return nil
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				line := scanner.Text()
				if strings.Contains(line, substring) {
					results = append(results, SearchResult{
						FilePath:    validPath,
						LineNumber:  lineNum,
						LineContent: line,
						ResourceURI: pathToResourceURI(validPath),
					})
					resultCount++
					if resultCount >= maxResults {
						return filepath.SkipDir
					}
				}
			}
			// 忽略扫描错误
			_ = scanner.Err()

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return results, nil
}

// 辅助函数: 返回两个整数中的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 辅助函数: 返回两个整数中的最大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
