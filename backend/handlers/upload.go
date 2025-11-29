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

	// Extract text from file
	fmt.Printf("[Upload] Extracting text from: %s\n", filePath)
	text, err := utils.ExtractTextFromFile(filePath)
	if err != nil {
		fmt.Printf("[Upload] Error extracting text: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Error extracting text from file",
			"message": err.Error(),
		})
		return
	}
	fmt.Printf("[Upload] Text extracted, length: %d characters\n", len(text))

	// Check if text is empty
	if len(text) == 0 {
		fmt.Printf("[Upload] Warning: No text extracted from file\n")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "No text extracted from file",
			"message": "The file appears to be empty or could not be read",
		})
		return
	}

	// Split text into chunks
	fmt.Printf("[Upload] Splitting text into chunks...\n")
	chunks := utils.SplitText(text, 1000, 200)
	fmt.Printf("[Upload] Created %d chunks\n", len(chunks))
	if len(chunks) == 0 {
		fmt.Printf("[Upload] Error: No chunks generated\n")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "No text chunks generated from file",
		})
		return
	}

	// Process each chunk: generate embedding and save to database
	fmt.Printf("[Upload] Processing chunks (generating embeddings and saving to DB)...\n")
	var savedChunks int
	var lastError error

	for i, chunk := range chunks {
		fmt.Printf("[Upload] Processing chunk %d/%d (length: %d)\n", i+1, len(chunks), len(chunk))
		
		// Generate embedding for this chunk
		embedding, err := utils.GenerateEmbedding(chunk)
		if err != nil {
			fmt.Printf("[Upload] Error generating embedding for chunk %d: %v\n", i+1, err)
			lastError = err
			continue // Skip this chunk and continue with next
		}
		fmt.Printf("[Upload] Embedding generated for chunk %d (dimension: %d)\n", i+1, len(embedding))

		// Insert chunk with embedding to database
		err = db.InsertDocument(chunk, embedding, file.Filename)
		if err != nil {
			fmt.Printf("[Upload] Error inserting chunk %d to database: %v\n", i+1, err)
			lastError = err
			continue // Skip this chunk and continue with next
		}
		fmt.Printf("[Upload] Chunk %d saved to database\n", i+1)
		savedChunks++
	}
	
	fmt.Printf("[Upload] Completed: %d/%d chunks saved\n", savedChunks, len(chunks))

	// Check if at least one chunk was saved
	if savedChunks == 0 {
		fmt.Printf("[Upload] Error: No chunks saved. Last error: %v\n", lastError)
		errorMsg := "Unknown error"
		if lastError != nil {
			errorMsg = lastError.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to save any chunks to database",
			"message": errorMsg,
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

