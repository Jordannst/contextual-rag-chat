package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
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

