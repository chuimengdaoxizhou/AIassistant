package store

import (
	"Jarvis_2.0/backend/go/internal/models"
	"errors"

	"gorm.io/gorm"
)

// --- User Management ---

// CreateUser 在数据库中创建一个新用户，并为其分配一个默认角色。
func (s *Store) CreateUser(user *models.User) error {
	// 在事务中执行创建用户和分配角色的操作，确保原子性。
	return s.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 创建用户
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// 2. 查找默认的 "Member" 角色
		var defaultRole models.AuthRole
		if err := tx.Where("name = ?", "Member").First(&defaultRole).Error; err != nil {
			// 如果默认角色不存在，可以返回错误，或者先创建它。
			// 这里我们假设 `init.sql` 已经或将会创建这个角色。
			return errors.New("默认的 'Member' 角色未找到")
		}

		// 3. 为用户分配角色
		return tx.Model(user).Association("Roles").Append(&defaultRole)
	})
}

// GetUserByEmail 通过邮箱地址查找用户。
func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	// Preload Roles to get role information along with the user
	if err := s.DB.Preload("Roles").Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID 通过 ID 查找用户。
func (s *Store) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.DB.Preload("Roles").First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByProviderID 通过 OAuth 提供商和其提供的用户 ID 查找用户。
// 这是实现 OAuth 登录的关键。
func (s *Store) GetUserByProviderID(provider, providerID string) (*models.User, error) {
	var user models.User
	if err := s.DB.Preload("Roles").Where("provider = ? AND provider_id = ?", provider, providerID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户信息。
func (s *Store) UpdateUser(user *models.User) error {
	return s.DB.Save(user).Error
}
