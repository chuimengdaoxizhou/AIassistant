package api

import "github.com/gin-gonic/gin"

// SetupRouter 配置和返回一个 Gin 引擎实例。
func SetupRouter(h *Handler, jwtSecret string) *gin.Engine {
	// 使用默认中间件 (logger, recovery) 创建一个 Gin 引擎。
	r := gin.Default()

	// 创建认证中间件实例
	authMiddleware := AuthMiddleware(jwtSecret)

	// 使用 v1 版本对 API 进行分组
	apiV1 := r.Group("/api/v1")
	{
		// 用户认证路由组
		auth := apiV1.Group("/auth")
		{
			auth.POST("/register", h.RegisterEmail)
			auth.POST("/login", h.LoginEmail)
			auth.POST("/google/login", h.HandleGoogleLogin)
		}

		// 用户和权限管理路由组
		// 使用认证中间件保护这个组下的所有路由
		users := apiV1.Group("/users")
		users.Use(authMiddleware)
		{
			// 例如: POST /api/v1/users/123/roles
			users.POST("/:id/roles", h.AssignRoleToUser)
			// 可以在这里添加更多路由，例如撤销角色、获取用户信息等
		}
	}

	return r
}
