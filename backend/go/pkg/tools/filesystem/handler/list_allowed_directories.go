package handler

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleListAllowedDirectories 方法：列出允许访问的目录
func (fs *FilesystemHandler) HandleListAllowedDirectories(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 去掉路径末尾的分隔符（仅用于展示，不影响实际路径）
	displayDirs := make([]string, len(fs.allowedDirs))
	for i, dir := range fs.allowedDirs {
		displayDirs[i] = strings.TrimSuffix(dir, string(filepath.Separator))
	}

	// 构建展示结果
	var result strings.Builder
	result.WriteString("Allowed directories:\n\n")

	// 遍历允许访问的目录，生成对应的资源 URI
	for _, dir := range displayDirs {
		resourceURI := pathToResourceURI(dir)
		// 每行展示：目录路径 + 资源 URI
		result.WriteString(fmt.Sprintf("%s (%s)\n", dir, resourceURI))
	}

	// 返回结果
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text", // 普通文本输出
				Text: result.String(),
			},
		},
	}, nil
}
