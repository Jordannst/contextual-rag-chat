package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetFilePathFromSourceFile finds the actual file path in uploads/ directory
// based on the source_file name stored in database
// sourceFileName: the original filename (e.g., "data.csv")
// Returns: full path to the file (e.g., "uploads/data-1234567890.csv")
func GetFilePathFromSourceFile(sourceFileName string) (string, error) {
	if sourceFileName == "" {
		return "", fmt.Errorf("source file name cannot be empty")
	}

	uploadsDir := "uploads"
	
	// Check if uploads directory exists
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		return "", fmt.Errorf("uploads directory does not exist")
	}

	// Get file extension
	ext := filepath.Ext(sourceFileName)
	nameWithoutExt := strings.TrimSuffix(sourceFileName, ext)

	// Search for files in uploads directory that match the pattern
	// Files are stored with timestamp: name-timestamp.ext
	files, err := os.ReadDir(uploadsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read uploads directory: %w", err)
	}

	// Find file that matches the source_file name pattern
	// Pattern: name-timestamp.ext (where name is from sourceFileName)
	var foundFile string
	patternPrefix := nameWithoutExt + "-"

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		// Check if file matches pattern: starts with name- and ends with .ext
		if strings.HasPrefix(fileName, patternPrefix) && strings.HasSuffix(strings.ToLower(fileName), strings.ToLower(ext)) {
			foundFile = fileName
			break
		}
	}

	if foundFile == "" {
		return "", fmt.Errorf("file not found for source: %s (pattern: %s*%s)", sourceFileName, patternPrefix, ext)
	}

	filePath := filepath.Join(uploadsDir, foundFile)
	return filePath, nil
}

// GetFilePathFromSourceFiles finds file paths for multiple source files
// Returns a map: sourceFileName -> filePath
func GetFilePathFromSourceFiles(sourceFileNames []string) (map[string]string, error) {
	result := make(map[string]string)
	
	for _, sourceFileName := range sourceFileNames {
		filePath, err := GetFilePathFromSourceFile(sourceFileName)
		if err != nil {
			// Log error but continue with other files
			continue
		}
		result[sourceFileName] = filePath
	}
	
	return result, nil
}

