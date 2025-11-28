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

	// Get the generative model (using gemini-pro or gemini-1.5-flash)
	model := client.GenerativeModel("gemini-1.5-flash")

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
		for i, doc := range contextDocs {
			contextText += fmt.Sprintf("Dokumen %d:\n%s\n\n", i+1, doc)
		}
		contextText += "Gunakan informasi di atas untuk menjawab pertanyaan berikut. Jika informasi tidak cukup, katakan bahwa Anda tidak memiliki informasi yang cukup.\n\n"
	}

	// Build the prompt with history, context, and current question
	prompt := "Anda adalah asisten AI.\n\n"
	if historyText != "" {
		prompt += historyText
	}
	if contextText != "" {
		prompt += contextText
	}
	prompt += fmt.Sprintf("PERTANYAAN USER SAAT INI:\n%s\n\n", userQuery)
	prompt += "Jawablah pertanyaan user dengan mempertimbangkan riwayat percakapan di atas dan konteks dokumen."

	// Generate response
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
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

