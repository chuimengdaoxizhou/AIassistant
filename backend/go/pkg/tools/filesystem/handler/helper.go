package handler

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

// isPathInAllowedDirs 检查给定路径是否在任何允许的目录中。
func (fs *FilesystemHandler) isPathInAllowedDirs(path string) bool {
	// 确保路径是绝对且干净的
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// 添加结尾分隔符，以确保我们检查的是目录或目录内的文件，
	// 而不是前缀匹配（例如，/tmp/foo 不应匹配 /tmp/foobar）
	if !strings.HasSuffix(absPath, string(filepath.Separator)) {
		// 如果是文件，我们需要检查其目录
		if info, err := os.Stat(absPath); err == nil && !info.IsDir() {
			absPath = filepath.Dir(absPath) + string(filepath.Separator)
		} else {
			absPath = absPath + string(filepath.Separator)
		}
	}

	// 检查路径是否在任何允许的目录中
	for _, dir := range fs.allowedDirs {
		if strings.HasPrefix(absPath, dir) {
			return true
		}
	}
	return false
}

// validatePath 验证请求的路径是否有效且在允许的范围内。
func (fs *FilesystemHandler) validatePath(requestedPath string) (string, error) {
	// 首先总是转换为绝对路径
	abs, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// 检查路径是否在允许的目录内
	if !fs.isPathInAllowedDirs(abs) {
		return "", fmt.Errorf(
			"access denied - path outside allowed directories: %s",
			abs,
		)
	}

	// 处理符号链接
	realPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		// 对于新文件，检查父目录
		parent := filepath.Dir(abs)
		realParent, err := filepath.EvalSymlinks(parent)
		if err != nil {
			return "", fmt.Errorf("parent directory does not exist: %s", parent)
		}

		if !fs.isPathInAllowedDirs(realParent) {
			return "", fmt.Errorf(
				"access denied - parent directory outside allowed directories",
			)
		}
		return abs, nil
	}

	// 检查解析符号链接后的真实路径是否仍在允许的目录内
	if !fs.isPathInAllowedDirs(realPath) {
		return "", fmt.Errorf(
			"access denied - symlink target outside allowed directories",
		)
	}

	return realPath, nil
}

// detectMimeType 尝试确定文件的 MIME 类型。
func detectMimeType(path string) string {
	// 使用 mimetype 库进行更准确的检测
	mtype, err := mimetype.DetectFile(path)
	if err != nil {
		// 如果无法读取文件，则回退到基于扩展名的检测
		ext := filepath.Ext(path)
		if ext != "" {
			mimeType := mime.TypeByExtension(ext)
			if mimeType != "" {
				return mimeType
			}
		}
		return "application/octet-stream" // 默认值
	}

	return mtype.String()
}

// isTextFile 根据 MIME 类型确定文件是否可能是文本文件。
func isTextFile(mimeType string) bool {
	// 检查常见的文本 MIME 类型
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}

	// 常见的基于文本的应用程序类型
	textApplicationTypes := []string{
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/typescript",
		"application/x-typescript",
		"application/x-yaml",
		"application/yaml",
		"application/toml",
		"application/x-sh",
		"application/x-shellscript",
	}

	if slices.Contains(textApplicationTypes, mimeType) {
		return true
	}

	// 检查 +format 类型
	if strings.Contains(mimeType, "+xml") ||
		strings.Contains(mimeType, "+json") ||
		strings.Contains(mimeType, "+yaml") {
		return true
	}

	// 可能被错误识别的常见代码文件类型
	if strings.HasPrefix(mimeType, "text/x-") {
		return true
	}

	if strings.HasPrefix(mimeType, "application/x-") &&
		(strings.Contains(mimeType, "script") ||
			strings.Contains(mimeType, "source") ||
			strings.Contains(mimeType, "code")) {
		return true
	}

	return false
}

// isImageFile 根据 MIME 类型确定文件是否是图像。
func isImageFile(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/") ||
		(mimeType == "application/xml" && strings.HasSuffix(strings.ToLower(mimeType), ".svg"))
}
