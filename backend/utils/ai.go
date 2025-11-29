package utils

import (
	"context"
	"fmt"
	"os"

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

