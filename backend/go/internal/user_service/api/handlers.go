package api

import (
	"Jarvis_2.0/backend/go/internal/user_service/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler 封装了所有 API endpoint 的处理函数。
type Handler struct {
	service *service.Service
}

// NewHandler 创建一个新的 Handler 实例。
func NewHandler(s *service.Service) *Handler {
	return &Handler{service: s}
}

// --- Registration and Login Handlers ---

// RegisterEmailRequest 定义了邮箱注册请求的 JSON 结构。
type RegisterEmailRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Username string `json:"username" binding:"required"`
	FullName string `json:"fullName"`
}

// RegisterEmail 处理邮箱注册请求。
func (h *Handler) RegisterEmail(c *gin.Context) {
	var req RegisterEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.RegisterUserByEmail(req.Email, req.Password, req.Username, req.FullName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "注册成功", "user_id": user.ID})
}

// LoginEmailRequest 定义了邮箱登录请求的 JSON 结构。
type LoginEmailRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:t"password" binding:"required"`
}

// LoginEmail 处理邮箱登录请求。
func (h *Handler) LoginEmail(c *gin.Context) {
	var req LoginEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.service.LoginUserByEmail(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// GoogleLoginRequest 定义了 Google 登录请求的 JSON 结构。
// 注意：这是一个简化的实现。在生产环境中，后端应该接收来自前端的 id_token，
// 然后在后端验证这个 token 的有效性，而不是直接信任前端发来的用户信息。
type GoogleLoginRequest struct {
	ProviderID string `json:"provider_id" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Username   string `json:"username" binding:"required"`
	FullName   string `json:"fullName"`
	AvatarURL  string `json:"avatarUrl"`
}

// HandleGoogleLogin 处理 Google OAuth 登录回调。
func (h *Handler) HandleGoogleLogin(c *gin.Context) {
	var req GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.service.HandleGoogleLogin(req.ProviderID, req.Email, req.Username, req.FullName, req.AvatarURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// --- Permission Handlers ---

// AssignRoleToUser 为指定 ID 的用户分配一个角色。
func (h *Handler) AssignRoleToUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户 ID 格式"})
		return
	}

	var req struct {
		RoleID uint `json:"role_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从 JWT 中间件获取当前操作者的用户ID
	operatorID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无法获取操作者信息"})
		return
	}

	// 检查操作者是否拥有分配角色的权限
	hasPermission, err := h.service.CheckUserPermission(operatorID.(uint), "roles:assign")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查权限失败"})
		return
	}
	if !hasPermission {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	if err := h.service.AssignRoleToUser(uint(userID), req.RoleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "角色分配成功"})

}
