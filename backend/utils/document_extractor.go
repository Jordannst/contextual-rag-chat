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

