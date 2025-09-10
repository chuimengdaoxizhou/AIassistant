package service

import (
	"Jarvis_2.0/backend/go/internal/models"
	"Jarvis_2.0/backend/go/internal/user_service/store"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
	"time"
)

// Service 封装了业务逻辑。
type Service struct {
	store     *store.Store
	jwtSecret []byte
}

// NewService 创建一个新的 Service 实例。
func NewService(s *store.Store, jwtSecret string) *Service {
	return &Service{
		store:     s,
		jwtSecret: []byte(jwtSecret),
	}
}

// --- User Registration & Login ---

// RegisterUserByEmail 处理新用户通过邮箱注册的逻辑。
func (s *Service) RegisterUserByEmail(email, password, username, fullName string) (*models.User, error) {
	// 检查用户是否已存在
	_, err := s.store.GetUserByEmail(email)
	if err == nil {
		return nil, errors.New("该邮箱已被注册")
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	// 创建用户模型
	user := &models.User{
		Username:   username,
		FullName:   fullName,
		Email:      email,
		Provider:   "email",
		ProviderID: email,
		Status:     models.StatusActive,
		Password:   string(hashedPassword),
	}

	if err := s.store.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

// LoginUserByEmail 处理用户通过邮箱登录的逻辑。
func (s *Service) LoginUserByEmail(email, password string) (string, error) {
	user, err := s.store.GetUserByEmail(email)
	if err != nil {
		return "", errors.New("用户不存在或密码错误")
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("用户不存在或密码错误")
	}

	// 生成 JWT
	return s.generateJWT(user.ID)
}

// HandleGoogleLogin 处理 Google OAuth 登录。
func (s *Service) HandleGoogleLogin(providerID, email, username, fullName, avatarURL string) (string, error) {
	// 检查用户是否已通过 Google 登录过
	user, err := s.store.GetUserByProviderID("google", providerID)
	if err == nil {
		// 用户已存在，直接登录
		return s.generateJWT(user.ID)
	}

	// 用户不存在，创建新用户
	newUser := &models.User{
		Username:   username,
		FullName:   fullName,
		Email:      email,
		AvatarURL:  avatarURL,
		Provider:   "google",
		ProviderID: providerID,
		Status:     models.StatusActive,
	}

	if err := s.store.CreateUser(newUser); err != nil {
		return "", err
	}

	// 为新用户生成 JWT
	return s.generateJWT(newUser.ID)
}

// --- Permission Management ---

// AssignRoleToUser 为用户分配角色。
func (s *Service) AssignRoleToUser(userID, roleID uint) error {
	return s.store.AssignRoleToUser(userID, roleID)
}

// RevokeRoleFromUser 从用户撤销角色。
func (s *Service) RevokeRoleFromUser(userID, roleID uint) error {
	return s.store.RevokeRoleFromUser(userID, roleID)
}

// CheckUserPermission 检查用户是否拥有特定权限。
func (s *Service) CheckUserPermission(userID uint, requiredPermission string) (bool, error) {
	permissions, err := s.store.GetUserPermissions(userID)
	if err != nil {
		return false, err
	}

	for _, p := range permissions {
		if p.Name == requiredPermission {
			return true, nil
		}
	}

	return false, nil
}

// --- Helpers ---

// generateJWT 为指定用户 ID 生成一个新的 JWT。
func (s *Service) generateJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iss": "Jarvis_2.0_user_service",
		"aud": "Jarvis_2.0_clients",
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.jwtSecret)
}
