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

	// Step 3: Search for similar documents using Hybrid Search
	// Hybrid Search combines vector similarity (semantic) + full-text search (keyword)
	log.Printf("[Chat] Step 3: Mencari dokumen di DB menggunakan Hybrid Search...\n")
	limit := 3
	vectorWeight := 0.7 // 70% vector, 30% text
	
	similarDocs, err := db.SearchDocuments(queryEmbedding, req.Question, limit, vectorWeight)
	if err != nil {
		log.Printf("[Chat] ERROR DI STEP 3 (Search Documents): %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search similar documents",
			"message": err.Error(),
		})
		return
	}
	log.Printf("[Chat] Step 3: Hybrid Search menemukan: %d dokumen\n", len(similarDocs))
	
	// Fallback Strategy: Jika hybrid search tidak menemukan hasil, fallback ke vector-only
	if len(similarDocs) == 0 && req.Question != "" {
		log.Printf("[Chat] Step 3: WARNING - Hybrid search yielded 0 results, falling back to vector-only search.\n")
		similarDocs, err = db.SearchSimilarDocuments(queryEmbedding, limit)
		if err != nil {
			log.Printf("[Chat] ERROR DI STEP 3 (Fallback Vector Search): %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to search similar documents (fallback)",
				"message": err.Error(),
			})
			return
		}
		log.Printf("[Chat] Step 3: Vector-only search menemukan: %d dokumen\n", len(similarDocs))
	}

	// Step 4: Extract content from similar documents and collect unique source files
	// Apply similarity threshold to filter out irrelevant results
	const similarityThreshold = 0.65 // Cosine distance threshold (0 = identical, 2 = opposite)
	// Documents with distance < 0.65 are considered relevant
	// Documents with distance >= 0.65 are too dissimilar and should be excluded
	// Note: Increased from 0.5 to 0.65 to be less strict for short queries
	
	var contextDocs []string
	var sourceIDs []int32
	uniqueSourceFiles := make(map[string]bool) // Map untuk deduplikasi nama file
	var uniqueSources []string                  // List nama file unik
	var filteredCount int                       // Count of documents filtered out
	
	for i, doc := range similarDocs {
		// Log candidate before filtering to see actual distances
		log.Printf("[Chat] Step 4: Candidate %d - SourceFile: %s | Distance: %.4f\n", 
			i+1, doc.SourceFile, doc.Distance)
		
		// Apply similarity threshold filter
		// Only include documents with distance below threshold (more similar)
		if doc.Distance >= similarityThreshold {
			log.Printf("[Chat] Step 4: Dokumen %d - ID: %d, SourceFile: %s, Distance: %.4f (FILTERED OUT - too dissimilar, threshold: %.2f)\n", 
				i+1, doc.ID, doc.SourceFile, doc.Distance, similarityThreshold)
			filteredCount++
			continue // Skip this document - not relevant enough
		}
		
		// Document passed threshold - include in context and sources
		// Format context dengan metadata nama file untuk inline citations
		// Format: [Document: nama_file.pdf]\nIsi konten: ... potongan teks ...
		var formattedContext string
		if doc.SourceFile != "" {
			formattedContext = fmt.Sprintf("[Document: %s]\n%s", doc.SourceFile, doc.Content)
		} else {
			// Fallback jika source_file kosong
			formattedContext = fmt.Sprintf("[Document: unknown]\n%s", doc.Content)
		}
		contextDocs = append(contextDocs, formattedContext)
		sourceIDs = append(sourceIDs, doc.ID)
		
		// Kumpulkan source file dengan deduplikasi
		// Hanya masukkan jika: (1) tidak kosong/null, (2) belum ada di map
		if doc.SourceFile != "" && !uniqueSourceFiles[doc.SourceFile] {
			uniqueSourceFiles[doc.SourceFile] = true
			uniqueSources = append(uniqueSources, doc.SourceFile)
		}
		
		log.Printf("[Chat] Step 4: Dokumen %d - ID: %d, Content length: %d, SourceFile: %s, Distance: %.4f (INCLUDED)\n", 
			i+1, doc.ID, len(doc.Content), doc.SourceFile, doc.Distance)
	}
	log.Printf("[Chat] Step 4: Total context docs: %d, Unique source files: %d, Filtered out: %d\n", 
		len(contextDocs), len(uniqueSources), filteredCount)

	// Step 5: Set SSE headers for streaming
	log.Printf("[Chat] Step 5: Setting up SSE headers...\n")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no") // Disable buffering in Nginx if used

	// Send initial metadata event (sources information)
	// Kirim unique source file names, bukan content
	sourcesData := map[string]interface{}{
		"sources":    uniqueSources, // Nama file unik, bukan content
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

