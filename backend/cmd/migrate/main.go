package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"backend/db"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database connection
	if err := db.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.CloseDB()

	// Read migration file
	migrationPath := filepath.Join("db", "migration.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatal("Failed to read migration file:", err)
	}

	fmt.Println("Running migration...")
	fmt.Println("Migration SQL:")
	fmt.Println(string(migrationSQL))
	fmt.Println("---")

	// Execute migration
	_, err = db.Pool.Exec(context.Background(), string(migrationSQL))
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	fmt.Println("Migration completed successfully!")
}

