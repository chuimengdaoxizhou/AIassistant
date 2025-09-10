package handler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/djherbis/times"       // 用于获取文件的创建时间、访问时间等额外信息
	"github.com/mark3labs/mcp-go/mcp" // MCP 协议支持
)

// HandleGetFileInfo 方法：处理获取文件信息的请求
func (fs *FilesystemHandler) HandleGetFileInfo(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 从请求中获取路径参数
	path, err := request.RequireString("path")
	if err != nil {
		return nil, err
	}

	// 如果路径是 "." 或 "./"，则转换为绝对路径（当前工作目录）
	if path == "." || path == "./" {
		// 获取当前工作目录
		cwd, err := os.Getwd()
		if err != nil {
			// 获取失败，返回错误信息
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

	// 校验路径是否合法（防止目录穿越或访问非法路径）
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

	// 获取文件的详细信息
	info, err := fs.getFileStats(validPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error getting file info: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 判断 MIME 类型（目录固定为 "directory"，文件则尝试探测）
	mimeType := "directory"
	if info.IsFile {
		mimeType = detectMimeType(validPath)
	}

	// 生成资源 URI（统一标识文件）
	resourceURI := pathToResourceURI(validPath)

	// 判断文件类型文本（目录/文件）
	var fileTypeText string
	if info.IsDirectory {
		fileTypeText = "Directory"
	} else {
		fileTypeText = "File"
	}

	// 返回结果（包含详细信息 + 资源内容）
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			// 详细文本信息
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf(
					"File information for: %s\n\nSize: %d bytes\nCreated: %s\nModified: %s\nAccessed: %s\nIsDirectory: %v\nIsFile: %v\nPermissions: %s\nMIME Type: %s\nResource URI: %s",
					validPath,
					info.Size,
					info.Created.Format(time.RFC3339),
					info.Modified.Format(time.RFC3339),
					info.Accessed.Format(time.RFC3339),
					info.IsDirectory,
					info.IsFile,
					info.Permissions,
					mimeType,
					resourceURI,
				),
			},
			// 资源嵌入信息（适合机器处理）
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text: fmt.Sprintf("%s: %s (%s, %d bytes)",
						fileTypeText,
						validPath,
						mimeType,
						info.Size),
				},
			},
		},
	}, nil
}

// getFileStats 方法：获取文件的详细信息（大小、时间戳、权限等）
func (fs *FilesystemHandler) getFileStats(path string) (FileInfo, error) {
	// 调用 os.Stat 获取基本文件信息（大小、是否目录、权限等）
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}

	// 使用 times 库获取文件的访问时间、修改时间、创建时间
	timespec, err := times.Stat(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to get file times: %w", err)
	}

	// 初始化创建时间（有些系统不支持获取创建时间）
	createdTime := time.Time{}
	if timespec.HasBirthTime() {
		createdTime = timespec.BirthTime()
	}

	// 封装文件信息并返回
	return FileInfo{
		Size:        info.Size(),                           // 文件大小（字节）
		Created:     createdTime,                           // 创建时间
		Modified:    timespec.ModTime(),                    // 修改时间
		Accessed:    timespec.AccessTime(),                 // 最后访问时间
		IsDirectory: info.IsDir(),                          // 是否目录
		IsFile:      !info.IsDir(),                         // 是否文件
		Permissions: fmt.Sprintf("%o", info.Mode().Perm()), // 权限（八进制字符串，如 755）
	}, nil
}
