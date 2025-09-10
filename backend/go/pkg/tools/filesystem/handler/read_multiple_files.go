package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleReadMultipleFiles 处理读取多个文件内容的工具请求。
func (fs *FilesystemHandler) HandleReadMultipleFiles(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 提取文件路径列表参数
	pathsSlice, err := request.RequireStringSlice("paths")
	if err != nil {
		return nil, err
	}

	// 检查是否指定了文件
	if len(pathsSlice) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "No files specified to read",
				},
			},
			IsError: true,
		}, nil
	}

	// 单次请求读取的最大文件数
	const maxFiles = 50
	if len(pathsSlice) > maxFiles {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Too many files requested. Maximum is %d files per request.", maxFiles),
				},
			},
			IsError: true,
		}, nil
	}

	// 处理每个文件
	var results []mcp.Content
	for _, path := range pathsSlice {
		// 处理空的或相对路径（如 "." 或 "./"），将其转换为绝对路径
		if path == "." || path == "./" {
			// 获取当前工作目录
			cwd, err := os.Getwd()
			if err != nil {
				results = append(results, mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error resolving current directory for path '%s': %v", path, err),
				})
				continue
			}
			path = cwd
		}

		// 验证路径是否在允许的目录内
		validPath, err := fs.validatePath(path)
		if err != nil {
			results = append(results, mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error with path '%s': %v", path, err),
			})
			continue
		}

		// 检查路径是否为目录
		info, err := os.Stat(validPath)
		if err != nil {
			results = append(results, mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error accessing '%s': %v", path, err),
			})
			continue
		}

		if info.IsDir() {
			// 对于目录，返回资源引用而不是内容
			resourceURI := pathToResourceURI(validPath)
			results = append(results, mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("'%s' is a directory. Use list_directory tool or resource URI: %s", path, resourceURI),
			})
			continue
		}

		// 确定 MIME 类型
		mimeType := detectMimeType(validPath)

		// 检查文件大小
		if info.Size() > MAX_INLINE_SIZE {
			// 文件太大，无法内联显示，返回资源引用
			resourceURI := pathToResourceURI(validPath)
			results = append(results, mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("File '%s' is too large to display inline (%d bytes). Access it via resource URI: %s",
					path, info.Size(), resourceURI),
			})
			continue
		}

		// 读取文件内容
		content, err := os.ReadFile(validPath)
		if err != nil {
			results = append(results, mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error reading file '%s': %v", path, err),
			})
			continue
		}

		// 添加文件头
		results = append(results, mcp.TextContent{
			Type: "text",
			Text: fmt.Sprintf("--- File: %s ---", path),
		})

		// 检查是否为文本文件
		if isTextFile(mimeType) {
			// 是文本文件，以文本形式返回内容
			results = append(results, mcp.TextContent{
				Type: "text",
				Text: string(content),
			})
		} else if isImageFile(mimeType) {
			// 是图像文件，以图像内容形式返回
			if info.Size() <= MAX_BASE64_SIZE {
				results = append(results, mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Image file: %s (%s, %d bytes)", path, mimeType, info.Size()),
				})
				results = append(results, mcp.ImageContent{
					Type:     "image",
					Data:     base64.StdEncoding.EncodeToString(content),
					MIMEType: mimeType,
				})
			} else {
				// 文件太大无法进行 Base64 编码，返回引用
				resourceURI := pathToResourceURI(validPath)
				results = append(results, mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Image file '%s' is too large to display inline (%d bytes). Access it via resource URI: %s",
						path, info.Size(), resourceURI),
				})
			}
		} else {
			// 是其他类型的二进制文件
			resourceURI := pathToResourceURI(validPath)

			if info.Size() <= MAX_BASE64_SIZE {
				// 文件足够小，可以进行 Base64 编码
				results = append(results, mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Binary file: %s (%s, %d bytes)", path, mimeType, info.Size()),
				})
				results = append(results, mcp.EmbeddedResource{
					Type: "resource",
					Resource: mcp.BlobResourceContents{
						URI:      resourceURI,
						MIMEType: mimeType,
						Blob:     base64.StdEncoding.EncodeToString(content),
					},
				})
			} else {
				// 文件太大无法进行 Base64 编码，返回引用
				results = append(results, mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Binary file '%s' (%s, %d bytes). Access it via resource URI: %s",
						path, mimeType, info.Size(), resourceURI),
				})
			}
		}
	}

	return &mcp.CallToolResult{
		Content: results,
	}, nil
}