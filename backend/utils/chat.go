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
	prompt += "Jangan membuat nama file sendiri, gunakan nama file yang ada di label [Document: ...].\n"
	prompt += "\nATURAN ANTI-REPETISI (SANGAT PENTING):\n"
	prompt += "JANGAN PERNAH mengulang sitasi yang sama di setiap kalimat secara berurutan.\n"
	prompt += "Jika beberapa kalimat atau satu paragraf penuh berasal dari dokumen yang SAMA, letakkan sitasi HANYA SATU KALI di akhir kalimat terakhir atau akhir paragraf tersebut.\n"
	prompt += "HANYA gunakan sitasi per kalimat jika kalimat-kalimat tersebut berasal dari dokumen yang BERBEDA.\n"
	prompt += "\nCONTOH POLA JAWABAN YANG BENAR (TIRULAH INI):\n"
	prompt += "\nUser: \"Jelaskan tentang fitur login.\" Dokumen: [Login.pdf]\n"
	prompt += "\nJAWABAN SALAH (JANGAN SEPERTI INI):\n"
	prompt += "Fitur login menggunakan OAuth 2.0 (Login.pdf). Token akan kadaluarsa dalam 1 jam (Login.pdf). Jika gagal, user harus reset password (Login.pdf).\n"
	prompt += "\nJAWABAN BENAR (SEPERTI INI):\n"
	prompt += "Fitur login menggunakan OAuth 2.0 dan token akan kadaluarsa dalam 1 jam. Jika gagal login, user harus melakukan reset password melalui email (Login.pdf).\n"
	prompt += "\nContoh lain yang BENAR:\n"
	prompt += "'Bab 4 dari manual operasional membahas kontak darurat. Jika sistem mengalami kegagalan total, pengguna diminta menghubungi tim DevOps melalui email di devops-support@rag-system.internal atau melalui extension telepon #8812 (JUDUL_ MANUAL OPERASIONAL & KODE ETIK SISTEM RAG v1.pdf).'\n"
	prompt += "\nContoh yang SALAH (JANGAN LAKUKAN INI):\n"
	prompt += "'Bab 4 dari manual operasional membahas kontak darurat (JUDUL_ MANUAL OPERASIONAL & KODE ETIK SISTEM RAG v1.pdf). Jika sistem mengalami kegagalan total, pengguna diminta menghubungi tim DevOps melalui email di devops-support@rag-system.internal atau melalui extension telepon #8812 (JUDUL_ MANUAL OPERASIONAL & KODE ETIK SISTEM RAG v1.pdf).'\n"
	prompt += "\nATURAN UTAMA:\n"
	prompt += "1. Gabungkan informasi dari sumber yang sama menjadi satu paragraf yang mengalir.\n"
	prompt += "2. Letakkan sitasi (NamaFile) HANYA SATU KALI di akhir paragraf tersebut.\n"
	prompt += "3. HANYA jika informasi berikutnya berasal dari FILE YANG BERBEDA, barulah buat sitasi baru.\n"
	prompt += "\nINGAT: Jika semua informasi dalam satu paragraf atau beberapa kalimat berasal dari dokumen yang sama, cukup letakkan sitasi SATU KALI di akhir paragraf/kalimat terakhir.\n\n"
	if historyText != "" {
		prompt += historyText
	}
	if contextText != "" {
		prompt += contextText
	}
	prompt += fmt.Sprintf("PERTANYAAN USER SAAT INI:\n%s\n\n", userQuery)
	prompt += "Jawablah pertanyaan user dengan mempertimbangkan riwayat percakapan di atas dan konteks dokumen. Ingat: WAJIB sertakan citation (nama file) di akhir setiap kalimat atau paragraf yang mengandung fakta dari dokumen, TETAPI JANGAN PERNAH mengulang sitasi yang sama secara berurutan dalam satu paragraf. GABUNGKAN kalimat dari sumber yang sama menjadi paragraf yang mengalir, lalu berikan sitasi sekali di akhir."

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
	prompt += "Jangan membuat nama file sendiri, gunakan nama file yang ada di label [Document: ...].\n"
	prompt += "\nATURAN ANTI-REPETISI (SANGAT PENTING):\n"
	prompt += "JANGAN PERNAH mengulang sitasi yang sama di setiap kalimat secara berurutan.\n"
	prompt += "Jika beberapa kalimat atau satu paragraf penuh berasal dari dokumen yang SAMA, letakkan sitasi HANYA SATU KALI di akhir kalimat terakhir atau akhir paragraf tersebut.\n"
	prompt += "HANYA gunakan sitasi per kalimat jika kalimat-kalimat tersebut berasal dari dokumen yang BERBEDA.\n"
	prompt += "\nCONTOH POLA JAWABAN YANG BENAR (TIRULAH INI):\n"
	prompt += "\nUser: \"Jelaskan tentang fitur login.\" Dokumen: [Login.pdf]\n"
	prompt += "\nJAWABAN SALAH (JANGAN SEPERTI INI):\n"
	prompt += "Fitur login menggunakan OAuth 2.0 (Login.pdf). Token akan kadaluarsa dalam 1 jam (Login.pdf). Jika gagal, user harus reset password (Login.pdf).\n"
	prompt += "\nJAWABAN BENAR (SEPERTI INI):\n"
	prompt += "Fitur login menggunakan OAuth 2.0 dan token akan kadaluarsa dalam 1 jam. Jika gagal login, user harus melakukan reset password melalui email (Login.pdf).\n"
	prompt += "\nContoh lain yang BENAR:\n"
	prompt += "'Bab 4 dari manual operasional membahas kontak darurat. Jika sistem mengalami kegagalan total, pengguna diminta menghubungi tim DevOps melalui email di devops-support@rag-system.internal atau melalui extension telepon #8812 (JUDUL_ MANUAL OPERASIONAL & KODE ETIK SISTEM RAG v1.pdf).'\n"
	prompt += "\nContoh yang SALAH (JANGAN LAKUKAN INI):\n"
	prompt += "'Bab 4 dari manual operasional membahas kontak darurat (JUDUL_ MANUAL OPERASIONAL & KODE ETIK SISTEM RAG v1.pdf). Jika sistem mengalami kegagalan total, pengguna diminta menghubungi tim DevOps melalui email di devops-support@rag-system.internal atau melalui extension telepon #8812 (JUDUL_ MANUAL OPERASIONAL & KODE ETIK SISTEM RAG v1.pdf).'\n"
	prompt += "\nATURAN UTAMA:\n"
	prompt += "1. Gabungkan informasi dari sumber yang sama menjadi satu paragraf yang mengalir.\n"
	prompt += "2. Letakkan sitasi (NamaFile) HANYA SATU KALI di akhir paragraf tersebut.\n"
	prompt += "3. HANYA jika informasi berikutnya berasal dari FILE YANG BERBEDA, barulah buat sitasi baru.\n"
	prompt += "\nINGAT: Jika semua informasi dalam satu paragraf atau beberapa kalimat berasal dari dokumen yang sama, cukup letakkan sitasi SATU KALI di akhir paragraf/kalimat terakhir.\n\n"
	if historyText != "" {
		prompt += historyText
	}
	if contextText != "" {
		prompt += contextText
	}
	prompt += fmt.Sprintf("PERTANYAAN USER SAAT INI:\n%s\n\n", userQuery)
	prompt += "Jawablah pertanyaan user dengan mempertimbangkan riwayat percakapan di atas dan konteks dokumen. Ingat: WAJIB sertakan citation (nama file) di akhir setiap kalimat atau paragraf yang mengandung fakta dari dokumen, TETAPI JANGAN PERNAH mengulang sitasi yang sama secara berurutan dalam satu paragraf. GABUNGKAN kalimat dari sumber yang sama menjadi paragraf yang mengalir, lalu berikan sitasi sekali di akhir."

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

