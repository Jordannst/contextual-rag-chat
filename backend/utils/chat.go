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
	prompt := "[INSTRUKSI UTAMA]\n"
	prompt += "Anda adalah asisten AI cerdas untuk sistem RAG. Tugas Anda adalah menjawab pertanyaan pengguna berdasarkan konteks dokumen yang diberikan.\n\n"
	prompt += "[ATURAN RESPON - EKSEKUSI LANGSUNG]\n\n"
	prompt += "JANGAN PERNAH menuliskan teks seperti \"Kategori: ...\", \"Jenis input: ...\", \"Ini adalah pertanyaan...\", \"Pertanyaan ini termasuk dalam kategori...\", atau sejenisnya. LANGSUNG berikan jawaban intinya.\n\n"
	prompt += "JIKA input adalah Small Talk (Halo, Terima kasih, Baik, Oke, Baiklah, Siap, Mengerti, Paham, Tidak ada, Bye, Sampai jumpa):\n"
	prompt += "   - Jawab dengan sopan, singkat, dan natural.\n"
	prompt += "   - DILARANG menggunakan sitasi/referensi dokumen.\n"
	prompt += "   - Contoh: User: \"Baiklah\" → AI: \"Oke. Silakan tanya lagi jika butuh bantuan.\"\n"
	prompt += "   - Contoh: User: \"Terima kasih\" → AI: \"Sama-sama! Beritahu saya jika ada hal lain yang perlu dibahas.\"\n"
	prompt += "   - Contoh: User: \"Tidak ada\" → AI: \"Oke, siap. Jangan ragu menghubungi saya lagi nanti.\"\n\n"
	prompt += "JIKA input adalah Pertanyaan tentang Dokumen (Apa itu..., Siapa..., Bagaimana..., Jelaskan..., Apa prosedur..., atau permintaan lanjut seperti \"Lanjutkan\", \"Terus?\"):\n"
	prompt += "   - Jawab lengkap berdasarkan konteks dokumen.\n"
	prompt += "   - Kelompokkan penjelasan per dokumen.\n"
	prompt += "   - Letakkan sitasi (NamaFile) HANYA SATU KALI di akhir paragraf penjelasan dokumen tersebut.\n"
	prompt += "   - Contoh: User: \"Apa prosedur login?\" → AI: \"Prosedur login menggunakan OAuth 2.0 sebagai metode autentikasi. User harus memasukkan email dan password, lalu sistem akan mengirim token akses. Token akan kadaluarsa dalam 1 jam dan harus diperbarui untuk melanjutkan sesi. Jika login gagal 3 kali berturut-turut, akun akan terkunci sementara. (Login.pdf)\"\n\n"
	prompt += "[ATURAN SITASI PER-SEKSI]\n"
	prompt += "1. JANGAN menaruh sitasi `(NamaFile)` di setiap kalimat. Itu dilarang.\n"
	prompt += "2. Kelompokkan penjelasan berdasarkan sumber dokumennya.\n"
	prompt += "3. Tuliskan seluruh penjelasan dari satu dokumen sampai selesai dalam satu blok/paragraf.\n"
	prompt += "4. Letakkan sitasi `(NamaFile)` HANYA SATU KALI di **akhir total** penjelasan untuk dokumen tersebut.\n"
	prompt += "\nCONTOH POLA YANG BENAR:\n"
	prompt += "User: \"Jelaskan tentang pendaftaran dan sanksi.\" Dokumen: [SOP_Pendaftaran.pdf, Aturan_Sanksi.pdf]\n"
	prompt += "\nJAWABAN BENAR:\n"
	prompt += "\"Dokumen pertama membahas tentang tata cara pendaftaran. Pengguna harus mengisi form A, lalu upload KTP, dan menunggu verifikasi 2x24 jam. Jika gagal, hubungi admin. (SOP_Pendaftaran.pdf)\n"
	prompt += "\nSementara itu, dokumen kedua menjelaskan tentang sanksi pelanggaran. Pelanggaran ringan kena teguran, sedangkan berat langsung blokir akun. (Aturan_Sanksi.pdf)\"\n\n"
	prompt += "JIKA informasi tidak ada di dokumen:\n"
	prompt += "   - Katakan dengan jujur \"Tidak ditemukan informasi di dokumen\".\n"
	prompt += "   - Jangan mengarang jawaban.\n\n"
	if historyText != "" {
		prompt += historyText
	}
	if contextText != "" {
		prompt += contextText
	}
	prompt += fmt.Sprintf("PERTANYAAN USER SAAT INI:\n%s\n\n", userQuery)
	prompt += "Jawablah pertanyaan user dengan natural dan profesional. JANGAN menuliskan kategori, klasifikasi, atau proses internal apapun. Langsung berikan jawaban intinya."

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
	prompt := "[INSTRUKSI UTAMA]\n"
	prompt += "Anda adalah asisten AI cerdas untuk sistem RAG. Tugas Anda adalah menjawab pertanyaan pengguna berdasarkan konteks dokumen yang diberikan.\n\n"
	prompt += "[ATURAN RESPON - EKSEKUSI LANGSUNG]\n\n"
	prompt += "JANGAN PERNAH menuliskan teks seperti \"Kategori: ...\", \"Jenis input: ...\", \"Ini adalah pertanyaan...\", \"Pertanyaan ini termasuk dalam kategori...\", atau sejenisnya. LANGSUNG berikan jawaban intinya.\n\n"
	prompt += "JIKA input adalah Small Talk (Halo, Terima kasih, Baik, Oke, Baiklah, Siap, Mengerti, Paham, Tidak ada, Bye, Sampai jumpa):\n"
	prompt += "   - Jawab dengan sopan, singkat, dan natural.\n"
	prompt += "   - DILARANG menggunakan sitasi/referensi dokumen.\n"
	prompt += "   - Contoh: User: \"Baiklah\" → AI: \"Oke. Silakan tanya lagi jika butuh bantuan.\"\n"
	prompt += "   - Contoh: User: \"Terima kasih\" → AI: \"Sama-sama! Beritahu saya jika ada hal lain yang perlu dibahas.\"\n"
	prompt += "   - Contoh: User: \"Tidak ada\" → AI: \"Oke, siap. Jangan ragu menghubungi saya lagi nanti.\"\n\n"
	prompt += "JIKA input adalah Pertanyaan tentang Dokumen (Apa itu..., Siapa..., Bagaimana..., Jelaskan..., Apa prosedur..., atau permintaan lanjut seperti \"Lanjutkan\", \"Terus?\"):\n"
	prompt += "   - Jawab lengkap berdasarkan konteks dokumen.\n"
	prompt += "   - Kelompokkan penjelasan per dokumen.\n"
	prompt += "   - Letakkan sitasi (NamaFile) HANYA SATU KALI di akhir paragraf penjelasan dokumen tersebut.\n"
	prompt += "   - Contoh: User: \"Apa prosedur login?\" → AI: \"Prosedur login menggunakan OAuth 2.0 sebagai metode autentikasi. User harus memasukkan email dan password, lalu sistem akan mengirim token akses. Token akan kadaluarsa dalam 1 jam dan harus diperbarui untuk melanjutkan sesi. Jika login gagal 3 kali berturut-turut, akun akan terkunci sementara. (Login.pdf)\"\n\n"
	prompt += "[ATURAN SITASI PER-SEKSI]\n"
	prompt += "1. JANGAN menaruh sitasi `(NamaFile)` di setiap kalimat. Itu dilarang.\n"
	prompt += "2. Kelompokkan penjelasan berdasarkan sumber dokumennya.\n"
	prompt += "3. Tuliskan seluruh penjelasan dari satu dokumen sampai selesai dalam satu blok/paragraf.\n"
	prompt += "4. Letakkan sitasi `(NamaFile)` HANYA SATU KALI di **akhir total** penjelasan untuk dokumen tersebut.\n"
	prompt += "\nCONTOH POLA YANG BENAR:\n"
	prompt += "User: \"Jelaskan tentang pendaftaran dan sanksi.\" Dokumen: [SOP_Pendaftaran.pdf, Aturan_Sanksi.pdf]\n"
	prompt += "\nJAWABAN BENAR:\n"
	prompt += "\"Dokumen pertama membahas tentang tata cara pendaftaran. Pengguna harus mengisi form A, lalu upload KTP, dan menunggu verifikasi 2x24 jam. Jika gagal, hubungi admin. (SOP_Pendaftaran.pdf)\n"
	prompt += "\nSementara itu, dokumen kedua menjelaskan tentang sanksi pelanggaran. Pelanggaran ringan kena teguran, sedangkan berat langsung blokir akun. (Aturan_Sanksi.pdf)\"\n\n"
	prompt += "JIKA informasi tidak ada di dokumen:\n"
	prompt += "   - Katakan dengan jujur \"Tidak ditemukan informasi di dokumen\".\n"
	prompt += "   - Jangan mengarang jawaban.\n\n"
	if historyText != "" {
		prompt += historyText
	}
	if contextText != "" {
		prompt += contextText
	}
	prompt += fmt.Sprintf("PERTANYAAN USER SAAT INI:\n%s\n\n", userQuery)
	prompt += "Jawablah pertanyaan user dengan natural dan profesional. JANGAN menuliskan kategori, klasifikasi, atau proses internal apapun. Langsung berikan jawaban intinya."

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

