package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
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

// extractChartData extracts [CHART_DATA:...] markers from Python output
// Returns: cleanOutput (without chart markers) and chartParts (array of base64 strings)
func extractChartData(output string) (string, []string) {
	// Pattern untuk mendeteksi [CHART_DATA:...base64...]
	// Format: [CHART_DATA:...] dimana ... adalah base64 string
	chartPattern := regexp.MustCompile(`\[CHART_DATA:([^\]]+)\]`)
	
	// Find all matches
	matches := chartPattern.FindAllStringSubmatch(output, -1)
	
	// Extract chart data (base64 strings)
	chartParts := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 2 {
			chartParts = append(chartParts, match[1]) // match[1] is the captured base64 string
		}
	}
	
	// Remove all chart markers from output
	cleanOutput := chartPattern.ReplaceAllString(output, "")
	
	// Clean up extra whitespace/newlines that might be left
	cleanOutput = strings.TrimSpace(cleanOutput)
	
	return cleanOutput, chartParts
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

	// Step 1.1: Detect file type and decide on flow (RAG vs Data Analyst)
	// STRICT BRANCHING: Check file extension to determine routing
	fileFilters := req.SelectedFiles
	var isDataAnalysisFlow bool
	var dataFilePaths map[string]string
	
	// Get file names to check (from SelectedFiles or auto-latest)
	var fileNamesToCheck []string
	if len(fileFilters) > 0 {
		// User explicitly selected files
		fileNamesToCheck = fileFilters
		log.Printf("[Chat] Step 1.1: Checking selected files: %v\n", fileFilters)
	} else {
		// No selected files - get latest uploaded file from database
		log.Printf("[Chat] Step 1.1: No file selected, fetching latest document from database...\n")
		latestFile, err := db.GetLatestDocument()
		if err != nil {
			log.Printf("[Chat] WARNING: Failed to get latest document: %v. Defaulting to RAG flow.\n", err)
			isDataAnalysisFlow = false
		} else if latestFile == "" {
			// No documents in database
			log.Printf("[Chat] Step 1.1: No documents found in database. Defaulting to RAG flow.\n")
			isDataAnalysisFlow = false
		} else {
			// Use latest file as default
			fileNamesToCheck = []string{latestFile}
			log.Printf("[Chat] Step 1.1: No file selected, defaulting to latest file: %s\n", latestFile)
		}
	}
	
	// STRICT BRANCHING: Check each file's extension
	if len(fileNamesToCheck) > 0 {
		var detectedDataFiles []string
		var detectedTextFiles []string
		
		// Categorize files by extension
		for _, fileName := range fileNamesToCheck {
			ext := utils.GetFileExtension(fileName)
			log.Printf("[Chat] Step 1.1: File: %s, Extension: %s\n", fileName, ext)
			
			if utils.IsDataFile(fileName) {
				detectedDataFiles = append(detectedDataFiles, fileName)
				log.Printf("[Chat] Step 1.1: ✅ Detected DATA FILE: %s (ext: %s)\n", fileName, ext)
			} else if utils.IsTextDocument(fileName) {
				detectedTextFiles = append(detectedTextFiles, fileName)
				log.Printf("[Chat] Step 1.1: ✅ Detected TEXT DOCUMENT: %s (ext: %s)\n", fileName, ext)
			} else {
				log.Printf("[Chat] Step 1.1: ⚠️ Unknown file type: %s (ext: %s) - Defaulting to RAG\n", fileName, ext)
			}
		}
		
		// STRICT DECISION: If ANY file is CSV/Excel, use Data Analyst flow
		// Only use RAG if ALL files are text documents (PDF/TXT/DOCX)
		if len(detectedDataFiles) > 0 {
			// At least one data file detected - use Data Analyst flow
			isDataAnalysisFlow = true
			log.Printf("[Chat] Step 1.1: 🎯 ROUTING DECISION: Data Analyst Flow (found %d data file(s): %v)\n", 
				len(detectedDataFiles), detectedDataFiles)
			
			// Get file paths for data files
			var err error
			dataFilePaths, err = utils.GetFilePathFromSourceFiles(detectedDataFiles)
			if err != nil {
				log.Printf("[Chat] WARNING: Failed to get some file paths: %v\n", err)
			}
			
			// Check if we have at least one valid file path
			if len(dataFilePaths) == 0 {
				log.Printf("[Chat] ERROR: No valid file paths found for data files! Falling back to RAG flow\n")
				isDataAnalysisFlow = false
			} else {
				log.Printf("[Chat] Step 1.1: ✅ Found %d valid data file path(s)\n", len(dataFilePaths))
			}
		} else if len(detectedTextFiles) > 0 {
			// Only text documents - use RAG flow
			isDataAnalysisFlow = false
			log.Printf("[Chat] Step 1.1: 🎯 ROUTING DECISION: RAG Flow (found %d text document(s): %v)\n", 
				len(detectedTextFiles), detectedTextFiles)
		} else {
			// Unknown file types - default to RAG
			isDataAnalysisFlow = false
			log.Printf("[Chat] Step 1.1: 🎯 ROUTING DECISION: RAG Flow (unknown file types)\n")
		}
	}
	
	// Final logging for debugging
	mode := "RAG"
	if isDataAnalysisFlow {
		mode = "Data Analyst"
	}
	if len(fileNamesToCheck) > 0 {
		firstFile := fileNamesToCheck[0]
		ext := utils.GetFileExtension(firstFile)
		log.Printf("[Chat] Step 1.1: 📊 FINAL ROUTING - File: %s, Extension: %s, Mode: %s\n", firstFile, ext, mode)
	} else {
		log.Printf("[Chat] Step 1.1: 📊 FINAL ROUTING - No files selected, Mode: %s\n", mode)
	}

	// Branch: Data Analyst Flow (CSV/Excel)
	if isDataAnalysisFlow {
		handleDataAnalysisFlow(c, &req, dataFilePaths)
		return
	}

	// Branch: RAG Flow (PDF/TXT/DOCX or default)
	// Continue with existing RAG logic below...

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

	// Use file filters (already set above)
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

// handleDataAnalysisFlow handles chat requests for CSV/Excel files using Data Analyst Agent
func handleDataAnalysisFlow(c *gin.Context, req *ChatRequest, dataFilePaths map[string]string) {
	log.Printf("[DataAnalyst] ===== Starting Data Analyst flow =====\n")
	
	// Step 1: Get the first data file (for now, we support one file at a time)
	var filePath string
	var sourceFileName string
	
	if len(dataFilePaths) == 0 {
		log.Printf("[DataAnalyst] ERROR: No valid file paths found\n")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No valid data files found. Please ensure CSV/Excel files are uploaded.",
		})
		return
	}
	
	// Get first file (we can extend to support multiple files later)
	for sourceName, path := range dataFilePaths {
		filePath = path
		sourceFileName = sourceName
		break
	}
	
	log.Printf("[DataAnalyst] Processing file: %s (path: %s)\n", sourceFileName, filePath)
	
	// Step 2: Generate file preview (structure + sample data)
	log.Printf("[DataAnalyst] Step 2: Generating file preview...\n")
	preview, err := utils.GenerateFilePreview(filePath)
	if err != nil {
		log.Printf("[DataAnalyst] ERROR: Failed to generate preview: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to read data file",
			"message": err.Error(),
		})
		return
	}
	log.Printf("[DataAnalyst] Step 2: Preview generated (length: %d chars)\n", len(preview))
	
	// Step 3: Generate Python code from user query using AI
	log.Printf("[DataAnalyst] Step 3: Generating Python code from query...\n")
	pythonCode, err := utils.GenerateAnalysisCode(req.Question, preview)
	if err != nil {
		log.Printf("[DataAnalyst] ERROR: Failed to generate code: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate analysis code",
			"message": err.Error(),
		})
		return
	}
	log.Printf("[DataAnalyst] Step 3: Generated code: %s\n", pythonCode)
	
	// Step 4: Validate generated code
	log.Printf("[DataAnalyst] Step 4: Validating generated code...\n")
	if err := utils.ValidatePythonCode(pythonCode); err != nil {
		log.Printf("[DataAnalyst] ERROR: Code validation failed: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Generated code contains unsafe operations",
			"message": err.Error(),
			"code":    pythonCode,
		})
		return
	}
	log.Printf("[DataAnalyst] Step 4: Code validation passed\n")
	
	// Step 5: Execute Python code
	log.Printf("[DataAnalyst] Step 5: Executing Python code...\n")
	pythonOutput, err := utils.RunPythonAnalysis(filePath, pythonCode)
	if err != nil {
		log.Printf("[DataAnalyst] ERROR: Code execution failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to execute analysis",
			"message": err.Error(),
			"code":    pythonCode,
		})
		return
	}
	log.Printf("[DataAnalyst] Step 5: Execution successful. Output length: %d chars\n", len(pythonOutput))
	
	// Step 5.5: Extract chart data from Python output to save tokens and prevent AI confusion
	log.Printf("[DataAnalyst] Step 5.5: Extracting chart data from Python output...\n")
	cleanOutput, chartParts := extractChartData(pythonOutput)
	log.Printf("[DataAnalyst] Step 5.5: Extracted %d chart(s). Clean output length: %d chars (saved %d chars)\n", 
		len(chartParts), len(cleanOutput), len(pythonOutput)-len(cleanOutput))
	
	// Step 6: Interpret Python output with AI (convert technical output to natural language)
	log.Printf("[DataAnalyst] Step 6: Interpreting Python output with AI...\n")
	
	// Build context for AI interpretation using cleanOutput (without chart data)
	// Format sesuai requirement: Plaintext dengan instruksi jelas
	// Inject Chart Awareness: Informasikan AI tentang keberadaan grafik
	var outputText string
	if cleanOutput == "" && len(chartParts) > 0 {
		// Jika output teks kosong tapi ada chart, tambahkan placeholder
		outputText = "Visualisasi grafik telah berhasil dibuat."
		log.Printf("[DataAnalyst] Step 6: Clean output is empty but charts exist, adding placeholder text\n")
	} else {
		outputText = cleanOutput
	}
	
	// Build base context
	interpretationContext := fmt.Sprintf(`[HASIL ANALISIS DATA PROGRAMMATIK]

Dokumen: %s

Hasil Eksekusi Python:

%s`, sourceFileName, outputText)
	
	// Add chart awareness if charts exist
	if len(chartParts) > 0 {
		chartInfo := fmt.Sprintf("\n\n[SYSTEM INFO]: Sebanyak %d grafik visual telah berhasil di-generate dan dikirim ke user secara terpisah. Gunakan data teks di atas untuk menjelaskan insight grafik tersebut. JANGAN bilang tidak ada grafik atau tidak ada informasi.", len(chartParts))
		interpretationContext += chartInfo
		log.Printf("[DataAnalyst] Step 6: Injected chart awareness (%d chart(s))\n", len(chartParts))
	}
	
	// Add instructions
	interpretationContext += `

INSTRUKSI:
Jelaskan hasil analisis data di atas kepada user dengan bahasa yang natural, ringkas, dan mudah dimengerti. Jangan tampilkan kode atau struktur data mentah kecuali diminta.`
	
	// Convert history to models.ChatMessage format
	history := make([]models.ChatMessage, 0, len(req.History))
	for _, msg := range req.History {
		history = append(history, models.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	// Use StreamChatResponse to interpret the Python output
	// We'll use the user's original question as the query, with the Python output as context
	interpretationQuery := req.Question
	
	// Step 7: Handle session persistence (same as RAG flow)
	var currentSessionID int
	if req.SessionID != nil && *req.SessionID > 0 {
		currentSessionID = *req.SessionID
		log.Printf("[DataAnalyst] Using existing session ID: %d\n", currentSessionID)
		
		if err := db.SaveMessage(currentSessionID, "user", req.Question); err != nil {
			log.Printf("[DataAnalyst] WARNING: Failed to save user message: %v\n", err)
		}
	} else {
		title := req.Question
		if len(title) > 30 {
			title = title[:30] + "..."
		}
		if title == "" {
			title = "Data Analysis"
		}
		
		newSessionID, err := db.CreateSession(title)
		if err != nil {
			log.Printf("[DataAnalyst] WARNING: Failed to create session: %v\n", err)
			currentSessionID = 0
		} else {
			currentSessionID = newSessionID
			log.Printf("[DataAnalyst] Created new session ID: %d\n", currentSessionID)
			
			if err := db.SaveMessage(currentSessionID, "user", req.Question); err != nil {
				log.Printf("[DataAnalyst] WARNING: Failed to save user message: %v\n", err)
			}
		}
	}
	
	// Step 8: Stream interpreted response using AI (using SSE format for consistency with RAG flow)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	
	// Send metadata event
	sourcesData := map[string]interface{}{
		"sources":   []string{sourceFileName},
		"type":      "metadata",
		"analysis":  true, // Flag to indicate this is data analysis, not RAG
		"code":      pythonCode,
	}
	if currentSessionID > 0 {
		sourcesData["sessionId"] = currentSessionID
	}
	sourcesJSON, _ := json.Marshal(sourcesData)
	fmt.Fprintf(c.Writer, "event: metadata\ndata: %s\n\n", sourcesJSON)
	c.Writer.Flush()
	
	// Step 8.1: Get streaming iterator for AI interpretation
	// Pass the interpretation context as a single context document
	contextDocs := []string{interpretationContext}
	
	iter, err := utils.StreamChatResponse(interpretationQuery, contextDocs, history)
	if err != nil {
		log.Printf("[DataAnalyst] ERROR: Failed to start streaming interpretation: %v\n", err)
		
		// Fallback: send clean Python output if interpretation fails
		log.Printf("[DataAnalyst] WARNING: Falling back to clean Python output\n")
		fallbackResponse := cleanOutput
		if fallbackResponse == "" {
			fallbackResponse = "Tidak ada hasil yang ditemukan."
		}
		
		lines := strings.Split(fallbackResponse, "\n")
		for _, line := range lines {
			if line != "" {
				chunkData := map[string]string{
					"chunk": line + "\n",
					"type":  "chunk",
				}
				chunkJSON, _ := json.Marshal(chunkData)
				fmt.Fprintf(c.Writer, "data: %s\n\n", chunkJSON)
				c.Writer.Flush()
			}
		}
		
		// Send chart data in fallback case too
		if len(chartParts) > 0 {
			log.Printf("[DataAnalyst] Sending %d chart(s) in fallback...\n", len(chartParts))
			for i, chartData := range chartParts {
				chartEvent := map[string]interface{}{
					"type":      "chart",
					"chartData": chartData,
					"index":     i,
				}
				chartJSON, _ := json.Marshal(chartEvent)
				fmt.Fprintf(c.Writer, "event: chart\ndata: %s\n\n", chartJSON)
				c.Writer.Flush()
			}
		}
		
		// Save fallback response
		if currentSessionID > 0 {
			if err := db.SaveMessage(currentSessionID, "model", fallbackResponse); err != nil {
				log.Printf("[DataAnalyst] WARNING: Failed to save AI message: %v\n", err)
			}
		}
		
		// Send completion event
		completeData := map[string]interface{}{
			"type":       "done",
			"fullLength": len(fallbackResponse),
			"analysis":   true,
			"chartCount": len(chartParts),
		}
		if currentSessionID > 0 {
			completeData["sessionId"] = currentSessionID
		}
		completeJSON, _ := json.Marshal(completeData)
		fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", completeJSON)
		c.Writer.Flush()
		
		log.Printf("[DataAnalyst] ===== Data Analyst flow completed with fallback =====\n")
		return
	}
	
	// Step 8.2: Stream chunks from iterator (same as RAG flow)
	log.Printf("[DataAnalyst] Step 8.2: Streaming interpreted response...\n")
	var fullResponse strings.Builder
	chunkCount := 0
	
	for {
		// Get next chunk from iterator
		resp, err := iter.Next()
		if err != nil {
			// Check if iteration is done
			if err == iterator.Done {
				log.Printf("[DataAnalyst] Streaming completed. Total chunks: %d\n", chunkCount)
				break
			}
			
			// Check for other "done" indicators (fallback)
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "done") ||
				strings.Contains(errStr, "eof") ||
				strings.Contains(errStr, "no more") {
				log.Printf("[DataAnalyst] Streaming completed. Total chunks: %d\n", chunkCount)
				break
			}
			
			// Real error occurred
			log.Printf("[DataAnalyst] ERROR during streaming: %v\n", err)
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
								log.Printf("[DataAnalyst] ERROR marshaling chunk: %v\n", err)
								continue
							}
							
							// Send with SSE format: data: <json>\n\n
							fmt.Fprintf(c.Writer, "data: %s\n\n", chunkJSON)
							c.Writer.Flush()
							
							// Accumulate for logging and saving
							fullResponse.WriteString(text)
							chunkCount++
							
							log.Printf("[DataAnalyst] Chunk %d sent (length: %d)\n", chunkCount, len(text))
						}
					}
				}
			}
		}
	}
	
	// Step 9: Send chart data if any (before saving to DB)
	if len(chartParts) > 0 {
		log.Printf("[DataAnalyst] Step 9: Sending %d chart(s) to frontend...\n", len(chartParts))
		for i, chartData := range chartParts {
			chartEvent := map[string]interface{}{
				"type":      "chart",
				"chartData": chartData,
				"index":     i,
			}
			chartJSON, _ := json.Marshal(chartEvent)
			fmt.Fprintf(c.Writer, "event: chart\ndata: %s\n\n", chartJSON)
			c.Writer.Flush()
			log.Printf("[DataAnalyst] Chart %d sent (length: %d chars)\n", i+1, len(chartData))
		}
	}
	
	// Step 10: Save AI response to database (without chart data)
	if currentSessionID > 0 {
		aiResponse := fullResponse.String()
		if aiResponse != "" {
			if err := db.SaveMessage(currentSessionID, "model", aiResponse); err != nil {
				log.Printf("[DataAnalyst] WARNING: Failed to save AI message: %v\n", err)
			} else {
				log.Printf("[DataAnalyst] Step 10: Saved AI response to session %d\n", currentSessionID)
			}
		}
	}
	
	// Send completion event
	log.Printf("[DataAnalyst] Step 11: Sending completion event...\n")
	completeData := map[string]interface{}{
		"type":        "done",
		"totalChunks":  chunkCount,
		"fullLength":  fullResponse.Len(),
		"analysis":    true,
		"chartCount":  len(chartParts),
	}
	if currentSessionID > 0 {
		completeData["sessionId"] = currentSessionID
	}
	completeJSON, _ := json.Marshal(completeData)
	fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", completeJSON)
	c.Writer.Flush()
	
	log.Printf("[DataAnalyst] ===== Data Analyst flow completed successfully (total: %d chars, %d chunks, session: %d) =====\n", fullResponse.Len(), chunkCount, currentSessionID)
}
