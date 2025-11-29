package utils

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnvWithBOMHandling loads .env file with BOM (Byte Order Mark) handling
// Tries multiple locations and removes BOM before parsing
func LoadEnvWithBOMHandling() {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	var envPath string
	possiblePaths := []string{
		filepath.Join(wd, ".env"),
		filepath.Join(wd, "..", ".env"),
		filepath.Join(wd, "..", "..", ".env"),
		".env",
		"../.env",
	}

	for _, path := range possiblePaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				envPath = absPath
				break
			}
		}
	}

	if envPath != "" {
		file, err := os.Open(envPath)
		if err != nil {
			log.Printf("Warning: Failed to open .env file: %v", err)
			// Fallback to direct load
			if err := godotenv.Load(envPath); err == nil {
				log.Printf("Loaded .env from: %s", envPath)
				return
			}
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			log.Printf("Warning: Failed to read .env file: %v", err)
			// Fallback to direct load
			if err := godotenv.Load(envPath); err == nil {
				log.Printf("Loaded .env from: %s", envPath)
			}
			return
		}

		// Remove BOM (UTF-8 BOM is 0xEF 0xBB 0xBF or \ufeff)
		content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
		content = bytes.TrimPrefix(content, []byte("\ufeff"))

		// Write cleaned content to temp file and load it
		tempFile, err := os.CreateTemp("", ".env_cleaned_*")
		if err == nil {
			tempFile.Write(content)
			tempFile.Close()

			if err := godotenv.Load(tempFile.Name()); err != nil {
				log.Printf("Warning: Failed to load .env: %v", err)
			} else {
				log.Printf("Loaded .env from: %s (BOM removed)", envPath)
			}
			os.Remove(tempFile.Name())
		} else {
			// Fallback: try direct load
			if err := godotenv.Load(envPath); err != nil {
				log.Printf("Warning: Failed to load .env: %v", err)
			} else {
				log.Printf("Loaded .env from: %s", envPath)
			}
		}
	} else {
		log.Printf("Warning: .env file not found. Working directory: %s", wd)
		// Last resort: default load
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found")
		} else {
			log.Println("Loaded .env using default search")
		}
	}
}

