package api

import (
	"net/http"
	"os"

	"github.com/enkinvsh/focus-backend/internal/bot"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/bot/webhook", func(c *gin.Context) {
		secret := os.Getenv("WEBHOOK_SECRET")
		if secret != "" && c.Query("token") != secret {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		var update bot.Update
		if err := c.ShouldBindJSON(&update); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := bot.HandleWebhook(&update); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	api := r.Group("/api/v1")
	api.Use(AuthMiddleware())
	{
		api.GET("/tasks", GetTasks)
		api.POST("/tasks", CreateTask)
		api.POST("/tasks/audio", CreateTaskFromAudio)
		api.PATCH("/tasks/:id", UpdateTask)
		api.DELETE("/tasks/:id", DeleteTask)

		api.GET("/user/preferences", GetPreferences)
		api.PATCH("/user/preferences", UpdatePreferences)
	}
}
