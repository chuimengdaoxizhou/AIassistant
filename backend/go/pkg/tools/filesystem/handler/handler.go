package handler

import (
	"fmt"
	"os"
	"path/filepath"
)

// FilesystemHandler 负责处理所有与文件系统相关的工具请求。
// 它包含一个允许访问的目录列表，以确保操作的安全性。
type FilesystemHandler struct {
	allowedDirs []string // 允许访问的目录列表
}

// NewFilesystemHandler 创建一个新的 FilesystemHandler 实例。
// 它接收一个允许的目录列表，并对这些目录进行规范化和验证。
func NewFilesystemHandler(allowedDirs []string) (*FilesystemHandler, error) {
	// 规范化和验证目录
	normalized := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		// 获取绝对路径
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %s: %w", dir, err)
		}

		// 检查路径是否存在且为目录
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to access directory %s: %w",
				abs,
				err,
			)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("path is not a directory: %s", abs)
		}

		// 确保路径以分隔符结尾，以防止前缀匹配问题
		// 例如, /tmp/foo 不应匹配 /tmp/foobar
		normalized = append(normalized, filepath.Clean(abs)+string(filepath.Separator))
	}
	return &FilesystemHandler{
		allowedDirs: normalized,
	}, nil
}

// pathToResourceURI 将文件路径转换为资源 URI。
func pathToResourceURI(path string) string {
	return "file://" + path
}
