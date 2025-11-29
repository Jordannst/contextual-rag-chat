package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
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
	log.Printf("[Chat] ===== Starting chat request (Streaming) =====\n")
	
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

	// Step 5: Set SSE headers for streaming
	log.Printf("[Chat] Step 5: Setting up SSE headers...\n")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no") // Disable buffering in Nginx if used

	// Send initial metadata event (sources information)
	sourcesData := map[string]interface{}{
		"sources":    contextDocs,
		"sourceIds":  sourceIDs,
		"type":       "metadata",
	}
	sourcesJSON, _ := json.Marshal(sourcesData)
	fmt.Fprintf(c.Writer, "event: metadata\ndata: %s\n\n", sourcesJSON)
	c.Writer.Flush()

	// Step 6: Get streaming iterator
	log.Printf("[Chat] Step 6: Starting streaming response...\n")
	iter, err := utils.StreamChatResponse(req.Question, contextDocs, req.History)
	if err != nil {
		log.Printf("[Chat] ERROR DI STEP 6 (Stream Chat Response): %v\n", err)
		// Send error event
		errorData := map[string]string{
			"error":   "Failed to start streaming",
			"message": err.Error(),
			"type":    "error",
		}
		errorJSON, _ := json.Marshal(errorData)
		fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
		c.Writer.Flush()
		return
	}

	// Step 7: Stream chunks from iterator
	log.Printf("[Chat] Step 7: Streaming chunks...\n")
	var fullResponse strings.Builder
	chunkCount := 0

	for {
		// Get next chunk from iterator
		resp, err := iter.Next()
		if err != nil {
			// Check if iteration is done
			if err == iterator.Done {
				log.Printf("[Chat] Streaming completed. Total chunks: %d\n", chunkCount)
				break
			}
			
			// Check for other "done" indicators (fallback)
			errStr := err.Error()
			if strings.Contains(errStr, "done") || 
			   strings.Contains(errStr, "EOF") || 
			   strings.Contains(errStr, "no more") {
				log.Printf("[Chat] Streaming completed. Total chunks: %d\n", chunkCount)
				break
			}
			
			// Real error occurred
			log.Printf("[Chat] ERROR during streaming: %v\n", err)
			errorData := map[string]string{
				"error":   "Streaming error",
				"message": err.Error(),
				"type":    "error",
			}
			errorJSON, _ := json.Marshal(errorData)
			fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
			c.Writer.Flush()
			return
		}

		// Check if response is valid
		if resp == nil {
			continue
		}

		// Extract text from response chunks
		if resp.Candidates != nil && len(resp.Candidates) > 0 {
			if resp.Candidates[0].Content != nil && len(resp.Candidates[0].Content.Parts) > 0 {
				for _, part := range resp.Candidates[0].Content.Parts {
					if textPart, ok := part.(genai.Text); ok {
						text := string(textPart)
						if text != "" {
							// Send chunk with SSE format: data: <json>\n\n
							chunkData := map[string]string{
								"chunk": text,
								"type":  "chunk",
							}
							chunkJSON, err := json.Marshal(chunkData)
							if err != nil {
								log.Printf("[Chat] ERROR marshaling chunk: %v\n", err)
								continue
							}
							
							// Send with SSE format: data: <json>\n\n
							// JSON marshal already handles escaping properly
							fmt.Fprintf(c.Writer, "data: %s\n\n", chunkJSON)
							c.Writer.Flush()
							
							// Accumulate for logging
							fullResponse.WriteString(text)
							chunkCount++
							
							log.Printf("[Chat] Chunk %d sent (length: %d)\n", chunkCount, len(text))
						}
					}
				}
			}
		}
	}

	// Send completion event
	log.Printf("[Chat] Step 8: Sending completion event...\n")
	completeData := map[string]interface{}{
		"type":     "done",
		"totalChunks": chunkCount,
		"fullLength": fullResponse.Len(),
	}
	completeJSON, _ := json.Marshal(completeData)
	fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", completeJSON)
	c.Writer.Flush()

	log.Printf("[Chat] ===== Chat streaming completed successfully (total: %d chars, %d chunks) =====\n", fullResponse.Len(), chunkCount)
	
	// Return false to prevent Gin from writing additional JSON body
	// Note: In Gin, we don't explicitly return false, we just don't call c.JSON
	// The streaming response is already sent
}

