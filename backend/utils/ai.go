package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GenerateEmbedding generates embedding vector from text using Google Gemini
// Returns embedding vector as []float32 and error if any
func GenerateEmbedding(text string) ([]float32, error) {
	// Get API key from environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set in environment variables")
	}

	// Initialize Gemini client with background context
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	// Get the embedding model
	model := client.EmbeddingModel("text-embedding-004")

	// Generate embedding from text
	embedding, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Check if embedding response is empty
	if embedding == nil || embedding.Embedding == nil {
		return nil, fmt.Errorf("embedding response is empty")
	}

	// Check if embedding values are empty
	if len(embedding.Embedding.Values) == 0 {
		return nil, fmt.Errorf("embedding values are empty")
	}

	// Return embedding values as []float32
	return embedding.Embedding.Values, nil
}

// GenerateQuestionSuggestions generates question suggestions based on document context
// Returns a slice of suggested questions as strings
func GenerateQuestionSuggestions(contextText string) ([]string, error) {
	// Get API key from environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set in environment variables")
	}

	// Initialize Gemini client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	// Get the generative model
	model := client.GenerativeModel("gemini-2.0-flash")

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

	// Generate response
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
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

