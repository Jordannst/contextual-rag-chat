package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"backend/db"
	"backend/routes"
	"backend/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load environment variables with BOM handling
	utils.LoadEnvWithBOMHandling()

	// Initialize database connection
	if err := db.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.CloseDB()

	// Setup router with recovery middleware
	r := gin.Default()
	
	// Add logging middleware
	r.Use(func(c *gin.Context) {
		// Log request
		log.Printf("[Request] %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
		// Log response
		log.Printf("[Response] %s %s - Status: %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	})
	
	// Add custom recovery middleware to log errors
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Printf("[PANIC] Recovered: %v", recovered)
		log.Printf("[PANIC] Request: %s %s", c.Request.Method, c.Request.URL.Path)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": fmt.Sprintf("%v", recovered),
		})
	}))

	// CORS middleware
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// Routes
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// Upload routes
	routes.UploadRoutes(r)

	// Chat routes
	routes.ChatRoutes(r)

	// Document management routes
	routes.DocumentRoutes(r)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Printf("Server is running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

