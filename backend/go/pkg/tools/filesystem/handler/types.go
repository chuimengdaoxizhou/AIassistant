package handler

import "time"

const (
	// MAX_INLINE_SIZE 内联内容的最大大小 (5MB)
	// 超过此大小的文件内容通常不会直接嵌入返回结果中
	MAX_INLINE_SIZE = 5 * 1024 * 1024

	// MAX_BASE64_SIZE base64 编码内容的最大大小 (1MB)
	// 用于限制通过 base64 返回的文件内容，避免占用过多内存
	MAX_BASE64_SIZE = 1 * 1024 * 1024

	// MAX_SEARCH_RESULTS 搜索结果的最大数量
	// 防止搜索返回过多结果导致输出过大或性能下降
	MAX_SEARCH_RESULTS = 1000

	// MAX_SEARCHABLE_SIZE 可搜索文件的最大大小 (10MB)
	// 超过此大小的文件不会被搜索以避免性能问题
	MAX_SEARCHABLE_SIZE = 10 * 1024 * 1024
)

// FileInfo 文件信息结构体
type FileInfo struct {
	Size        int64     `json:"size"`        // 文件大小（字节）
	Created     time.Time `json:"created"`     // 文件创建时间
	Modified    time.Time `json:"modified"`    // 文件最后修改时间
	Accessed    time.Time `json:"accessed"`    // 文件最后访问时间
	IsDirectory bool      `json:"isDirectory"` // 是否为目录
	IsFile      bool      `json:"isFile"`      // 是否为普通文件
	Permissions string    `json:"permissions"` // 文件权限字符串（如 rwxr-xr-x）
}

// FileNode 表示目录树中的一个节点
type FileNode struct {
	Name     string      `json:"name"`               // 文件或目录名称
	Path     string      `json:"path"`               // 文件或目录完整路径
	Type     string      `json:"type"`               // 类型："file" 或 "directory"
	Size     int64       `json:"size,omitempty"`     // 文件大小（仅文件有效）
	Modified time.Time   `json:"modified,omitempty"` // 最后修改时间
	Children []*FileNode `json:"children,omitempty"` // 子节点（仅目录有效）
}

// SearchResult 表示在文件中匹配到的单个搜索结果
type SearchResult struct {
	FilePath    string // 文件路径
	LineNumber  int    // 匹配行号
	LineContent string // 匹配行内容
	ResourceURI string // 文件资源 URI，用于前端展示或下载
}
