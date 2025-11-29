package utils

import (
	"context"
	"fmt"
	"os"
	"strings"

	"backend/models"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GenerateChatResponse generates a chat response using Gemini with RAG context and conversation history
func GenerateChatResponse(userQuery string, contextDocs []string, history []models.ChatMessage) (string, error) {
	// Get API key from environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY is not set in environment variables")
	}

	// Initialize Gemini client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	// Get the generative model
	// Using gemini-2.0-flash (confirmed available and supports generateContent)
	// Fallback chain: gemini-2.0-flash-001 -> gemini-flash-latest -> gemini-2.5-flash
	model := client.GenerativeModel("gemini-2.0-flash")

	// Build conversation history
	historyText := ""
	if len(history) > 0 {
		historyText = "RIWAYAT PERCAKAPAN:\n\n"
		for _, msg := range history {
			if msg.Role == "user" {
				historyText += fmt.Sprintf("User: %s\n", msg.Content)
			} else if msg.Role == "model" {
				historyText += fmt.Sprintf("Model: %s\n", msg.Content)
			}
		}
		historyText += "\n"
	}

	// Build context from retrieved documents
	contextText := ""
	if len(contextDocs) > 0 {
		contextText = "KONTEKS DOKUMEN (RAG):\n\n"
		for _, doc := range contextDocs {
			contextText += fmt.Sprintf("%s\n\n", doc)
		}
		contextText += "Gunakan informasi di atas untuk menjawab pertanyaan berikut. Jika informasi tidak cukup, katakan bahwa Anda tidak memiliki informasi yang cukup.\n\n"
	}

	// Build the prompt with history, context, and current question
	prompt := "Anda adalah asisten AI yang menjawab berdasarkan dokumen yang diberikan.\n\n"
	prompt += "PENTING - ATURAN CITATION:\n"
	prompt += "Setiap kali Anda memberikan fakta atau penjelasan, Anda WAJIB menyertakan nama dokumen sumbernya di akhir kalimat atau paragraf dalam tanda kurung.\n"
	prompt += "Contoh format citation: '(nama_file.pdf)' atau '(file_lain.txt)'.\n"
	prompt += "Gunakan nama file persis seperti yang tertera di label [Document: ...] di konteks dokumen di atas.\n"
	prompt += "Jika informasi berasal dari gabungan beberapa file, sebutkan semuanya, contoh: '(file1.pdf, file2.txt)'.\n"
	prompt += "Jangan membuat nama file sendiri, gunakan nama file yang ada di label [Document: ...].\n\n"
	if historyText != "" {
		prompt += historyText
	}
	if contextText != "" {
		prompt += contextText
	}
	prompt += fmt.Sprintf("PERTANYAAN USER SAAT INI:\n%s\n\n", userQuery)
	prompt += "Jawablah pertanyaan user dengan mempertimbangkan riwayat percakapan di atas dan konteks dokumen. Ingat: WAJIB sertakan citation (nama file) di akhir setiap kalimat atau paragraf yang mengandung fakta dari dokumen."

	// Generate response with fallback chain
	var resp *genai.GenerateContentResponse
	var genErr error
	
	modelsToTry := []string{"gemini-2.0-flash", "gemini-2.0-flash-001", "gemini-flash-latest", "gemini-2.5-flash"}
	
	for i, modelName := range modelsToTry {
		if i == 0 {
			// Use primary model (already created)
			resp, genErr = model.GenerateContent(ctx, genai.Text(prompt))
		} else {
			// Try fallback models
			fmt.Printf("[Chat] Warning: Failed with previous model, trying fallback %s: %v\n", modelName, genErr)
			fallbackModel := client.GenerativeModel(modelName)
			resp, genErr = fallbackModel.GenerateContent(ctx, genai.Text(prompt))
		}
		
		if genErr == nil {
			if i > 0 {
				fmt.Printf("[Chat] Successfully used fallback model: %s\n", modelName)
			}
			break
		}
	}
	
	if genErr != nil {
		return "", fmt.Errorf("failed to generate response (tried %s): %w", strings.Join(modelsToTry, ", "), genErr)
	}

	// Extract text from response
	if resp.Candidates == nil || len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response candidates")
	}

	if resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response content")
	}

	// Get text from the first part
	var responseText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			responseText.WriteString(string(textPart))
		}
	}

	if responseText.Len() == 0 {
		return "", fmt.Errorf("no text in response")
	}

	return responseText.String(), nil
}

// StreamChatResponse generates a streaming chat response using Gemini with RAG context and conversation history
// Returns an iterator for streaming responses
func StreamChatResponse(userQuery string, contextDocs []string, history []models.ChatMessage) (*genai.GenerateContentResponseIterator, error) {
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
	// Note: Don't defer Close() here as the iterator needs the client to stay alive
	// The caller should handle cleanup

	// Build conversation history
	historyText := ""
	if len(history) > 0 {
		historyText = "RIWAYAT PERCAKAPAN:\n\n"
		for _, msg := range history {
			if msg.Role == "user" {
				historyText += fmt.Sprintf("User: %s\n", msg.Content)
			} else if msg.Role == "model" {
				historyText += fmt.Sprintf("Model: %s\n", msg.Content)
			}
		}
		historyText += "\n"
	}

	// Build context from retrieved documents
	contextText := ""
	if len(contextDocs) > 0 {
		contextText = "KONTEKS DOKUMEN (RAG):\n\n"
		for _, doc := range contextDocs {
			contextText += fmt.Sprintf("%s\n\n", doc)
		}
		contextText += "Gunakan informasi di atas untuk menjawab pertanyaan berikut. Jika informasi tidak cukup, katakan bahwa Anda tidak memiliki informasi yang cukup.\n\n"
	}

	// Build the prompt with history, context, and current question
	prompt := "Anda adalah asisten AI yang menjawab berdasarkan dokumen yang diberikan.\n\n"
	prompt += "PENTING - ATURAN CITATION:\n"
	prompt += "Setiap kali Anda memberikan fakta atau penjelasan, Anda WAJIB menyertakan nama dokumen sumbernya di akhir kalimat atau paragraf dalam tanda kurung.\n"
	prompt += "Contoh format citation: '(nama_file.pdf)' atau '(file_lain.txt)'.\n"
	prompt += "Gunakan nama file persis seperti yang tertera di label [Document: ...] di konteks dokumen di atas.\n"
	prompt += "Jika informasi berasal dari gabungan beberapa file, sebutkan semuanya, contoh: '(file1.pdf, file2.txt)'.\n"
	prompt += "Jangan membuat nama file sendiri, gunakan nama file yang ada di label [Document: ...].\n\n"
	if historyText != "" {
		prompt += historyText
	}
	if contextText != "" {
		prompt += contextText
	}
	prompt += fmt.Sprintf("PERTANYAAN USER SAAT INI:\n%s\n\n", userQuery)
	prompt += "Jawablah pertanyaan user dengan mempertimbangkan riwayat percakapan di atas dan konteks dokumen. Ingat: WAJIB sertakan citation (nama file) di akhir setiap kalimat atau paragraf yang mengandung fakta dari dokumen."

	// Get the generative model
	// Using gemini-2.0-flash (confirmed available and supports generateContent)
	// Fallback chain: gemini-2.0-flash-001 -> gemini-flash-latest -> gemini-2.5-flash
	model := client.GenerativeModel("gemini-2.0-flash")

	// Generate streaming response
	iter := model.GenerateContentStream(ctx, genai.Text(prompt))
	
	// Note: If streaming fails, we might need to handle fallback
	// For now, we'll return the iterator and let the handler deal with errors
	
	return iter, nil
}

