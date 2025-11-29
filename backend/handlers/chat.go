package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"backend/db"
	"backend/models"
	"backend/utils"
)

type ChatRequest struct {
	Question string             `json:"question" binding:"required"`
	History  []models.ChatMessage `json:"history"`
}

type ChatResponse struct {
	Response   string   `json:"response"`
	Sources    []string `json:"sources,omitempty"`
	SourceIDs  []int32  `json:"sourceIds,omitempty"`
}

func ChatHandler(c *gin.Context) {
	log.Printf("[Chat] ===== Starting chat request =====\n")
	
	// Step 1: Parse request
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[Chat] ERROR DI STEP 1 (Parse Request): %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request. 'question' field is required",
		})
		return
	}
	log.Printf("[Chat] Step 1: Request diterima - Question: %s, History length: %d\n", req.Question, len(req.History))

	// Step 2: Generate embedding for user query
	log.Printf("[Chat] Step 2: Generating embedding for query...\n")
	queryEmbedding, err := utils.GenerateEmbedding(req.Question)
	if err != nil {
		log.Printf("[Chat] ERROR DI STEP 2 (Generate Embedding): %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate query embedding",
			"message": err.Error(),
		})
		return
	}
	log.Printf("[Chat] Step 2: Embedding Query berhasil generate (dimension: %d)\n", len(queryEmbedding))

	// Step 3: Search for similar documents
	log.Printf("[Chat] Step 3: Mencari dokumen di DB...\n")
	similarDocs, err := db.SearchSimilarDocuments(queryEmbedding, 3)
	if err != nil {
		log.Printf("[Chat] ERROR DI STEP 3 (Search Documents): %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search similar documents",
			"message": err.Error(),
		})
		return
	}
	log.Printf("[Chat] Step 3: Dokumen ditemukan: %d dokumen\n", len(similarDocs))

	// Step 4: Extract content from similar documents
	var contextDocs []string
	var sourceIDs []int32
	for i, doc := range similarDocs {
		contextDocs = append(contextDocs, doc.Content)
		sourceIDs = append(sourceIDs, doc.ID)
		log.Printf("[Chat] Step 4: Dokumen %d - ID: %d, Content length: %d, Distance: %.4f\n", i+1, doc.ID, len(doc.Content), doc.Distance)
	}
	log.Printf("[Chat] Step 4: Total context docs: %d\n", len(contextDocs))

	// Step 5: Generate chat response with RAG context and history
	log.Printf("[Chat] Step 5: Mengirim prompt ke Gemini...\n")
	response, err := utils.GenerateChatResponse(req.Question, contextDocs, req.History)
	if err != nil {
		log.Printf("[Chat] ERROR DI STEP 5 (Generate Chat Response): %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate chat response",
			"message": err.Error(),
		})
		return
	}
	log.Printf("[Chat] Step 5: Response berhasil di-generate (length: %d characters)\n", len(response))

	// Step 6: Prepare and send response
	log.Printf("[Chat] Step 6: Menyiapkan response...\n")
	chatResp := ChatResponse{
		Response:  response,
		Sources:   contextDocs,
		SourceIDs: sourceIDs,
	}
	log.Printf("[Chat] ===== Chat request completed successfully =====\n")

	c.JSON(http.StatusOK, chatResp)
}

