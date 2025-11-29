package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"backend/utils"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	// Load environment variables with BOM handling
	utils.LoadEnvWithBOMHandling()

	// Get API key from environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY is not set in environment variables")
	}

	fmt.Println("Connecting to Google Gemini API...")
	fmt.Println("API Key:", maskAPIKey(apiKey))
	fmt.Println()

	// Initialize Gemini client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	fmt.Println("Fetching available models...")
	fmt.Println()

	// List all available models
	iter := client.ListModels(ctx)
	
	var modelsWithGenerateContent []string
	var allModels []string

	for {
		model, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating models: %v", err)
			continue
		}

		allModels = append(allModels, model.Name)

		// Check if model supports generateContent
		if model.SupportedGenerationMethods != nil {
			for _, method := range model.SupportedGenerationMethods {
				if method == "generateContent" {
					modelsWithGenerateContent = append(modelsWithGenerateContent, model.Name)
					break
				}
			}
		}
	}

	// Print results
	separator := "============================================================"
	
	fmt.Println(separator)
	fmt.Println("MODELS YANG SUPPORT generateContent:")
	fmt.Println(separator)
	
	if len(modelsWithGenerateContent) == 0 {
		fmt.Println("Tidak ada model yang ditemukan dengan generateContent support")
	} else {
		fmt.Printf("Total: %d model(s)\n\n", len(modelsWithGenerateContent))
		for i, modelName := range modelsWithGenerateContent {
			// Extract model name without "models/" prefix for easier copy-paste
			shortName := modelName
			if len(modelName) > 7 && modelName[:7] == "models/" {
				shortName = modelName[7:]
			}
			fmt.Printf("%2d. %s\n", i+1, modelName)
			fmt.Printf("     → Gunakan: client.GenerativeModel(\"%s\")\n", shortName)
		}
	}

	fmt.Println()
	fmt.Println(separator)
	fmt.Println("REKOMENDASI MODEL UNTUK CHAT RAG:")
	fmt.Println(separator)
	
	// Filter recommended models
	recommended := []string{}
	recommendedNames := []string{
		"gemini-2.5-flash",
		"gemini-2.0-flash",
		"gemini-2.0-flash-001",
		"gemini-flash-latest",
		"gemini-pro-latest",
		"gemini-2.5-pro",
	}
	
	for _, modelName := range modelsWithGenerateContent {
		for _, recName := range recommendedNames {
			if len(modelName) > 7 && modelName[7:] == recName {
				recommended = append(recommended, modelName)
				break
			}
		}
	}
	
	if len(recommended) > 0 {
		for i, modelName := range recommended {
			shortName := modelName[7:] // Remove "models/" prefix
			fmt.Printf("%d. %s\n", i+1, shortName)
			fmt.Printf("   → client.GenerativeModel(\"%s\")\n", shortName)
		}
	} else {
		fmt.Println("Tidak ada model rekomendasi yang ditemukan")
	}

	fmt.Println()
	fmt.Println(separator)
	fmt.Println("CARA MENGGUNAKAN:")
	fmt.Println(separator)
	fmt.Println("1. Copy nama model dari list di atas (misal: gemini-2.5-flash)")
	fmt.Println("2. Gunakan di backend/utils/chat.go:")
	fmt.Println("   model := client.GenerativeModel(\"gemini-2.5-flash\")")
	fmt.Println("3. Hapus prefix 'models/' saat menggunakan di kode")
	fmt.Println()
}

// maskAPIKey masks the API key for security (show only first 10 and last 4 characters)
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 14 {
		return "***"
	}
	return apiKey[:10] + "..." + apiKey[len(apiKey)-4:]
}

