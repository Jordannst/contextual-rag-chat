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
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func main() {
	// Load environment variables with BOM handling
	utils.LoadEnvWithBOMHandling()

	// Initialize KeyManager early to validate API keys
	keyManager := utils.GetKeyManager()
	if !keyManager.IsInitialized() {
		log.Fatal("ERROR: No valid API keys found. Please set GEMINI_API_KEY or GEMINI_API_KEYS in your .env file")
	}
	log.Println("✓ KeyManager initialized successfully")

	// Initialize database connection
	if err := db.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.CloseDB()

	// Run chat sessions migration if tables don't exist
	if err := db.RunChatSessionsMigration(); err != nil {
		log.Printf("Warning: Failed to run chat sessions migration: %v", err)
		log.Printf("You may need to run migration manually: psql -d your_database -f backend/db/migration_chat_sessions.sql")
	}

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

	// Rate Limiting Middleware
	// Create a rate limiter based on IP address: 100 requests per minute
	rate, err := limiter.NewRateFromFormatted("100-M")
	if err != nil {
		log.Fatal(err)
	}
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	r.Use(mgin.NewMiddleware(instance))

	// CORS middleware
	config := cors.DefaultConfig()
	// Security improvement: Restrict origins instead of allowing all
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		config.AllowOrigins = []string{allowedOrigins}
	} else {
		// Default to localhost:3000 for development
		config.AllowOrigins = []string{"http://localhost:3000"}
	}
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

	// Document management routes (includes file handler)
	// GetFileHandler handles file serving with pattern matching for timestamped filenames
	// Files can be accessed via: http://localhost:5000/api/files/filename.pdf
	routes.DocumentRoutes(r)

	// Session routes (chat history persistence)
	routes.SessionRoutes(r)

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

