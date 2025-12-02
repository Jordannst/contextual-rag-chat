package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"backend/db"
	"backend/utils"
)

// GetSuggestionsHandler generates question suggestions based on random document context
func GetSuggestionsHandler(c *gin.Context) {
	log.Printf("[Suggestions] Generating question suggestions...\n")

	// Step 1: Get random context from database
	limit := 5 // Get 5 random chunks
	contexts, err := db.GetRandomContext(limit)
	if err != nil {
		log.Printf("[Suggestions] ERROR getting random context: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get document context",
			"message": err.Error(),
		})
		return
	}

	// If no documents found, return empty suggestions
	if len(contexts) == 0 {
		log.Printf("[Suggestions] No documents found in database\n")
		c.JSON(http.StatusOK, gin.H{
			"questions": []string{},
		})
		return
	}

	log.Printf("[Suggestions] Retrieved %d random document chunks\n", len(contexts))

	// Step 2: Combine contexts into a single text (limit total length)
	// Combine up to 2000 characters to avoid token limits
	var combinedContext strings.Builder
	totalLength := 0
	maxLength := 2000

	for _, ctx := range contexts {
		if totalLength+len(ctx) > maxLength {
			// Add partial content if there's space
			remaining := maxLength - totalLength
			if remaining > 100 {
				combinedContext.WriteString(ctx[:remaining])
				combinedContext.WriteString("...\n\n")
			}
			break
		}
		combinedContext.WriteString(ctx)
		combinedContext.WriteString("\n\n")
		totalLength += len(ctx) + 2
	}

	contextText := combinedContext.String()
	log.Printf("[Suggestions] Combined context length: %d characters\n", len(contextText))

	// Step 3: Generate question suggestions using AI
	questions, err := utils.GenerateQuestionSuggestions(contextText)
	if err != nil {
		log.Printf("[Suggestions] ERROR generating suggestions: %v\n", err)
		// Return default questions as fallback
		questions = []string{
			"Apa topik utama dari dokumen ini?",
			"Bisa jelaskan lebih detail tentang isi dokumen?",
			"Apa saja poin penting yang perlu diketahui?",
		}
		log.Printf("[Suggestions] Using fallback questions\n")
	}

	log.Printf("[Suggestions] Generated %d question suggestions\n", len(questions))

	// Step 4: Return suggestions
	c.JSON(http.StatusOK, gin.H{
		"questions": questions,
	})
}

