package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"backend/models"

	"github.com/google/generative-ai-go/genai"
)

// GenerateEmbedding generates embedding vector from text using Google Gemini
// Returns embedding vector as []float32 and error if any
func GenerateEmbedding(text string) ([]float32, error) {
	ctx := context.Background()
	keyManager := GetKeyManager()

	var result []float32
	var lastErr error

	err := keyManager.ExecuteWithRetry(ctx, func(client *genai.Client) error {
		// Get the embedding model
		model := client.EmbeddingModel("text-embedding-004")

		// Generate embedding from text
		embedding, err := model.EmbedContent(ctx, genai.Text(text))
		if err != nil {
			lastErr = err
			return err
		}

		// Check if embedding response is empty
		if embedding == nil || embedding.Embedding == nil {
			lastErr = fmt.Errorf("embedding response is empty")
			return lastErr
		}

		// Check if embedding values are empty
		if len(embedding.Embedding.Values) == 0 {
			lastErr = fmt.Errorf("embedding values are empty")
			return lastErr
		}

		// Store result
		result = embedding.Embedding.Values
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("embedding generation failed: %w", lastErr)
	}

	// Return embedding values as []float32
	return result, nil
}

// GenerateQuestionSuggestions generates question suggestions based on document context
// Returns a slice of suggested questions as strings
func GenerateQuestionSuggestions(contextText string) ([]string, error) {
	ctx := context.Background()
	keyManager := GetKeyManager()

	// Build prompt for question suggestions
	prompt := fmt.Sprintf(`Berdasarkan teks berikut, buatkan 3-4 pertanyaan pendek dan menarik yang mungkin ditanyakan user tentang dokumen ini.

Teks dokumen:
%s

Instruksi:
- Buat pertanyaan yang spesifik dan relevan dengan isi dokumen
- Gunakan bahasa Indonesia
- Pertanyaan harus singkat (maksimal 15 kata)
- Format output: JSON array of strings, contoh: ["Pertanyaan 1?", "Pertanyaan 2?", "Pertanyaan 3?"]
- Hanya return JSON array, tanpa penjelasan tambahan

Contoh format output:
["Apa tujuan utama dari dokumen ini?", "Bagaimana cara menggunakan fitur X?", "Apa saja persyaratan yang diperlukan?"]`, contextText)

	var resp *genai.GenerateContentResponse
	err := keyManager.ExecuteWithRetry(ctx, func(client *genai.Client) error {
		// Get the generative model
		model := client.GenerativeModel("gemini-2.0-flash")
		
		// Generate response
		var genErr error
		resp, genErr = model.GenerateContent(ctx, genai.Text(prompt))
		if genErr != nil {
			return genErr
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate question suggestions: %w", err)
	}

	// Extract text from response
	if resp.Candidates == nil || len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response candidates")
	}

	if resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response content")
	}

	// Get text from the first part
	var responseText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			responseText.WriteString(string(textPart))
		}
	}

	if responseText.Len() == 0 {
		return nil, fmt.Errorf("no text in response")
	}

	// Parse JSON array from response
	text := strings.TrimSpace(responseText.String())
	
	// Remove markdown code blocks if present
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	// Try to find JSON array in the response
	// Look for array pattern: [...]
	startIdx := strings.Index(text, "[")
	endIdx := strings.LastIndex(text, "]")
	
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		jsonText := text[startIdx : endIdx+1]
		var questions []string
		if err := json.Unmarshal([]byte(jsonText), &questions); err == nil {
			// Successfully parsed JSON
			// Limit to 4 questions max
			if len(questions) > 4 {
				questions = questions[:4]
			}
			// Filter out empty questions
			var validQuestions []string
			for _, q := range questions {
				q = strings.TrimSpace(q)
				if q != "" {
					validQuestions = append(validQuestions, q)
				}
			}
			if len(validQuestions) > 0 {
				return validQuestions, nil
			}
		}
	}

	// If parsing failed, return default questions
	return []string{
		"Apa topik utama dari dokumen ini?",
		"Bisa jelaskan lebih detail tentang isi dokumen?",
		"Apa saja poin penting yang perlu diketahui?",
	}, nil
}

// RewriteQuery rewrites an ambiguous follow-up question into a standalone, complete query
// using conversation history for context. This improves RAG accuracy for follow-up questions.
func RewriteQuery(history []models.ChatMessage, currentQuery string) (string, error) {
	// If no history, return original query (no need to rewrite)
	if len(history) == 0 {
		return currentQuery, nil
	}

	ctx := context.Background()
	keyManager := GetKeyManager()

	// Build conversation history text
	historyText := ""
	for _, msg := range history {
		if msg.Role == "user" {
			historyText += fmt.Sprintf("User: %s\n", msg.Content)
		} else if msg.Role == "model" {
			historyText += fmt.Sprintf("Model: %s\n", msg.Content)
		}
	}

	// Build prompt for query rewriting
	prompt := fmt.Sprintf(`Diberikan riwayat percakapan berikut dan pertanyaan terbaru dari user, tulis ulang (rewrite) pertanyaan terbaru agar menjadi kalimat lengkap, berdiri sendiri, dan tidak ambigu. 

Jangan menjawab pertanyaannya, hanya tulis ulang pertanyaannya menjadi pertanyaan yang lengkap dan jelas.

Riwayat Percakapan:
%s

Pertanyaan User Terbaru: %s

Standalone Query (tulis ulang pertanyaan menjadi lengkap dan jelas):`, historyText, currentQuery)

	var resp *genai.GenerateContentResponse
	err := keyManager.ExecuteWithRetry(ctx, func(client *genai.Client) error {
		// Get the generative model (using Flash for speed)
		model := client.GenerativeModel("gemini-2.0-flash")
		
		// Generate response
		var genErr error
		resp, genErr = model.GenerateContent(ctx, genai.Text(prompt))
		if genErr != nil {
			return genErr
		}
		return nil
	})

	if err != nil {
		// If rewriting fails, return original query as fallback
		return currentQuery, fmt.Errorf("failed to rewrite query: %w", err)
	}

	// Extract text from response
	if resp.Candidates == nil || len(resp.Candidates) == 0 {
		return currentQuery, fmt.Errorf("no response candidates for query rewriting")
	}

	if resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return currentQuery, fmt.Errorf("empty response content for query rewriting")
	}

	// Get text from the first part
	var responseText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			responseText.WriteString(string(textPart))
		}
	}

	rewrittenQuery := strings.TrimSpace(responseText.String())
	
	// If rewritten query is empty, return original
	if rewrittenQuery == "" {
		return currentQuery, nil
	}

	return rewrittenQuery, nil
}

// GenerateAnalysisCode generates Python pandas code to answer user's data analysis question
// userQuery: The question from user (e.g., "Berapa total penjualan?")
// filePreview: Structure of the data (column names and sample data)
// Returns: Python code string that can be executed directly
func GenerateAnalysisCode(userQuery string, filePreview string) (string, error) {
	ctx := context.Background()
	keyManager := GetKeyManager()

	// Build prompt for code generation
	prompt := fmt.Sprintf("Anda adalah Data Analyst Python yang expert dalam Pandas.\n\n" +
		"Diberikan struktur data berikut:\n%s\n\n" +
		"Pertanyaan User:\n%s\n\n" +
		"Instruksi:\n" +
		"- Tulis kode Python Pandas untuk menjawab pertanyaan user\n" +
		"- Variable dataframe bernama 'df' SUDAH TERSEDIA (tidak perlu import atau load data)\n" +
		"- Variable 'pd' (pandas) dan 'np' (numpy) SUDAH TERSEDIA\n" +
		"- HANYA berikan kode Python yang dapat dieksekusi\n" +
		"- JANGAN gunakan markdown code blocks\n" +
		"- JANGAN berikan penjelasan atau komentar\n" +
		"- LANGSUNG berikan kodenya saja\n" +
		"- PASTIKAN hasil akhir dicetak menggunakan print()\n" +
		"- Jika hasil berupa angka, format dengan 2 desimal jika perlu\n" +
		"- Jika hasil berupa tabel/series, gunakan print() untuk menampilkannya\n\n" +
		"Contoh output yang BENAR:\n" +
		"print(df['Total'].sum())\n\n" +
		"Contoh output yang SALAH (JANGAN SEPERTI INI):\n" +
		"Jangan pakai markdown wrapper atau triple backticks\n\n" +
		"Sekarang tulis kode Python untuk menjawab pertanyaan user:", filePreview, userQuery)

	var resp *genai.GenerateContentResponse
	err := keyManager.ExecuteWithRetry(ctx, func(client *genai.Client) error {
		// Get the generative model (using Flash for speed)
		model := client.GenerativeModel("gemini-2.0-flash")
		
		// Configure model for code generation
		model.SetTemperature(0.1) // Low temperature for more deterministic code
		
		// Generate response
		var genErr error
		resp, genErr = model.GenerateContent(ctx, genai.Text(prompt))
		if genErr != nil {
			return genErr
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate analysis code: %w", err)
	}

	// Extract text from response
	if resp.Candidates == nil || len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response candidates for code generation")
	}

	if resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response content for code generation")
	}

	// Get text from the first part
	var responseText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			responseText.WriteString(string(textPart))
		}
	}

	code := strings.TrimSpace(responseText.String())
	
	// Clean up code - remove markdown code blocks if AI ignores instruction
	code = strings.TrimPrefix(code, "```python")
	code = strings.TrimPrefix(code, "```py")
	code = strings.TrimPrefix(code, "```")
	code = strings.TrimSuffix(code, "```")
	code = strings.TrimSpace(code)

	// Validate that code is not empty
	if code == "" {
		return "", fmt.Errorf("generated code is empty")
	}

	// Validate that code contains print statement
	if !strings.Contains(code, "print") {
		// If no print, wrap the code
		// Try to detect if it's an expression
		lines := strings.Split(code, "\n")
		if len(lines) == 1 && !strings.Contains(code, "=") {
			// Single line without assignment, likely an expression
			code = fmt.Sprintf("print(%s)", code)
		} else {
			// Multiple lines or contains assignment, add print at the end
			// This is a best-effort approach
			code = code + "\nprint('Code executed successfully')"
		}
	}

	return code, nil
}

