package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

// ExtractTextFromFile extracts text from PDF or TXT files
func ExtractTextFromFile(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".pdf":
		return extractTextFromPDF(filePath)
	case ".txt":
		return extractTextFromTXT(filePath)
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

// extractTextFromPDF extracts text from PDF file
func extractTextFromPDF(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer file.Close()

	reader, err := model.NewPdfReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var textBuilder strings.Builder
	numPages, err := reader.GetNumPages()
	if err != nil {
		return "", fmt.Errorf("failed to get number of pages: %w", err)
	}

	for i := 1; i <= numPages; i++ {
		page, err := reader.GetPage(i)
		if err != nil {
			continue
		}

		ex, err := extractor.New(page)
		if err != nil {
			continue
		}

		pageText, err := ex.ExtractText()
		if err != nil {
			continue
		}

		textBuilder.WriteString(pageText)
		textBuilder.WriteString("\n")
	}

	return textBuilder.String(), nil
}

// extractTextFromTXT extracts text from TXT file
func extractTextFromTXT(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open TXT file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read TXT file: %w", err)
	}

	return string(content), nil
}

// SplitText splits a long text into chunks with overlap
// chunkSize: maximum characters per chunk
// overlap: number of characters to overlap between chunks
func SplitText(text string, chunkSize int, overlap int) []string {
	if len(text) == 0 {
		return []string{}
	}

	if chunkSize <= 0 {
		chunkSize = 1000 // Default chunk size
	}

	if overlap < 0 {
		overlap = 0
	}

	if overlap >= chunkSize {
		overlap = chunkSize / 5 // Prevent overlap from being too large
	}

	var chunks []string
	start := 0
	textLen := len(text)

	for start < textLen {
		end := start + chunkSize
		if end > textLen {
			end = textLen
		}

		// Extract chunk
		chunk := text[start:end]

		// Try to break at sentence boundary if not at the end
		if end < textLen {
			// Look for sentence endings within the last 100 characters
			searchStart := len(chunk) - 100
			if searchStart < 0 {
				searchStart = 0
			}

			// Find last sentence boundary (., !, ?, \n\n)
			lastPeriod := -1
			for i := len(chunk) - 1; i >= searchStart; i-- {
				if chunk[i] == '.' || chunk[i] == '!' || chunk[i] == '?' {
					// Check if followed by space or newline
					if i+1 < len(chunk) && (chunk[i+1] == ' ' || chunk[i+1] == '\n') {
						lastPeriod = i + 1
						break
					}
				} else if i+1 < len(chunk) && chunk[i] == '\n' && chunk[i+1] == '\n' {
					lastPeriod = i + 2
					break
				}
			}

			// If found sentence boundary, adjust chunk
			if lastPeriod > searchStart {
				chunk = chunk[:lastPeriod]
				end = start + len(chunk)
			}
		}

		chunks = append(chunks, chunk)

		// Move start position with overlap
		if end >= textLen {
			break
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}
