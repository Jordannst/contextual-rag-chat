package utils

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"backend/db"
)

// ProcessAndSaveDocument processes a document file and saves it to the database
// This function extracts text, chunks it, generates embeddings, and saves to DB
// filePath: full path to the file (e.g., "uploads/document.pdf")
// sourceFileName: original filename to store in DB (e.g., "document.pdf")
// Returns the number of chunks saved and any error
func ProcessAndSaveDocument(filePath string, sourceFileName string) (int, error) {
	log.Printf("[ProcessDocument] Processing file: %s (source: %s)\n", filePath, sourceFileName)

	// Step 1: Extract text from file
	log.Printf("[ProcessDocument] Extracting text from: %s\n", filePath)
	text, err := ExtractTextFromFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to extract text: %w", err)
	}
	log.Printf("[ProcessDocument] Text extracted, length: %d characters\n", len(text))

	// Check if text is empty
	if len(text) == 0 {
		return 0, fmt.Errorf("no text extracted from file")
	}

	// Step 2: Split text into chunks
	log.Printf("[ProcessDocument] Splitting text into chunks...\n")
	chunks := SplitText(text, 1000, 200)
	log.Printf("[ProcessDocument] Created %d chunks\n", len(chunks))
	if len(chunks) == 0 {
		return 0, fmt.Errorf("no text chunks generated from file")
	}

	// Step 3: Process each chunk: generate embedding and save to database
	log.Printf("[ProcessDocument] Processing chunks (generating embeddings and saving to DB)...\n")
	var savedChunks int
	var lastError error

	for i, chunk := range chunks {
		log.Printf("[ProcessDocument] Processing chunk %d/%d (length: %d)\n", i+1, len(chunks), len(chunk))

		// Generate embedding for this chunk
		embedding, err := GenerateEmbedding(chunk)
		if err != nil {
			log.Printf("[ProcessDocument] Error generating embedding for chunk %d: %v\n", i+1, err)
			lastError = err
			continue // Skip this chunk and continue with next
		}
		log.Printf("[ProcessDocument] Embedding generated for chunk %d (dimension: %d)\n", i+1, len(embedding))

		// Insert chunk with embedding to database
		err = db.InsertDocument(chunk, embedding, sourceFileName)
		if err != nil {
			log.Printf("[ProcessDocument] Error inserting chunk %d to database: %v\n", i+1, err)
			lastError = err
			continue // Skip this chunk and continue with next
		}
		log.Printf("[ProcessDocument] Chunk %d saved to database\n", i+1)
		savedChunks++
	}

	log.Printf("[ProcessDocument] Completed: %d/%d chunks saved for file: %s\n", savedChunks, len(chunks), sourceFileName)

	// Check if at least one chunk was saved
	if savedChunks == 0 {
		errorMsg := "unknown error"
		if lastError != nil {
			errorMsg = lastError.Error()
		}
		return 0, fmt.Errorf("failed to save any chunks to database: %s", errorMsg)
	}

	return savedChunks, nil
}

// ValidateFileExtension checks if the file extension is supported
func ValidateFileExtension(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".pdf" || ext == ".txt"
}

