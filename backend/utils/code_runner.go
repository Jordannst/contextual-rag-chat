package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// CodeExecutionError represents an error from Python code execution
type CodeExecutionError struct {
	Error string `json:"error"`
}

// RunPythonAnalysis executes Python code on a data file (CSV/Excel)
// filePath: path to the CSV or Excel file
// pythonCode: Python code string to execute (e.g., "print(df['Harga'].mean())")
// Returns: output string from stdout, or error
func RunPythonAnalysis(filePath string, pythonCode string) (string, error) {
	// Tentukan path ke script Python - coba beberapa lokasi
	possiblePaths := []string{
		filepath.Join("scripts", "code_interpreter.py"),           // Dari backend/
		filepath.Join("backend", "scripts", "code_interpreter.py"), // Dari root
		filepath.Join("..", "scripts", "code_interpreter.py"),     // Dari cmd/
	}
	
	var scriptPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			scriptPath = path
			break
		}
	}
	
	if scriptPath == "" {
		// Try absolute path based on current working directory
		cwd, _ := os.Getwd()
		absPath := filepath.Join(cwd, "scripts", "code_interpreter.py")
		if _, err := os.Stat(absPath); err == nil {
			scriptPath = absPath
		}
	}
	
	if scriptPath == "" {
		return "", fmt.Errorf("code_interpreter.py tidak ditemukan. Cek lokasi: %v", possiblePaths)
	}
	
	// Tentukan command Python berdasarkan OS
	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}
	
	// Buat command
	cmd := exec.Command(pythonCmd, scriptPath, filePath, pythonCode)
	
	// Buffer untuk stdout dan stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Jalankan command
	err := cmd.Run()
	
	// Ambil output dari stderr untuk logging
	stderrStr := stderr.String()
	
	if err != nil {
		// Cek apakah ada error JSON dari Python
		if strings.Contains(stderrStr, `{"error":`) {
			// Parse error JSON
			var codeErr CodeExecutionError
			lines := strings.Split(stderrStr, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "{") {
					if jsonErr := json.Unmarshal([]byte(line), &codeErr); jsonErr == nil {
						return "", fmt.Errorf("Python execution error: %s", codeErr.Error)
					}
				}
			}
		}
		
		// Fallback ke error message biasa
		return "", fmt.Errorf("failed to execute Python code: %v\nStderr: %s", err, stderrStr)
	}
	
	// Ambil output dari stdout
	output := stdout.String()
	
	// Decode UTF-8 jika diperlukan
	output = strings.TrimSpace(output)
	
	return output, nil
}

// RunPythonAnalysisWithLogging is similar to RunPythonAnalysis but returns stderr logs as well
// Useful for debugging
func RunPythonAnalysisWithLogging(filePath string, pythonCode string) (output string, logs string, err error) {
	// Gunakan logic yang sama dengan RunPythonAnalysis
	possiblePaths := []string{
		filepath.Join("scripts", "code_interpreter.py"),
		filepath.Join("backend", "scripts", "code_interpreter.py"),
		filepath.Join("..", "scripts", "code_interpreter.py"),
	}
	
	var scriptPath string
	for _, path := range possiblePaths {
		if _, statErr := os.Stat(path); statErr == nil {
			scriptPath = path
			break
		}
	}
	
	if scriptPath == "" {
		cwd, _ := os.Getwd()
		absPath := filepath.Join(cwd, "scripts", "code_interpreter.py")
		if _, err := os.Stat(absPath); err == nil {
			scriptPath = absPath
		}
	}
	
	if scriptPath == "" {
		return "", "", fmt.Errorf("code_interpreter.py tidak ditemukan")
	}
	
	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}
	
	cmd := exec.Command(pythonCmd, scriptPath, filePath, pythonCode)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	execErr := cmd.Run()
	
	stderrStr := stderr.String()
	logs = stderrStr
	
	if execErr != nil {
		if strings.Contains(stderrStr, `{"error":`) {
			var codeErr CodeExecutionError
			lines := strings.Split(stderrStr, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "{") {
					if jsonErr := json.Unmarshal([]byte(line), &codeErr); jsonErr == nil {
						return "", logs, fmt.Errorf("Python execution error: %s", codeErr.Error)
					}
				}
			}
		}
		
		return "", logs, fmt.Errorf("failed to execute Python code: %v\nStderr: %s", execErr, stderrStr)
	}
	
	output = strings.TrimSpace(stdout.String())
	
	return output, logs, nil
}

// fileExists checks if a file exists (removed - using os.Stat directly now)

// ValidatePythonCode performs basic validation on Python code
// Returns error if code contains dangerous patterns
func ValidatePythonCode(code string) error {
	// Daftar pattern berbahaya
	dangerousPatterns := []string{
		"import os",
		"import sys",
		"import subprocess",
		"__import__",
		"eval(",
		"exec(",
		"compile(",
		"open(",
		"file(",
		"input(",
	}
	
	codeLower := strings.ToLower(code)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(codeLower, pattern) {
			return fmt.Errorf("kode tidak diizinkan: mengandung '%s'. Hanya operasi pandas yang diperbolehkan", pattern)
		}
	}
	
	return nil
}

// Example usage patterns
/*
Example 1: Calculate mean of a column
	result, err := RunPythonAnalysis("data.csv", "print(df['Harga'].mean())")

Example 2: Get dataframe info
	result, err := RunPythonAnalysis("data.csv", "print(df.describe())")

Example 3: Count rows
	result, err := RunPythonAnalysis("data.csv", "print(len(df))")

Example 4: Filter and count
	result, err := RunPythonAnalysis("data.csv", "print(len(df[df['Status'] == 'Active']))")

Example 5: Get column names
	result, err := RunPythonAnalysis("data.csv", "print(', '.join(df.columns))")
*/

