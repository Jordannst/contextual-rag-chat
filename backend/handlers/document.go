package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"backend/db"
	"backend/utils"
)

// GetDocumentsHandler returns a list of unique document file names
func GetDocumentsHandler(c *gin.Context) {
	log.Printf("[Documents] Fetching list of documents...\n")

	documents, err := db.GetUniqueDocuments()
	if err != nil {
		log.Printf("[Documents] ERROR fetching documents: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch documents",
			"message": err.Error(),
		})
		return
	}

	log.Printf("[Documents] Found %d unique documents\n", len(documents))

	c.JSON(http.StatusOK, gin.H{
		"documents": documents,
		"count":     len(documents),
	})
}

// DeleteDocumentHandler deletes all chunks belonging to a specific file and removes the physical file
func DeleteDocumentHandler(c *gin.Context) {
	fileName := c.Param("filename")
	if fileName == "" {
		log.Printf("[Documents] ERROR: filename parameter is empty\n")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Filename parameter is required",
		})
		return
	}

	log.Printf("[Documents] Deleting document: %s\n", fileName)

	// Step 1: Delete physical file from disk
	uploadsDir := "uploads"
	filePath := filepath.Join(uploadsDir, fileName)

	err := os.Remove(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - that's okay, maybe it was already deleted manually
			// Continue with database deletion
			log.Printf("[Documents] File not found on disk (may have been deleted manually): %s\n", filePath)
		} else {
			// Other error (permission, etc.) - log it but continue with DB deletion
			// This ensures RAG system stays consistent even if file deletion fails
			log.Printf("[Documents] WARNING: Failed to delete physical file: %v (continuing with DB deletion)\n", err)
		}
	} else {
		log.Printf("[Documents] Physical file deleted successfully: %s\n", filePath)
	}

	// Step 2: Delete from database (always proceed, even if file deletion had issues)
	err = db.DeleteDocument(fileName)
	if err != nil {
		log.Printf("[Documents] ERROR deleting document from database: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete document from database",
			"message": err.Error(),
		})
		return
	}

	log.Printf("[Documents] Successfully deleted document from database: %s\n", fileName)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Document deleted successfully",
		"fileName": fileName,
	})
}

// SyncDocumentsHandler syncs the database with physical files, removing orphaned records and importing new files
func SyncDocumentsHandler(c *gin.Context) {
	log.Printf("[Documents] Starting database sync...\n")

	uploadsDir := "uploads"
	deletedCount := 0
	addedCount := 0
	var deletedFiles []string
	var addedFiles []string

	// Step 1: Get all unique documents from database
	documentsInDB, err := db.GetUniqueDocuments()
	if err != nil {
		log.Printf("[Documents] ERROR fetching documents for sync: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch documents",
			"message": err.Error(),
		})
		return
	}

	log.Printf("[Documents] Found %d documents in database, checking physical files...\n", len(documentsInDB))

	// Create a map for quick lookup of documents in DB
	dbFileMap := make(map[string]bool)
	for _, fileName := range documentsInDB {
		dbFileMap[fileName] = true
	}

	// Step 2: Remove orphaned records (files in DB but not on disk)
	for _, fileName := range documentsInDB {
		filePath := filepath.Join(uploadsDir, fileName)

		// Check if file exists
		_, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				// File doesn't exist - it's an orphaned record, delete from DB
				log.Printf("[Documents] Found orphaned record (file missing): %s, deleting from DB...\n", fileName)

				err := db.DeleteDocument(fileName)
				if err != nil {
					log.Printf("[Documents] ERROR deleting orphaned document %s: %v\n", fileName, err)
					// Continue with other files even if one fails
					continue
				}

				deletedCount++
				deletedFiles = append(deletedFiles, fileName)
				log.Printf("[Documents] Deleted orphaned record: %s\n", fileName)
			} else {
				// Other error (permission, etc.) - log but don't delete
				log.Printf("[Documents] WARNING: Error checking file %s: %v (skipping)\n", filePath, err)
			}
		}
		// If file exists, do nothing (it's valid)
	}

	// Step 3: Scan folder for new files (files on disk but not in DB)
	log.Printf("[Documents] Scanning folder for new files...\n")
	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		log.Printf("[Documents] ERROR reading uploads directory: %v\n", err)
		// Continue with sync results even if folder scan fails
	} else {
		for _, entry := range entries {
			// Skip directories
			if entry.IsDir() {
				continue
			}

			fileName := entry.Name()

			// Skip if file is already in DB
			if dbFileMap[fileName] {
				continue
			}

			// Check if file extension is supported
			if !utils.ValidateFileExtension(fileName) {
				log.Printf("[Documents] Skipping unsupported file: %s\n", fileName)
				continue
			}

			// This is a new file - process and save it
			filePath := filepath.Join(uploadsDir, fileName)
			log.Printf("[Documents] Found new file: %s, processing...\n", fileName)

			savedChunks, err := utils.ProcessAndSaveDocument(filePath, fileName)
			if err != nil {
				log.Printf("[Documents] ERROR processing new file %s: %v\n", fileName, err)
				// Continue with other files even if one fails
				continue
			}

			addedCount++
			addedFiles = append(addedFiles, fileName)
			log.Printf("[Documents] Successfully imported new file: %s (%d chunks)\n", fileName, savedChunks)
		}
	}

	log.Printf("[Documents] Sync complete. Deleted %d orphaned records, added %d new files.\n", deletedCount, addedCount)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Sync complete",
		"deleted_count": deletedCount,
		"added_count":   addedCount,
		"deleted_files": deletedFiles,
		"added_files":   addedFiles,
	})
}

