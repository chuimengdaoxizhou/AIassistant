package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleReadResource 处理 MCP 的资源读取功能
func (fs *FilesystemHandler) HandleReadResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) ([]mcp.ResourceContents, error) {
	uri := request.Params.URI

	// 判断 URI 是否是 file:// 协议
	if !strings.HasPrefix(uri, "file://") {
		return nil, fmt.Errorf("不支持的 URI 协议: %s", uri)
	}

	// 去掉 file:// 前缀，获取真实文件路径
	path := strings.TrimPrefix(uri, "file://")

	// 验证路径是否合法（是否在允许的目录范围内）
	validPath, err := fs.validatePath(path)
	if err != nil {
		return nil, err
	}

	// 获取文件信息
	fileInfo, err := os.Stat(validPath)
	if err != nil {
		return nil, err
	}

	// 如果是目录，则返回目录内容列表
	if fileInfo.IsDir() {
		entries, err := os.ReadDir(validPath)
		if err != nil {
			return nil, err
		}

		var result strings.Builder
		result.WriteString(fmt.Sprintf("目录内容: %s\n\n", validPath))

		// 遍历目录下的文件和子目录
		for _, entry := range entries {
			entryPath := filepath.Join(validPath, entry.Name())
			entryURI := pathToResourceURI(entryPath)

			if entry.IsDir() {
				// 子目录
				result.WriteString(fmt.Sprintf("[DIR]  %s (%s)\n", entry.Name(), entryURI))
			} else {
				// 普通文件，显示大小
				info, err := entry.Info()
				if err == nil {
					result.WriteString(fmt.Sprintf("[FILE] %s (%s) - %d bytes\n",
						entry.Name(), entryURI, info.Size()))
				} else {
					// 如果获取不到大小，直接返回文件名
					result.WriteString(fmt.Sprintf("[FILE] %s (%s)\n", entry.Name(), entryURI))
				}
			}
		}

		// 返回目录内容作为文本
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "text/plain",
				Text:     result.String(),
			},
		}, nil
	}

	// 如果是文件，先检测 MIME 类型
	mimeType := detectMimeType(validPath)

	// 检查文件大小是否超过最大内联限制
	if fileInfo.Size() > MAX_INLINE_SIZE {
		// 太大，不能直接返回内容，给出提示
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "text/plain",
				Text:     fmt.Sprintf("文件过大，无法直接显示内容 (%d bytes)。请使用 read_file 工具读取指定部分。", fileInfo.Size()),
			},
		}, nil
	}

	// 读取文件内容
	content, err := os.ReadFile(validPath)
	if err != nil {
		return nil, err
	}

	// 判断是否是文本文件
	if isTextFile(mimeType) {
		// 文本文件，直接返回内容
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: mimeType,
				Text:     string(content),
			},
		}, nil
	} else {
		// 二进制文件
		if fileInfo.Size() <= MAX_BASE64_SIZE {
			// 如果大小合适，就进行 Base64 编码后返回
			return []mcp.ResourceContents{
				mcp.BlobResourceContents{
					URI:      uri,
					MIMEType: mimeType,
					Blob:     base64.StdEncoding.EncodeToString(content),
				},
			}, nil
		} else {
			// 太大，无法直接 Base64 返回，只返回提示信息
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      uri,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("二进制文件 (%s, %d bytes)。请使用 read_file 工具读取指定部分。", mimeType, fileInfo.Size()),
				},
			}, nil
		}
	}
}
