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
		api.GET("/chat/suggestions", handlers.GetSuggestionsHandler)
	}
}

func DocumentRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/documents", handlers.GetDocumentsHandler)
		api.DELETE("/documents/:filename", handlers.DeleteDocumentHandler)
		api.POST("/documents/sync", handlers.SyncDocumentsHandler)
		api.GET("/files/:filename", handlers.GetFileHandler)
	}
}

func SessionRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/sessions", handlers.CreateSessionHandler)
		api.GET("/sessions", handlers.GetSessionsHandler)
		api.GET("/sessions/:id", handlers.GetSessionMessagesHandler)
		api.DELETE("/sessions/:id", handlers.DeleteSessionHandler)
	}
}

