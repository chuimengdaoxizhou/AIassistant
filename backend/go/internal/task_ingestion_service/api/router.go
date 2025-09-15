package api

import (
	"github.com/gin-gonic/gin"
)

// AuthMiddleware is a placeholder for the actual authentication middleware.
// In a real application, this would validate a JWT and set the "userID" in the context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For demonstration purposes, we'll use a static user ID.
		// Replace this with actual token validation logic.
		c.Set("userID", "user-12345")
		c.Next()
	}
}

// RegisterRoutes registers all the routes for the task ingestion service.
func RegisterRoutes(router *gin.Engine, api *API) {
	// All routes will be under /api/v1
	v1 := router.Group("/api/v1")
	
	// Apply authentication middleware to all task-related routes
	tasks := v1.Group("/tasks")
	tasks.Use(AuthMiddleware())
	{
		tasks.POST("", api.SubmitTaskHandler)
		tasks.GET("", api.GetTasksHandler)
		tasks.GET("/:id", api.GetTaskHandler)
	}

	// WebSocket route
	ws := router.Group("/ws")
	ws.Use(AuthMiddleware())
	{
		ws.GET("/subscribe", api.WebSocketHandler)
	}
}
