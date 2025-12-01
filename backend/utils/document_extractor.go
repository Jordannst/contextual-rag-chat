package utils

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractTextFromFile extracts text from PDF, TXT, or DOCX files
func ExtractTextFromFile(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".pdf":
		return extractTextFromPDF(filePath)
	case ".txt":
		return extractTextFromTXT(filePath)
	case ".docx":
		return extractTextFromDocx(filePath)
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

// extractTextFromPDF extracts text from PDF file using github.com/ledongthuc/pdf
func extractTextFromPDF(filePath string) (string, error) {
	// Open PDF file - returns (*os.File, *Reader, error)
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer file.Close()

	// Get plain text reader from PDF
	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("failed to get plain text from PDF: %w", err)
	}

	// Read all text from reader
	textBytes, err := io.ReadAll(textReader)
	if err != nil {
		return "", fmt.Errorf("failed to read text from PDF: %w", err)
	}

	return string(textBytes), nil
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

// extractTextFromDocx extracts text from DOCX file by reading main document.xml
// This treats the DOCX as a ZIP archive and strips XML tags from word/document.xml.
func extractTextFromDocx(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open DOCX file: %w", err)
	}
	defer r.Close()

	var docXML []byte
	for _, f := range r.File {
		// Main document content
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("failed to open DOCX document.xml: %w", err)
			}
			defer rc.Close()

			docXML, err = io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("failed to read DOCX document.xml: %w", err)
			}
			break
		}
	}

	if len(docXML) == 0 {
		return "", fmt.Errorf("document.xml not found in DOCX file")
	}

	// Remove XML tags with a simple regex and unescape entities.
	// This is lightweight and good enough for plain text extraction.
	// Note: This is not a full XML parser, but works well for most docx bodies.
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAll(docXML, []byte(" "))
	text = bytes.ReplaceAll(text, []byte("\r"), []byte("\n"))

	// Collapse multiple spaces/newlines
	clean := strings.TrimSpace(html.UnescapeString(string(text)))
	clean = regexp.MustCompile(`\s+`).ReplaceAllString(clean, " ")

	return clean, nil
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
