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

func DocumentRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/documents", handlers.GetDocumentsHandler)
		api.DELETE("/documents/:filename", handlers.DeleteDocumentHandler)
		api.POST("/documents/sync", handlers.SyncDocumentsHandler)
	}
}

