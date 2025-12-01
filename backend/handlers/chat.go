package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"backend/db"
	"backend/models"
	"backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
)

type ChatRequest struct {
	Question      string               `json:"question" binding:"required"`
	History       []models.ChatMessage `json:"history"`
	SelectedFiles []string             `json:"selectedFiles,omitempty"` // Optional: filter by specific files
	SessionID     *int                 `json:"sessionId,omitempty"`     // Optional: session ID for persistence
}

type ChatResponse struct {
	Response  string   `json:"response"`
	Sources   []string `json:"sources,omitempty"`
	SourceIDs []int32  `json:"sourceIds,omitempty"`
	SessionID *int     `json:"sessionId,omitempty"` // Return session ID (new or existing)
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

	// Step 1.5: Rewrite query if there's history (for better RAG accuracy on follow-up questions)
	var rewrittenQuery string
	var err error
	if len(req.History) > 0 {
		log.Printf("[Chat] Step 1.5: Rewriting query with context from history...\n")
		rewrittenQuery, err = utils.RewriteQuery(req.History, req.Question)
		if err != nil {
			log.Printf("[Chat] WARNING: Query rewriting failed: %v. Using original query.\n", err)
			rewrittenQuery = req.Question // Fallback to original
		}
		log.Printf("[Chat] Original: %s | Rewritten: %s\n", req.Question, rewrittenQuery)
	} else {
		rewrittenQuery = req.Question // No history, no need to rewrite
		log.Printf("[Chat] Step 1.5: No history, using original query\n")
	}

	// Step 2: Generate embedding for rewritten query
	log.Printf("[Chat] Step 2: Generating embedding for query...\n")
	queryEmbedding, err := utils.GenerateEmbedding(rewrittenQuery)
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
	// Broad search: ambil kandidat lebih banyak untuk direrank dengan Cohere
	limit := 25
	vectorWeight := 0.7 // 70% vector, 30% text

	// Get file filters from request (if any)
	fileFilters := req.SelectedFiles
	if len(fileFilters) > 0 {
		log.Printf("[Chat] Step 3: Filtering by files: %v\n", fileFilters)
	} else {
		log.Printf("[Chat] Step 3: No file filter - searching all documents\n")
	}

	similarDocs, err := db.SearchDocuments(queryEmbedding, rewrittenQuery, limit, vectorWeight, fileFilters)
	if err != nil {
		log.Printf("[Chat] ERROR DI STEP 3 (Search Documents): %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search similar documents",
			"message": err.Error(),
		})
		return
	}
	log.Printf("[Chat] Step 3: Hybrid Search menemukan: %d dokumen (kandidat sebelum rerank)\n", len(similarDocs))

	// Fallback Strategy: Jika hybrid search tidak menemukan hasil, fallback ke vector-only
	if len(similarDocs) == 0 && rewrittenQuery != "" {
		log.Printf("[Chat] Step 3: WARNING - Hybrid search yielded 0 results, falling back to vector-only search.\n")
		similarDocs, err = db.SearchSimilarDocuments(queryEmbedding, limit, fileFilters)
		if err != nil {
			log.Printf("[Chat] ERROR DI STEP 3 (Fallback Vector Search): %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to search similar documents (fallback)",
				"message": err.Error(),
			})
			return
		}
		log.Printf("[Chat] Step 3: Vector-only search menemukan: %d dokumen (kandidat sebelum rerank)\n", len(similarDocs))
	}

	// Step 3.5: Reranking dengan Cohere (AI filter) untuk memilih 5 dokumen terbaik
	const rerankTopN = 5
	if len(similarDocs) > 0 {
		log.Printf("[Chat] Step 3.5: Menjalankan Cohere Rerank untuk memilih %d dokumen terbaik...\n", rerankTopN)

		// Siapkan konten untuk dikirim ke Cohere
		contents := make([]string, 0, len(similarDocs))
		for _, doc := range similarDocs {
			contents = append(contents, doc.Content)
		}

		indices, rerankErr := utils.RerankDocuments(rewrittenQuery, contents, rerankTopN)
		if rerankErr != nil {
			// Fallback: pakai top 5 dokumen pertama dari hasil DB tanpa rerank
			log.Printf("[Chat] WARNING: Cohere Rerank gagal: %v. Fallback ke top %d dokumen dari DB.\n", rerankErr, rerankTopN)
			top := rerankTopN
			if len(similarDocs) < top {
				top = len(similarDocs)
			}
			similarDocs = similarDocs[:top]
		} else {
			// Susun ulang similarDocs berdasarkan indeks yang dikembalikan Cohere
			log.Printf("[Chat] Step 3.5: Cohere Rerank mengembalikan %d indeks\n", len(indices))
			reordered := make([]db.Document, 0, len(indices))
			seen := make(map[int]bool)
			for _, idx := range indices {
				if idx >= 0 && idx < len(similarDocs) && !seen[idx] {
					reordered = append(reordered, similarDocs[idx])
					seen[idx] = true
				}
			}

			// Jika karena alasan apapun tidak ada indeks valid, fallback ke top N original
			if len(reordered) == 0 {
				log.Printf("[Chat] WARNING: Cohere Rerank tidak menghasilkan indeks valid. Fallback ke top %d dokumen original.\n", rerankTopN)
				top := rerankTopN
				if len(similarDocs) < top {
					top = len(similarDocs)
				}
				reordered = similarDocs[:top]
			}

			// Batasi ke rerankTopN dokumen terbaik
			if len(reordered) > rerankTopN {
				reordered = reordered[:rerankTopN]
			}

			log.Printf("[Chat] Step 3.5: Setelah rerank, memakai %d dokumen terbaik sebagai konteks\n", len(reordered))
			similarDocs = reordered
		}
	}

	// Step 4: Extract content from (reranked) similar documents and collect unique source files
	// Apply similarity threshold to filter out irrelevant results
	const similarityThreshold = 0.65 // Cosine distance threshold (0 = identical, 2 = opposite)
	// Documents with distance < 0.65 are considered relevant
	// Documents with distance >= 0.65 are too dissimilar and should be excluded
	// Note: Increased from 0.5 to 0.65 to be less strict for short queries

	var contextDocs []string
	var sourceIDs []int32
	uniqueSourceFiles := make(map[string]bool) // Map untuk deduplikasi nama file
	var uniqueSources []string                 // List nama file unik
	var filteredCount int                      // Count of documents filtered out

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

	// Step 5.5: Handle session persistence
	var currentSessionID int
	if req.SessionID != nil && *req.SessionID > 0 {
		// Use existing session
		currentSessionID = *req.SessionID
		log.Printf("[Chat] Step 5.5: Using existing session ID: %d\n", currentSessionID)

		// Save user message to database
		if err := db.SaveMessage(currentSessionID, "user", req.Question); err != nil {
			log.Printf("[Chat] WARNING: Failed to save user message: %v\n", err)
			// Continue anyway - don't fail the request
		}
	} else {
		// Create new session with first 30 characters of question as title
		title := req.Question
		if len(title) > 30 {
			title = title[:30] + "..."
		}
		if title == "" {
			title = "New Chat"
		}

		newSessionID, err := db.CreateSession(title)
		if err != nil {
			log.Printf("[Chat] WARNING: Failed to create session: %v\n", err)
			// Continue without session - don't fail the request
			currentSessionID = 0
		} else {
			currentSessionID = newSessionID
			log.Printf("[Chat] Step 5.5: Created new session ID: %d (title: %s)\n", currentSessionID, title)

			// Save user message to database
			if err := db.SaveMessage(currentSessionID, "user", req.Question); err != nil {
				log.Printf("[Chat] WARNING: Failed to save user message: %v\n", err)
			}
		}
	}

	// Send initial metadata event (sources information + session ID)
	// Kirim unique source file names, bukan content
	sourcesData := map[string]interface{}{
		"sources":   uniqueSources, // Nama file unik, bukan content
		"sourceIds": sourceIDs,
		"type":      "metadata",
	}
	if currentSessionID > 0 {
		sourcesData["sessionId"] = currentSessionID
	}
	sourcesJSON, _ := json.Marshal(sourcesData)
	fmt.Fprintf(c.Writer, "event: metadata\ndata: %s\n\n", sourcesJSON)
	c.Writer.Flush()

	// Step 6: Get streaming iterator
	// Use rewritten query for better context understanding
	log.Printf("[Chat] Step 6: Starting streaming response...\n")
	iter, err := utils.StreamChatResponse(rewrittenQuery, contextDocs, req.History)
	if err != nil {
		log.Printf("[Chat] ERROR DI STEP 6 (Stream Chat Response): %v\n", err)

		// Check if it's an invalid API key error
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "api key not valid") ||
			strings.Contains(errStr, "api_key_invalid") ||
			strings.Contains(errStr, "invalid api key") {
			log.Printf("[Chat] ERROR: Invalid API key detected in streaming")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Invalid API key",
				"message": "Please check your GEMINI_API_KEY in .env file. The API key is not valid or has expired.",
			})
			return
		}

		// For other errors, try to send error event via SSE
		// But if headers already sent, we can't change status code
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
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "done") ||
				strings.Contains(errStr, "eof") ||
				strings.Contains(errStr, "no more") {
				log.Printf("[Chat] Streaming completed. Total chunks: %d\n", chunkCount)
				break
			}

			// Check if it's an invalid API key error
			if strings.Contains(errStr, "api key not valid") ||
				strings.Contains(errStr, "api_key_invalid") ||
				strings.Contains(errStr, "invalid api key") {
				log.Printf("[Chat] ERROR: Invalid API key detected during streaming")
				errorData := map[string]string{
					"error":   "Invalid API key",
					"message": "Please check your GEMINI_API_KEY in .env file. The API key is not valid or has expired.",
					"type":    "error",
				}
				errorJSON, _ := json.Marshal(errorData)
				fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", errorJSON)
				c.Writer.Flush()
				return
			}

			// Check if it's a rate limit error - try to rotate key
			if strings.Contains(errStr, "429") ||
				strings.Contains(errStr, "quota exceeded") ||
				strings.Contains(errStr, "rate limit") {
				log.Printf("[Chat] WARNING: Rate limit detected during streaming")
				// Note: Can't rotate key mid-stream, but we can log it
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

	// Step 8: Save AI response to database (if session exists)
	if currentSessionID > 0 {
		aiResponse := fullResponse.String()
		if aiResponse != "" {
			if err := db.SaveMessage(currentSessionID, "model", aiResponse); err != nil {
				log.Printf("[Chat] WARNING: Failed to save AI message: %v\n", err)
				// Continue anyway - message is already sent to user
			} else {
				log.Printf("[Chat] Step 8: Saved AI response to session %d\n", currentSessionID)
			}
		}
	}

	// Send completion event
	log.Printf("[Chat] Step 9: Sending completion event...\n")
	completeData := map[string]interface{}{
		"type":        "done",
		"totalChunks": chunkCount,
		"fullLength":  fullResponse.Len(),
	}
	if currentSessionID > 0 {
		completeData["sessionId"] = currentSessionID
	}
	completeJSON, _ := json.Marshal(completeData)
	fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", completeJSON)
	c.Writer.Flush()

	log.Printf("[Chat] ===== Chat streaming completed successfully (total: %d chars, %d chunks, session: %d) =====\n", fullResponse.Len(), chunkCount, currentSessionID)

	// Return false to prevent Gin from writing additional JSON body
	// Note: In Gin, we don't explicitly return false, we just don't call c.JSON
	// The streaming response is already sent
}
