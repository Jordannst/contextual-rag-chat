package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"backend/utils"
)

func UploadFile(c *gin.Context) {
	// Get file from form
	file, err := c.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Validate file extension
	ext := filepath.Ext(file.Filename)
	if ext != ".pdf" && ext != ".txt" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF and TXT files are allowed"})
		return
	}

	// Create uploads directory if not exists
	uploadsDir := "uploads"
	if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create uploads directory"})
		return
	}

	// Generate unique filename
	timestamp := time.Now().UnixNano()
	uniqueFilename := filepath.Base(file.Filename)
	fileExt := filepath.Ext(uniqueFilename)
	name := uniqueFilename[:len(uniqueFilename)-len(fileExt)]
	uniqueFilename = fmt.Sprintf("%s-%d%s", name, timestamp, fileExt)

	// Save file
	filePath := filepath.Join(uploadsDir, uniqueFilename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Extract text from file
	text, err := utils.ExtractTextFromFile(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error extracting text from file",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fileName": file.Filename,
		"filePath": filePath,
		"text":     text,
	})
}

