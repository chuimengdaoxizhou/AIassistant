package handler

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleCopyFile 处理复制文件或目录的工具请求。
func (fs *FilesystemHandler) HandleCopyFile(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 提取源和目标路径参数
	source, err := request.RequireString("source")
	if err != nil {
		return nil, err
	}
	destination, err := request.RequireString("destination")
	if err != nil {
		return nil, err
	}

	// 处理空的或相对的源路径
	if source == "." || source == "./" {
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
		source = cwd
	}
	// 处理空的或相对的目标路径
	if destination == "." || destination == "./" {
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
		destination = cwd
	}

	// 验证源路径
	validSource, err := fs.validatePath(source)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error with source path: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 检查源是否存在
	srcInfo, err := os.Stat(validSource)
	if os.IsNotExist(err) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Source does not exist: %s", source),
				},
			},
			IsError: true,
		}, nil
	} else if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error accessing source: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 验证目标路径
	validDest, err := fs.validatePath(destination)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error with destination path: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 如果目标路径的父目录不存在，则创建它
	destDir := filepath.Dir(validDest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error creating destination directory: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 根据源是文件还是目录执行复制操作
	if srcInfo.IsDir() {
		// 是目录，递归复制
		if err := copyDir(validSource, validDest); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error copying directory: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
	} else {
		// 是文件，直接复制
		if err := copyFile(validSource, validDest); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error copying file: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
	}

	// 创建响应
	resourceURI := pathToResourceURI(validDest)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf(
					"Successfully copied %s to %s",
					source,
					destination,
				),
			},
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "text/plain",
					Text:     fmt.Sprintf("Copied file: %s", validDest),
				},
			},
		},
	}, nil
}

// copyFile 将单个文件从 src 复制到 dst。
func copyFile(src, dst string) error {
	// 打开源文件
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// 创建目标文件
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// 复制内容
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// 获取源文件模式
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// 在目标文件上设置相同的文件模式
	return os.Chmod(dst, sourceInfo.Mode())
}

// copyDir 递归地将目录树从 src 复制到 dst。
func copyDir(src, dst string) error {
	// 获取源目录的属性
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// 使用相同的权限创建目标目录
	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// 读取目录条目
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// 处理符号链接
		if entry.Type()&os.ModeSymlink != 0 {
			// 为简单起见，我们在此实现中跳过符号链接
			continue
		}

		// 递归复制子目录或复制文件
		if entry.IsDir() {
			if err = copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err = copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
