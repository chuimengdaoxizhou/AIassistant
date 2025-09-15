package dal

import (
	"context"
	"errors"

	"Jarvis_2.0/backend/go/internal/models"
	"gorm.io/gorm"
)

// FolderDAL provides data access methods for RAG folders.
type FolderDAL struct {
	db *gorm.DB
}

// NewFolderDAL creates a new FolderDAL.
func NewFolderDAL(db *gorm.DB) *FolderDAL {
	return &FolderDAL{db: db}
}

// CreateFolder creates a new folder for a specific user.
// It returns an error if a folder with the same name already exists for that user.
func (dal *FolderDAL) CreateFolder(ctx context.Context, userID, folderName string) (*models.RagFolder, error) {
	folder := &models.RagFolder{
		UserID: userID,
		Name:   folderName,
	}

	result := dal.db.WithContext(ctx).Create(folder)
	if result.Error != nil {
		// Handle potential unique constraint violation
		return nil, result.Error
	}

	return folder, nil
}

// ListFoldersByUser retrieves all folders for a given user.
func (dal *FolderDAL) ListFoldersByUser(ctx context.Context, userID string) ([]*models.RagFolder, error) {
	var folders []*models.RagFolder
	result := dal.db.WithContext(ctx).Where("user_id = ?", userID).Find(&folders)
	if result.Error != nil {
		return nil, result.Error
	}
	return folders, nil
}

// DeleteFolder deletes a folder for a specific user by its ID.
// It ensures that the user owns the folder before deleting.
func (dal *FolderDAL) DeleteFolder(ctx context.Context, userID string, folderID uint) error {
	result := dal.db.WithContext(ctx).Where("user_id = ? AND id = ?", userID, folderID).Delete(&models.RagFolder{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("folder not found or user does not have permission to delete it")
	}

	return nil
}
