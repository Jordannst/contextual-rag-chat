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
	fmt.Printf("[Upload] Starting file upload handler\n")
	
	// Get file from form
	file, err := c.FormFile("document")
	if err != nil {
		fmt.Printf("[Upload] Error getting file: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	fmt.Printf("[Upload] File received: %s, size: %d\n", file.Filename, file.Size)

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

	// Process and save document using reusable function
	savedChunks, err := utils.ProcessAndSaveDocument(filePath, file.Filename)
	if err != nil {
		fmt.Printf("[Upload] Error processing document: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process document",
			"message": err.Error(),
		})
		return
	}

	// Extract text for preview (we need to read it again for preview)
	text, err := utils.ExtractTextFromFile(filePath)
	if err != nil {
		// If preview extraction fails, just use empty string
		text = ""
	}

	// Generate preview text (first 200 characters)
	previewText := text
	if len(text) > 200 {
		previewText = text[:200] + "..."
	}

	c.JSON(http.StatusOK, gin.H{
		"fileName":    file.Filename,
		"filePath":    filePath,
		"text":        text,
		"message":     fmt.Sprintf("File berhasil diupload, divektorisasi, dan disimpan ke database (%d chunks)", savedChunks),
		"previewText": previewText,
		"chunksCount": savedChunks,
	})
}

