package db

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

var Pool *pgxpool.Pool

// InitDB initializes the database connection
func InitDB() error {
	// Note: .env loading is handled by main.go or calling code
	// This allows for centralized BOM handling

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
	ID         int32
	Content    string
	SourceFile string
	Distance   float64
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
// Returns empty slice if no documents found (no error)
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

	// Query using cosine distance directly
	// embedding <=> $1 returns cosine distance (0 = identical, 2 = opposite)
	// Smaller distance = more similar
	// ORDER BY embedding <=> $1 means cosine distance (ascending = most similar)
	query := `
		SELECT id, content, source_file, (embedding <=> $1) as distance
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
		if err := rows.Scan(&doc.ID, &doc.Content, &doc.SourceFile, &doc.Distance); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	// Return empty slice if no documents found (not an error)
	if len(documents) == 0 {
		return []Document{}, nil
	}

	return documents, nil
}

// GetUniqueDocuments returns a list of unique source file names from the database
func GetUniqueDocuments() ([]string, error) {
	if Pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	ctx := context.Background()

	// Query to get distinct source_file names, excluding NULL and empty strings
	query := `
		SELECT DISTINCT source_file
		FROM documents
		WHERE source_file IS NOT NULL AND source_file != ''
		ORDER BY source_file
	`

	rows, err := Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unique documents: %w", err)
	}
	defer rows.Close()

	var documents []string
	for rows.Next() {
		var sourceFile string
		if err := rows.Scan(&sourceFile); err != nil {
			return nil, fmt.Errorf("failed to scan source file: %w", err)
		}
		documents = append(documents, sourceFile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	// Return empty slice if no documents found (not an error)
	if len(documents) == 0 {
		return []string{}, nil
	}

	return documents, nil
}

// DeleteDocument deletes all chunks belonging to a specific source file
func DeleteDocument(fileName string) error {
	if Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	if fileName == "" {
		return fmt.Errorf("file name cannot be empty")
	}

	ctx := context.Background()

	// Delete all documents with matching source_file
	result, err := Pool.Exec(ctx, "DELETE FROM documents WHERE source_file = $1", fileName)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	rowsAffected := result.RowsAffected()
	fmt.Printf("Deleted %d chunks for file: %s\n", rowsAffected, fileName)

	return nil
}

