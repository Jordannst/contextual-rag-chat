package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"backend/db"
)

type CreateSessionRequest struct {
	Title string `json:"title" binding:"required"`
}

// CreateSessionHandler creates a new chat session
func CreateSessionHandler(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request. 'title' field is required",
		})
		return
	}

	sessionID, err := db.CreateSession(req.Title)
	if err != nil {
		log.Printf("[Session] Error creating session: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":    sessionID,
		"title": req.Title,
	})
}

// GetSessionsHandler retrieves all chat sessions
func GetSessionsHandler(c *gin.Context) {
	sessions, err := db.GetSessions()
	if err != nil {
		log.Printf("[Session] Error getting sessions: %v\n", err)
		// Return empty array instead of error if table doesn't exist yet
		// This prevents frontend from breaking while migration is running
		if err.Error() != "" {
			log.Printf("[Session] Detailed error: %v\n", err)
		}
		c.JSON(http.StatusOK, gin.H{
			"sessions": []interface{}{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
	})
}

// GetSessionMessagesHandler retrieves all messages for a specific session
func GetSessionMessagesHandler(c *gin.Context) {
	sessionIDStr := c.Param("id")
	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid session ID",
		})
		return
	}

	messages, err := db.GetSessionMessages(sessionID)
	if err != nil {
		log.Printf("[Session] Error getting messages for session %d: %v\n", sessionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve messages",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
	})
}

// DeleteSessionHandler deletes a chat session
func DeleteSessionHandler(c *gin.Context) {
	sessionIDStr := c.Param("id")
	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid session ID",
		})
		return
	}

	err = db.DeleteSession(sessionID)
	if err != nil {
		log.Printf("[Session] Error deleting session %d: %v\n", sessionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session deleted successfully",
	})
}

