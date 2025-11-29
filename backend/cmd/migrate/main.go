package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"backend/db"

	"github.com/joho/godotenv"
)

func loadEnvWithBOMHandling() {
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
		if err == nil {
			defer file.Close()
			content, err := io.ReadAll(file)
			if err == nil {
				content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
				content = bytes.TrimPrefix(content, []byte("\ufeff"))
				
				tempFile, err := os.CreateTemp("", ".env_cleaned_*")
				if err == nil {
					tempFile.Write(content)
					tempFile.Close()
					
					if err := godotenv.Load(tempFile.Name()); err == nil {
						log.Printf("Loaded .env from: %s (BOM removed)", envPath)
						os.Remove(tempFile.Name())
						return
					}
					os.Remove(tempFile.Name())
				}
			}
		}
		// Fallback to direct load
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("Loaded .env from: %s", envPath)
			return
		}
	}
	
	// Last resort: default load
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	} else {
		log.Println("Loaded .env using default search")
	}
}

func main() {
	// Load environment variables with BOM handling
	loadEnvWithBOMHandling()

	// Initialize database connection
	if err := db.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.CloseDB()

	// Read main migration file
	migrationPath := filepath.Join("db", "migration.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatal("Failed to read migration file:", err)
	}

	fmt.Println("Running migration...")
	fmt.Println("Migration SQL:")
	fmt.Println(string(migrationSQL))
	fmt.Println("---")

	// Execute main migration (will fail if table exists, that's OK)
	_, err = db.Pool.Exec(context.Background(), string(migrationSQL))
	if err != nil {
		// If table already exists, that's fine - continue to add missing columns
		if !strings.Contains(err.Error(), "already exists") {
			log.Printf("Warning: Main migration error (may be expected): %v", err)
		} else {
			fmt.Println("Table already exists, checking for missing columns...")
		}
	} else {
		fmt.Println("Main migration completed successfully!")
	}

	// Read and execute additional migration for source_file column
	addSourceFilePath := filepath.Join("db", "migration_add_source_file.sql")
	if _, err := os.Stat(addSourceFilePath); err == nil {
		addSourceFileSQL, err := os.ReadFile(addSourceFilePath)
		if err == nil {
			fmt.Println("\nRunning additional migration for source_file column...")
			_, err = db.Pool.Exec(context.Background(), string(addSourceFileSQL))
			if err != nil {
				log.Printf("Warning: Additional migration error (may be expected): %v", err)
			} else {
				fmt.Println("Additional migration completed successfully!")
			}
		}
	}

	fmt.Println("\nMigration process completed!")
}

