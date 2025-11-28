package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"backend/db"
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

	// Split text into chunks
	chunks := utils.SplitText(text, 1000, 200)
	if len(chunks) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "No text chunks generated from file",
		})
		return
	}

	// Process each chunk: generate embedding and save to database
	var savedChunks int
	var lastError error

	for _, chunk := range chunks {
		// Generate embedding for this chunk
		embedding, err := utils.GenerateEmbedding(chunk)
		if err != nil {
			lastError = err
			continue // Skip this chunk and continue with next
		}

		// Insert chunk with embedding to database
		err = db.InsertDocument(chunk, embedding, file.Filename)
		if err != nil {
			lastError = err
			continue // Skip this chunk and continue with next
		}

		savedChunks++
	}

	// Check if at least one chunk was saved
	if savedChunks == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save any chunks to database",
			"message": lastError.Error(),
		})
		return
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
		"totalChunks": len(chunks),
	})
}

