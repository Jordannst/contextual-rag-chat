package utils

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// GenerateFilePreview generates a structured preview of a CSV/Excel file
// including column names and sample data for AI code generation
func GenerateFilePreview(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".csv":
		return generateCSVPreview(filePath)
	case ".xlsx", ".xls":
		// For Excel, we'll use Python to read it
		return generateExcelPreview(filePath)
	default:
		return "", fmt.Errorf("unsupported file type: %s (only .csv, .xlsx, .xls supported)", ext)
	}
}

// generateCSVPreview reads CSV file and generates preview
func generateCSVPreview(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		return "", fmt.Errorf("failed to read CSV header: %w", err)
	}
	
	// Read first 5 rows for sample
	var sampleRows [][]string
	maxSamples := 5
	totalRows := 0
	
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed rows
			continue
		}
		
		totalRows++
		if len(sampleRows) < maxSamples {
			sampleRows = append(sampleRows, row)
		}
	}
	
	// Build preview string
	var preview strings.Builder
	
	preview.WriteString("Struktur Data (CSV):\n")
	preview.WriteString(fmt.Sprintf("Kolom: %s\n\n", strings.Join(header, ", ")))
	
	if len(sampleRows) > 0 {
		preview.WriteString(fmt.Sprintf("Sample Data (%d baris pertama):\n", len(sampleRows)))
		
		// Build table header
		for i, col := range header {
			preview.WriteString(fmt.Sprintf("%-20s", col))
			if i < len(header)-1 {
				preview.WriteString(" | ")
			}
		}
		preview.WriteString("\n")
		
		// Build separator
		for i := range header {
			preview.WriteString(strings.Repeat("-", 20))
			if i < len(header)-1 {
				preview.WriteString("-+-")
			}
		}
		preview.WriteString("\n")
		
		// Build sample rows
		for _, row := range sampleRows {
			for i, val := range row {
				// Truncate long values
				displayVal := val
				if len(displayVal) > 18 {
					displayVal = displayVal[:15] + "..."
				}
				preview.WriteString(fmt.Sprintf("%-20s", displayVal))
				if i < len(row)-1 {
					preview.WriteString(" | ")
				}
			}
			preview.WriteString("\n")
		}
	}
	
	preview.WriteString(fmt.Sprintf("\nTotal rows: %d\n", totalRows))
	
	return preview.String(), nil
}

// generateExcelPreview uses Python to read Excel file and generate preview
func generateExcelPreview(filePath string) (string, error) {
	// Use Python to read Excel and get preview
	// This is simpler than using a Go Excel library
	
	pythonCode := `import pandas as pd
import sys

try:
    df = pd.read_excel(sys.argv[1])
    
    # Get column names
    columns = ', '.join(df.columns.tolist())
    print(f"Struktur Data (Excel):")
    print(f"Kolom: {columns}")
    print()
    
    # Get sample rows (first 5)
    sample_size = min(5, len(df))
    if sample_size > 0:
        print(f"Sample Data ({sample_size} baris pertama):")
        print(df.head(sample_size).to_string(index=False))
        print()
    
    print(f"Total rows: {len(df)}")
    
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
    sys.exit(1)
`
	
	// Create temporary Python script
	tmpScript, err := os.CreateTemp("", "excel_preview_*.py")
	if err != nil {
		return "", fmt.Errorf("failed to create temp script: %w", err)
	}
	defer os.Remove(tmpScript.Name())
	
	if _, err := tmpScript.WriteString(pythonCode); err != nil {
		return "", fmt.Errorf("failed to write temp script: %w", err)
	}
	tmpScript.Close()
	
	// Run Python script
	result, err := RunPythonAnalysis(filePath, pythonCode)
	if err != nil {
		// Fallback: just return basic info
		return fmt.Sprintf("Struktur Data (Excel):\nFile: %s\n\nNote: Unable to read Excel preview. Make sure pandas and openpyxl are installed.", filepath.Base(filePath)), nil
	}
	
	return result, nil
}

// GetQuickFileInfo returns basic file information without reading content
func GetQuickFileInfo(filePath string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}
	
	ext := strings.ToLower(filepath.Ext(filePath))
	fileType := "Unknown"
	
	switch ext {
	case ".csv":
		fileType = "CSV"
	case ".xlsx":
		fileType = "Excel (XLSX)"
	case ".xls":
		fileType = "Excel (XLS)"
	}
	
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("File: %s\n", filepath.Base(filePath)))
	buf.WriteString(fmt.Sprintf("Type: %s\n", fileType))
	buf.WriteString(fmt.Sprintf("Size: %.2f KB\n", float64(info.Size())/1024))
	
	return buf.String(), nil
}

// ValidateDataFile checks if a file is a valid CSV or Excel file
func ValidateDataFile(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found: %s", filePath)
	}
	
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".csv" && ext != ".xlsx" && ext != ".xls" {
		return fmt.Errorf("unsupported file type: %s (only .csv, .xlsx, .xls supported)", ext)
	}
	
	return nil
}

