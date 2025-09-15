package loaders

import (
	"Jarvis_2.0/backend/go/internal/rag_service/rag/interfaces"
	"Jarvis_2.0/backend/go/internal/rag_service/rag/schema"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"baliance.com/gooxml/document"
	"github.com/google/uuid"
)

// DocxLoader 实现了用于读取 Word (.docx) 文件的 Loader 接口。
type DocxLoader struct{}

// NewDocxLoader 创建一个新的 DocxLoader。
func NewDocxLoader() *DocxLoader {
	return &DocxLoader{}
}

// Load 读取一个 .docx 文件，提取其文本和图片，并返回一个包含所有内容的 Document。
func (l *DocxLoader) Load(ctx context.Context, path string) ([]*schema.Document, error) {
	doc, err := document.Open(path)
	if err != nil {
		return nil, err
	}
	// 提取所有段落的文本内容
	var textBuilder strings.Builder
	for _, p := range doc.Paragraphs() {
		for _, r := range p.Runs() {
			textBuilder.WriteString(r.Text())
		}
		textBuilder.WriteString("\n")
	}

	// 提取所有图片数据
	var imagesData [][]byte
	for _, imgRef := range doc.Images {
		// 1. 从 ImageRef 对象获取图片的临时文件在磁盘上的路径
		tempImagePath := imgRef.Path()
		if tempImagePath == "" {
			continue
		}

		// 2. 使用标准库 os.Open 打开这个临时文件
		file, err := os.Open(tempImagePath)
		if err != nil {
			// 如果打开文件失败，则跳过这张图片
			continue
		}

		// 3. 读取文件的所有二进制数据
		data, readErr := io.ReadAll(file)

		// 4. 关键：必须立即关闭文件句柄，防止资源泄漏
		file.Close()

		// 5. 如果在读取过程中出错，也跳过这张图片
		if readErr != nil {
			continue
		}

		imagesData = append(imagesData, data)
	}

	// 创建并填充 Document 结构体
	docResult := &schema.Document{
		ID:   uuid.New().String(),
		Text: textBuilder.String(),
		Metadata: map[string]interface{}{
			schema.MetadataKeyFileName: filepath.Base(path),
		},
	}

	// 如果提取到了图片，将其二进制数据存入元数据中
	if len(imagesData) > 0 {
		docResult.Metadata[schema.MetadataKeyImage] = imagesData
	}

	return []*schema.Document{docResult}, nil
}

// 编译时检查，确保 DocxLoader 实现了 Loader 接口
var _ interfaces.Loader = (*DocxLoader)(nil)
