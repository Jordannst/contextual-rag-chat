package db

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pgvector/pgvector-go"
)

var Pool *pgxpool.Pool

// InitDB initializes the database connection
func InitDB() error {
	if err := godotenv.Load(); err != nil {
		// .env file is optional
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	// Clean up query parameters that might cause issues (like schema parameter)
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}
	
	// Remove query parameters that pgx doesn't support
	parsedURL.RawQuery = ""
	cleanURL := parsedURL.String()

	var poolErr error
	Pool, poolErr = pgxpool.New(context.Background(), cleanURL)
	if poolErr != nil {
		return fmt.Errorf("failed to create connection pool: %w", poolErr)
	}

	// Test connection
	if err := Pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Connected to PostgreSQL database")
	return nil
}

// CloseDB closes the database connection pool
func CloseDB() {
	if Pool != nil {
		Pool.Close()
	}
}

// Document represents a document from the database
type Document struct {
	ID       int32
	Content  string
	Distance float64
}

// InsertDocument inserts a document with its embedding vector into the database
func InsertDocument(content string, embedding []float32, sourceFile string) error {
	if Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	ctx := context.Background()

	// Convert []float32 to pgvector.Vector
	vector := pgvector.NewVector(embedding)

	// Execute INSERT query with source_file
	_, err := Pool.Exec(ctx, "INSERT INTO documents (content, embedding, source_file) VALUES ($1, $2, $3)", content, vector, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	return nil
}

// SearchSimilarDocuments searches for similar documents using cosine distance
// Returns top K most similar documents ordered by similarity
func SearchSimilarDocuments(queryEmbedding []float32, limit int) ([]Document, error) {
	if Pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	if limit <= 0 {
		limit = 5 // Default limit
	}

	ctx := context.Background()

	// Convert []float32 to pgvector.Vector
	queryVector := pgvector.NewVector(queryEmbedding)

	// Query using cosine distance (1 - cosine similarity)
	// ORDER BY embedding <=> $1 means cosine distance (ascending = most similar)
	query := `
		SELECT id, content, 1 - (embedding <=> $1) as distance
		FROM documents
		ORDER BY embedding <=> $1
		LIMIT $2
	`

	rows, err := Pool.Query(ctx, query, queryVector, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar documents: %w", err)
	}
	defer rows.Close()

	var documents []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.Content, &doc.Distance); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	return documents, nil
}

