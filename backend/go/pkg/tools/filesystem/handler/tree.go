package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleTree 生成指定目录的树状结构
func (fs *FilesystemHandler) HandleTree(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// 获取路径参数
	path, err := request.RequireString("path")
	if err != nil {
		return nil, err
	}

	// 将相对路径 "." 或 "./" 转换为绝对路径
	if path == "." || path == "./" {
		cwd, err := os.Getwd() // 获取当前工作目录
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

	// 可选参数：目录树深度，默认 3
	depth := 3
	if depthParam, err := request.RequireFloat("depth"); err == nil {
		depth = int(depthParam)
	}

	// 可选参数：是否跟随符号链接（symlink），默认 false
	followSymlinks := false
	if followParam, err := request.RequireBool("follow_symlinks"); err == nil {
		followSymlinks = followParam
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
					Text: "错误: 指定路径不是目录",
				},
			},
			IsError: true,
		}, nil
	}

	// 构建目录树
	tree, err := fs.buildTree(validPath, depth, 0, followSymlinks)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("生成目录树出错: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 将目录树转换为 JSON 格式
	jsonData, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("生成 JSON 出错: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// 生成资源 URI
	resourceURI := pathToResourceURI(validPath)

	// 返回目录树结果
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("目录树 %s (最大深度: %d):\n\n%s", validPath, depth, string(jsonData)),
			},
			mcp.EmbeddedResource{
				Type: "resource",
				Resource: mcp.TextResourceContents{
					URI:      resourceURI,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		},
	}, nil
}

// buildTree 递归生成指定路径的目录树结构
func (fs *FilesystemHandler) buildTree(path string, maxDepth int, currentDepth int, followSymlinks bool) (*FileNode, error) {
	// 验证路径是否合法
	validPath, err := fs.validatePath(path)
	if err != nil {
		return nil, err
	}

	// 获取文件信息
	info, err := os.Stat(validPath)
	if err != nil {
		return nil, err
	}

	// 创建节点
	node := &FileNode{
		Name:     filepath.Base(validPath), // 文件或目录名
		Path:     validPath,                // 完整路径
		Modified: info.ModTime(),           // 最后修改时间
	}

	// 判断类型并设置
	if info.IsDir() {
		node.Type = "directory" // 目录类型

		// 如果当前深度小于最大深度，继续处理子节点
		if currentDepth < maxDepth {
			entries, err := os.ReadDir(validPath) // 读取目录条目
			if err != nil {
				return nil, err
			}

			for _, entry := range entries {
				entryPath := filepath.Join(validPath, entry.Name())

				// 处理符号链接
				if entry.Type()&os.ModeSymlink != 0 {
					if !followSymlinks {
						// 不跟随 symlink，则跳过
						continue
					}

					// 解析符号链接目标
					linkDest, err := filepath.EvalSymlinks(entryPath)
					if err != nil {
						continue // 无效 symlink 跳过
					}

					// 确保 symlink 目标在允许目录内
					if !fs.isPathInAllowedDirs(linkDest) {
						continue
					}

					entryPath = linkDest
				}

				// 递归构建子节点
				childNode, err := fs.buildTree(entryPath, maxDepth, currentDepth+1, followSymlinks)
				if err != nil {
					continue // 出错则跳过
				}

				// 添加子节点到当前节点
				node.Children = append(node.Children, childNode)
			}
		}
	} else {
		node.Type = "file"      // 文件类型
		node.Size = info.Size() // 文件大小
	}

	return node, nil
}
