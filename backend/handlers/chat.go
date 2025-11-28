package handlers

import (
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
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request. 'question' field is required",
		})
		return
	}

	// Generate embedding for user query
	queryEmbedding, err := utils.GenerateEmbedding(req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate query embedding",
			"message": err.Error(),
		})
		return
	}

	// Search for similar documents (top 3 most similar)
	similarDocs, err := db.SearchSimilarDocuments(queryEmbedding, 3)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search similar documents",
			"message": err.Error(),
		})
		return
	}

	// Extract content from similar documents
	var contextDocs []string
	var sourceIDs []int32
	for _, doc := range similarDocs {
		contextDocs = append(contextDocs, doc.Content)
		sourceIDs = append(sourceIDs, doc.ID)
	}

	// Generate chat response with RAG context and history
	response, err := utils.GenerateChatResponse(req.Question, contextDocs, req.History)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate chat response",
			"message": err.Error(),
		})
		return
	}

	// Prepare response
	chatResp := ChatResponse{
		Response:  response,
		Sources:   contextDocs,
		SourceIDs: sourceIDs,
	}

	c.JSON(http.StatusOK, chatResp)
}

