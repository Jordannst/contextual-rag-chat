package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"bytes"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: Could not get working directory: %v", err)
		wd = "."
	}
	
	// Try to find .env file - check if we're in backend/ or cmd/create-db/
	var envPath string
	possiblePaths := []string{
		filepath.Join(wd, ".env"),           // Current dir
		filepath.Join(wd, "..", ".env"),    // Parent (if in cmd/create-db/)
		filepath.Join(wd, "..", "..", ".env"), // Root (if deeper)
		".env",                              // Relative current
		"../.env",                           // Relative parent
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
		// Read file and remove BOM if present
		file, err := os.Open(envPath)
		if err != nil {
			log.Printf("Warning: Failed to open .env file: %v", err)
		} else {
			defer file.Close()
			
			// Read all content
			content, err := io.ReadAll(file)
			if err != nil {
				log.Printf("Warning: Failed to read .env file: %v", err)
			} else {
				// Remove BOM (UTF-8 BOM is 0xEF 0xBB 0xBF or \ufeff)
				content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
				content = bytes.TrimPrefix(content, []byte("\ufeff"))
				
				// Parse manually or use godotenv with cleaned content
				// Create a temporary approach: write cleaned content to temp file
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
					}
				}
			}
		}
	} else {
		log.Printf("Warning: .env file not found. Working directory: %s", wd)
		// Try default Load() as fallback
		if err := godotenv.Load(); err != nil {
			log.Println("Failed to load .env using default search")
		} else {
			log.Println("Loaded .env using default search")
		}
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set in .env file")
	}

	// Parse DATABASE_URL to get database name
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		log.Fatal("Failed to parse DATABASE_URL:", err)
	}

	dbName := strings.TrimPrefix(parsedURL.Path, "/")

	// Connect to default 'postgres' database
	// Remove query parameters and rebuild URL
	parsedURL.Path = "/postgres"
	parsedURL.RawQuery = ""
	defaultURL := parsedURL.String()

	conn, err := pgx.Connect(context.Background(), defaultURL)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL server:", err)
	}
	defer conn.Close(context.Background())

	fmt.Println("Connected to PostgreSQL server")

	// Check if database exists
	var exists bool
	err = conn.QueryRow(
		context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)",
		dbName,
	).Scan(&exists)

	if err != nil {
		log.Fatal("Failed to check database existence:", err)
	}

	if exists {
		fmt.Printf("Database \"%s\" already exists\n", dbName)
	} else {
		// Create database
		_, err = conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			log.Fatal("Failed to create database:", err)
		}
		fmt.Printf("Database \"%s\" created successfully\n", dbName)
	}
}

