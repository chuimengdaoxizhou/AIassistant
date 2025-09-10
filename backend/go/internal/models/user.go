package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UserStatus 定义了用户账户的生命周期状态。
type UserStatus string

const (
	StatusPending     UserStatus = "pending"     // 账号待激活或验证
	StatusActive      UserStatus = "active"      // 账号正常
	StatusSuspended   UserStatus = "suspended"   // 账号被暂停
	StatusDeactivated UserStatus = "deactivated" // 账号已停用
)

// --- RBAC 模型 ---

// Permission 代表一个可以被执行的具体操作权限。
type Permission struct {
	gorm.Model
	Name        string `gorm:"unique;not null;size:255"` // 权限标识，例如 "articles:create", "users:read"
	Description string `gorm:"size:1024"`                // 权限的详细描述
}

// AuthRole 代表一组权限的集合。
type AuthRole struct {
	gorm.Model
	Name        string        `gorm:"unique;not null;size:255"` // 角色名称，例如 "Admin", "VIP", "Member"
	Description string        `gorm:"size:1024"`                // 角色的详细描述
	Permissions []*Permission `gorm:"many2many:role_permissions;"`
}

// User 代表系统中的一个用户账户。
type User struct {
	gorm.Model

	Username  string `gorm:"unique;not null"`
	FullName  string `gorm:"size:255"`
	Email     string `gorm:"uniqueIndex;not null"`
	Password  string `gorm:"size:255" json:"-"` // 存储哈希后的密码，json中忽略
	AvatarURL string

	Provider   string `gorm:"not null"`
	ProviderID string `gorm:"index:idx_provider_id,unique;not null"`

	Status      UserStatus `gorm:"type:varchar(20);default:'pending';not null"`
	LastLoginAt *time.Time
	TenantID    uint `gorm:"index"`
	Settings    datatypes.JSON

	// RBAC 关系: 一个用户可以拥有多个角色
	Roles []*AuthRole `gorm:"many2many:user_roles;"`
}

// --- 自定义表名 ---

func (User) TableName() string {
	return "users"
}

func (AuthRole) TableName() string {
	return "roles"
}

func (Permission) TableName() string {
	return "permissions"
}