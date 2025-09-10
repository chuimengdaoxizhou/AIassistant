package api

import (
	"errors"
	"github.com/golang-jwt/jwt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 创建一个 Gin 中间件，用于验证 JWT。
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请求未包含授权标头"})
			c.Abort()
			return
		}

		// 我们期望的格式是 "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "授权标头格式不正确"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 解析和验证 token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 确保 token 的签名方法是我们期望的
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("非预期的签名方法")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// 从 claims 中获取用户 ID
			userID, ok := claims["sub"].(float64) // JWT 解析数字时默认为 float64
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 token claims"})
				c.Abort()
				return
			}
			// 将用户 ID 存储在 Gin 的上下文中，以便后续的处理函数可以使用
			c.Set("userID", uint(userID))
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 token"})
			c.Abort()
			return
		}

		// 进入下一个处理函数
		c.Next()
	}
}
