package store

import (
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/pkg/logger"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	// defaultBucket 定义了用于存储 agent 生成的文件的 MinIO 存储桶名称。
	defaultBucket = "jarvis-agent-files"
)

// ContentProcessor 负责处理和存储多模态内容。
// 它遍历任务结果，并将需要持久化存储的文件（如视频、音频、大型文档）上传到 MinIO。
type ContentProcessor struct {
	minioClient *minio.Client
	logger      *logger.Logger
}

// NewContentProcessor 创建一个新的 ContentProcessor 实例。
// 它需要一个 MinIO 客户端和一个日志记录器。
func NewContentProcessor(minioClient *minio.Client, logger *logger.Logger) *ContentProcessor {
	return &ContentProcessor{
		minioClient: minioClient,
		logger:      logger,
	}
}

// ProcessAndStoreContent 遍历内容切片，处理需要外部存储的部分（如 FileData），
// 并返回一个更新了 URI 的新内容切片，使其适合存储在 MongoDB 中。
//
// ctx: 上下文，用于控制上传操作的取消或超时。
// contents: 从 agent 执行返回的原始内容切片。
// 返回值:
//   - []models.Content: 处理后的内容切片，其中 FileData 的 URI 已被替换为 MinIO 的对象路径。
//   - error: 如果在处理或上传过程中发生错误，则返回 error。
func (p *ContentProcessor) ProcessAndStoreContent(ctx context.Context, contents []models.Content) ([]models.Content, error) {
	// 创建一个新切片以存储处理后的内容，避免修改原始输入。
	processedContents := make([]models.Content, 0, len(contents))

	for _, content := range contents {
		processedParts := make([]*models.Part, 0, len(content.Parts))
		for _, part := range content.Parts {
			// 创建 part 的深拷贝以进行修改
			newPart := *part

			// 只处理包含 FileData 且 FileURI 是本地文件路径的 Part
			if newPart.FileData != nil && isLocalFilePath(newPart.FileData.FileURI) {
				p.logger.Info(fmt.Sprintf("Processing FileData with local URI: %s", newPart.FileData.FileURI))

				// 上传文件到 MinIO 并获取新的对象路径
				objectName, err := p.uploadFileToMinio(ctx, newPart.FileData.FileURI, newPart.FileData.MIMEType)
				if err != nil {
					p.logger.WithError(models.ErrorInfo{Message: err.Error()}).Error("Failed to upload file to MinIO")
					// 返回错误，让调用者决定如何处理（例如，将任务标记为失败）
					return nil, fmt.Errorf("failed to upload file '%s' to MinIO: %w", newPart.FileData.FileURI, err)
				}

				p.logger.Info(fmt.Sprintf("Successfully uploaded file to MinIO. Object name: %s", objectName))

				// 更新 FileData 的 URI 为 MinIO 中的对象名称
				newPart.FileData.FileURI = objectName
			}
			processedParts = append(processedParts, &newPart)
		}
		// 使用处理过的 parts 创建新的 content
		processedContents = append(processedContents, models.Content{
			Role:  content.Role,
			Parts: processedParts,
		})
	}

	return processedContents, nil
}

// uploadFileToMinio 负责将单个文件上传到 MinIO。
func (p *ContentProcessor) uploadFileToMinio(ctx context.Context, localFilePath, contentType string) (string, error) {
	// 1. 确保存储桶存在
	err := p.ensureBucketExists(ctx, defaultBucket)
	if err != nil {
		return "", err
	}

	// 2. 打开本地文件
	file, err := os.Open(localFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open local file %s: %w", localFilePath, err)
	}
	defer file.Close()

	// 3. 获取文件状态以确定文件大小
	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file stats for %s: %w", localFilePath, err)
	}

	// 4. 生成一个唯一的对象名称，以避免冲突
	// 格式: <uuid>.<original_extension>
	extension := filepath.Ext(localFilePath)
	objectName := fmt.Sprintf("%s%s", uuid.New().String(), extension)

	// 5. 上传文件
	_, err = p.minioClient.PutObject(ctx, defaultBucket, objectName, file, stat.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to put object to MinIO: %w", err)
	}

	// 6. 返回在 MinIO 中的对象名称（路径）
	return objectName, nil
}

// ensureBucketExists 检查指定的存储桶是否存在，如果不存在则创建它。
func (p *ContentProcessor) ensureBucketExists(ctx context.Context, bucketName string) error {
	found, err := p.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket '%s' exists: %w", bucketName, err)
	}
	if !found {
		p.logger.Info(fmt.Sprintf("Bucket '%s' not found, creating it.", bucketName))
		err = p.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket '%s': %w", bucketName, err)
		}
	}
	return nil
}

// isLocalFilePath 检查给定的 URI 是否是一个本地文件路径。
// 一个简单的检查是查看它是否缺少 URL scheme。
func isLocalFilePath(uri string) bool {
	// 如果 URI 可以被解析为一个带有 scheme（如 http, https, s3）的 URL，那么它不是一个本地文件路径。
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return true // 如果解析失败，可能是一个本地路径
	}
	// 拥有 scheme 的通常不是本地文件路径，除非是 "file://"
	if parsedURL.Scheme != "" && parsedURL.Scheme != "file" {
		return false
	}

	// 进一步检查，以防 "file://" URI
	if parsedURL.Scheme == "file" {
		uri = parsedURL.Path
	}

	// 检查路径是否以 "/" 或 "." (Windows) 开头，或者不包含 "://"
	// 这是一个启发式方法，可能需要根据具体情况调整
	return !strings.Contains(uri, "://") || strings.HasPrefix(uri, "/") || filepath.IsAbs(uri)
}