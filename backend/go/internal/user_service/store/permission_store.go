package store

import (
	"Jarvis_2.0/backend/go/internal/models"
	"gorm.io/gorm"
)

// Store 封装了所有与用户服务相关的数据库操作。
type Store struct {
	DB *gorm.DB
}

// NewStore 创建一个新的 Store 实例。
func NewStore(db *gorm.DB) *Store {
	return &Store{DB: db}
}

// --- Role Management ---

// CreateRole 在数据库中创建一个新的角色。
func (s *Store) CreateRole(role *models.AuthRole) error {
	return s.DB.Create(role).Error
}

// GetRoleByName 通过名称查找角色。
func (s *Store) GetRoleByName(name string) (*models.AuthRole, error) {
	var role models.AuthRole
	if err := s.DB.Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// DeleteRole 从数据库中删除一个角色。
func (s *Store) DeleteRole(roleID uint) error {
	return s.DB.Delete(&models.AuthRole{}, roleID).Error
}

// --- Permission Management ---

// CreatePermission 在数据库中创建一个新的权限。
func (s *Store) CreatePermission(permission *models.Permission) error {
	return s.DB.Create(permission).Error
}

// DeletePermission 从数据库中删除一个权限。
func (s Store) DeletePermission(permissionID uint) error {
	return s.DB.Delete(&models.Permission{}, permissionID).Error
}

// --- Role-Permission Association ---

// AddPermissionToRole 为角色添加一个权限。
func (s *Store) AddPermissionToRole(roleID, permissionID uint) error {
	role := &models.AuthRole{Model: gorm.Model{ID: roleID}}
	permission := &models.Permission{Model: gorm.Model{ID: permissionID}}
	return s.DB.Model(role).Association("Permissions").Append(permission)
}

// RemovePermissionFromRole 从角色中移除一个权限。
func (s *Store) RemovePermissionFromRole(roleID, permissionID uint) error {
	role := &models.AuthRole{Model: gorm.Model{ID: roleID}}
	permission := &models.Permission{Model: gorm.Model{ID: permissionID}}
	return s.DB.Model(role).Association("Permissions").Delete(permission)
}

// --- User-Role Association ---

// AssignRoleToUser 为用户分配一个角色。
func (s *Store) AssignRoleToUser(userID, roleID uint) error {
	user := &models.User{Model: gorm.Model{ID: userID}}
	role := &models.AuthRole{Model: gorm.Model{ID: roleID}}
	return s.DB.Model(user).Association("Roles").Append(role)
}

// RevokeRoleFromUser 从用户中撤销一个角色。
func (s *Store) RevokeRoleFromUser(userID, roleID uint) error {
	user := &models.User{Model: gorm.Model{ID: userID}}
	role := &models.AuthRole{Model: gorm.Model{ID: roleID}}
	return s.DB.Model(user).Association("Roles").Delete(role)
}

// GetUserRoles 获取一个用户的所有角色。
func (s *Store) GetUserRoles(userID uint) ([]*models.AuthRole, error) {
	var user models.User
	if err := s.DB.Preload("Roles").First(&user, userID).Error; err != nil {
		return nil, err
	}
	return user.Roles, nil
}

// GetUserPermissions 获取一个用户的所有权限（通过其角色）。
func (s *Store) GetUserPermissions(userID uint) ([]*models.Permission, error) {
	var user models.User
	// Preload Roles and their associated Permissions
	if err := s.DB.Preload("Roles.Permissions").First(&user, userID).Error; err != nil {
		return nil, err
	}

	// Use a map to collect unique permissions
	permissionMap := make(map[uint]*models.Permission)
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			permissionMap[perm.ID] = perm
		}
	}

	// Convert map to slice
	permissions := make([]*models.Permission, 0, len(permissionMap))
	for _, perm := range permissionMap {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}
