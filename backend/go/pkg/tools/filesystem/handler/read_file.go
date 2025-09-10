package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleReadFile 处理读取文件内容的工具请求。
func (fs *FilesystemHandler) HandleReadFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 提取文件路径参数
	path, err := request.RequireString("path")
	if err != nil {
		return nil, err
	}

	// 处理空的或相对路径（如 "." 或 "./"），将其转换为绝对路径
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

	// 验证路径是否在允许的目录内
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

	// 检查路径是否为目录
	info, err := os.Stat(validPath)
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

	if info.IsDir() {
		// 对于目录，返回资源引用而不是内容
		resourceURI := pathToResourceURI(validPath)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("This is a directory. Use the resource URI to browse its contents: %s", resourceURI),
				},
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

	// 确定 MIME 类型
	mimeType := detectMimeType(validPath)

	// 检查文件大小
	if info.Size() > MAX_INLINE_SIZE {
		// 文件太大，无法内联显示，返回资源引用
		resourceURI := pathToResourceURI(validPath)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("File is too large to display inline (%d bytes). Access it via resource URI: %s", info.Size(), resourceURI),
				},
				mcp.EmbeddedResource{
					Type: "resource",
					Resource: mcp.TextResourceContents{
						URI:      resourceURI,
						MIMEType: "text/plain",
						Text:     fmt.Sprintf("Large file: %s (%s, %d bytes)", validPath, mimeType, info.Size()),
					},
				},
			},
		}, nil
	}

	// 读取文件内容
	content, err := os.ReadFile(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error reading file: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 检查是否为文本文件
	if isTextFile(mimeType) {
		// 是文本文件，以文本形式返回内容
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(content),
				},
			},
		}, nil
	} else if isImageFile(mimeType) {
		// 是图像文件，以图像内容形式返回
		if info.Size() <= MAX_BASE64_SIZE {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Image file: %s (%s, %d bytes)", validPath, mimeType, info.Size()),
					},
					mcp.ImageContent{
						Type:     "image",
						Data:     base64.StdEncoding.EncodeToString(content),
						MIMEType: mimeType,
					},
				},
			}, nil
		} else {
			// 文件太大无法进行 Base64 编码，返回引用
			resourceURI := pathToResourceURI(validPath)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Image file is too large to display inline (%d bytes). Access it via resource URI: %s", info.Size(), resourceURI),
					},
					mcp.EmbeddedResource{
						Type: "resource",
						Resource: mcp.TextResourceContents{
							URI:      resourceURI,
							MIMEType: "text/plain",
							Text:     fmt.Sprintf("Large image: %s (%s, %d bytes)", validPath, mimeType, info.Size()),
						},
					},
				},
			}, nil
		}
	} else {
		// 是其他类型的二进制文件
		resourceURI := pathToResourceURI(validPath)

		if info.Size() <= MAX_BASE64_SIZE {
			// 文件足够小，可以进行 Base64 编码
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Binary file: %s (%s, %d bytes)", validPath, mimeType, info.Size()),
					},
					mcp.EmbeddedResource{
						Type: "resource",
						Resource: mcp.BlobResourceContents{
							URI:      resourceURI,
							MIMEType: mimeType,
							Blob:     base64.StdEncoding.EncodeToString(content),
						},
					},
				},
			}, nil
		} else {
			// 文件太大无法进行 Base64 编码，返回引用
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Binary file: %s (%s, %d bytes). Access it via resource URI: %s", validPath, mimeType, info.Size(), resourceURI),
					},
					mcp.EmbeddedResource{
						Type: "resource",
						Resource: mcp.TextResourceContents{
							URI:      resourceURI,
							MIMEType: "text/plain",
							Text:     fmt.Sprintf("Binary file: %s (%s, %d bytes)", validPath, mimeType, info.Size()),
						},
					},
				},
			}, nil
		}
	}
}
