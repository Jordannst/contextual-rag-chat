package routes

import (
	"backend/handlers"

	"github.com/gin-gonic/gin"
)

func UploadRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/upload", handlers.UploadFile)
	}
}

func ChatRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/chat", handlers.ChatHandler)
	}
}

