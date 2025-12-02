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

// RunChatSessionsMigration runs the chat sessions migration if tables don't exist
func RunChatSessionsMigration() error {
	ctx := context.Background()
	
	// Check if chat_sessions table exists
	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'chat_sessions'
		)
	`
	err := Pool.QueryRow(ctx, checkQuery).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if chat_sessions table exists: %w", err)
	}
	
	if exists {
		fmt.Println("Chat sessions tables already exist, skipping migration")
		return nil
	}
	
	// Run migration
	migrationSQL := `
		-- Table: chat_sessions
		CREATE TABLE IF NOT EXISTS chat_sessions (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);

		-- Table: chat_messages
		CREATE TABLE IF NOT EXISTS chat_messages (
			id SERIAL PRIMARY KEY,
			session_id INTEGER NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
			role TEXT NOT NULL CHECK (role IN ('user', 'model')),
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);

		-- Index for faster queries
		CREATE INDEX IF NOT EXISTS idx_chat_messages_session_id ON chat_messages(session_id);
		CREATE INDEX IF NOT EXISTS idx_chat_sessions_created_at ON chat_sessions(created_at DESC);
	`
	
	_, err = Pool.Exec(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to run chat sessions migration: %w", err)
	}
	
	fmt.Println("Chat sessions migration completed successfully")
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
// Note: text_search column is automatically populated by database trigger
// The trigger (trigger_update_text_search) will create tsvector from content
func InsertDocument(content string, embedding []float32, sourceFile string) error {
	if Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	ctx := context.Background()

	// Convert []float32 to pgvector.Vector
	vector := pgvector.NewVector(embedding)

	// Execute INSERT query with source_file
	// text_search will be automatically populated by trigger
	_, err := Pool.Exec(ctx, "INSERT INTO documents (content, embedding, source_file) VALUES ($1, $2, $3)", content, vector, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	return nil
}

// SearchSimilarDocuments searches for similar documents using cosine distance
// Returns top K most similar documents ordered by similarity
// Returns empty slice if no documents found (no error)
// fileFilters: optional list of source_file names to filter by. If empty, searches all files.
func SearchSimilarDocuments(queryEmbedding []float32, limit int, fileFilters []string) ([]Document, error) {
	if Pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	if limit <= 0 {
		limit = 5 // Default limit
	}

	ctx := context.Background()

	// Convert []float32 to pgvector.Vector
	queryVector := pgvector.NewVector(queryEmbedding)

	// Build query with optional file filter
	var query string
	var args []interface{}
	
	if len(fileFilters) > 0 {
		// Query with file filter: WHERE source_file = ANY($3)
		query = `
			SELECT id, content, source_file, (embedding <=> $1) as distance
			FROM documents
			WHERE source_file = ANY($3)
			ORDER BY embedding <=> $1
			LIMIT $2
		`
		args = []interface{}{queryVector, limit, fileFilters}
	} else {
		// Query without filter: search all files
		query = `
			SELECT id, content, source_file, (embedding <=> $1) as distance
			FROM documents
			ORDER BY embedding <=> $1
			LIMIT $2
		`
		args = []interface{}{queryVector, limit}
	}

	rows, err := Pool.Query(ctx, query, args...)
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

// SearchHybridDocuments performs hybrid search combining vector similarity and full-text search
// queryEmbedding: vector embedding for semantic search
// queryText: text query for full-text search (will be converted to tsquery)
// limit: maximum number of results to return
// vectorWeight: weight for vector search (0.0 to 1.0), textWeight = 1.0 - vectorWeight
// fileFilters: optional list of source_file names to filter by. If empty, searches all files.
// Returns documents sorted by combined score
func SearchHybridDocuments(queryEmbedding []float32, queryText string, limit int, vectorWeight float64, fileFilters []string) ([]Document, error) {
	if Pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	if limit <= 0 {
		limit = 5 // Default limit
	}

	// Normalize weights
	if vectorWeight < 0 {
		vectorWeight = 0
	}
	if vectorWeight > 1 {
		vectorWeight = 1
	}
	textWeight := 1.0 - vectorWeight

	// Default to 0.7 vector, 0.3 text if not specified
	if vectorWeight == 0 && textWeight == 0 {
		vectorWeight = 0.7
		textWeight = 0.3
	}

	ctx := context.Background()

	// Convert []float32 to pgvector.Vector
	queryVector := pgvector.NewVector(queryEmbedding)

	// Convert query text to tsquery format
	// This handles multiple words: "search term" becomes "search & term"
	// Using plainto_tsquery for user-friendly input (handles phrases naturally)
	var query string
	var args []interface{}
	
	if len(fileFilters) > 0 {
		// Query with file filter: WHERE text_search @@ ... AND source_file = ANY($6)
		query = `
			SELECT 
				id, 
				content, 
				source_file,
				(embedding <=> $1) as vector_distance,
				ts_rank(text_search, plainto_tsquery('english', $2)) as text_rank,
				-- Combined score: lower vector_distance is better, higher text_rank is better
				-- Normalize: (1 - vector_distance/2) for vector, text_rank for text
				((1 - (embedding <=> $1) / 2.0) * $3 + ts_rank(text_search, plainto_tsquery('english', $2)) * $4) as combined_score
			FROM documents
			WHERE text_search @@ plainto_tsquery('english', $2)
				AND source_file = ANY($6)
			ORDER BY combined_score DESC
			LIMIT $5
		`
		args = []interface{}{queryVector, queryText, vectorWeight, textWeight, limit, fileFilters}
	} else {
		// Query without filter: search all files
		query = `
			SELECT 
				id, 
				content, 
				source_file,
				(embedding <=> $1) as vector_distance,
				ts_rank(text_search, plainto_tsquery('english', $2)) as text_rank,
				-- Combined score: lower vector_distance is better, higher text_rank is better
				-- Normalize: (1 - vector_distance/2) for vector, text_rank for text
				((1 - (embedding <=> $1) / 2.0) * $3 + ts_rank(text_search, plainto_tsquery('english', $2)) * $4) as combined_score
			FROM documents
			WHERE text_search @@ plainto_tsquery('english', $2)
			ORDER BY combined_score DESC
			LIMIT $5
		`
		args = []interface{}{queryVector, queryText, vectorWeight, textWeight, limit}
	}

	rows, err := Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to perform hybrid search: %w", err)
	}
	defer rows.Close()

	var documents []Document
	for rows.Next() {
		var doc Document
		var vectorDist, textRank, combinedScore float64
		if err := rows.Scan(&doc.ID, &doc.Content, &doc.SourceFile, &vectorDist, &textRank, &combinedScore); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		// Store vector distance as Distance (for compatibility with existing code)
		doc.Distance = vectorDist
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

// SearchDocuments is a convenience function that automatically chooses between
// vector-only search or hybrid search based on whether queryText is provided
// If queryText is empty, uses vector-only search (SearchSimilarDocuments)
// If queryText is provided, uses hybrid search (SearchHybridDocuments)
// vectorWeight: weight for vector search in hybrid mode (default: 0.7)
// fileFilters: optional list of source_file names to filter by. If empty, searches all files.
func SearchDocuments(queryEmbedding []float32, queryText string, limit int, vectorWeight float64, fileFilters []string) ([]Document, error) {
	if queryText == "" {
		// Use vector-only search if no text query provided
		return SearchSimilarDocuments(queryEmbedding, limit, fileFilters)
	}
	
	// Use hybrid search if text query is provided
	return SearchHybridDocuments(queryEmbedding, queryText, limit, vectorWeight, fileFilters)
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

// GetRandomContext retrieves random document chunks for generating question suggestions
// Returns a slice of content strings from random documents
func GetRandomContext(limit int) ([]string, error) {
	if Pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	if limit <= 0 {
		limit = 5 // Default limit
	}

	ctx := context.Background()

	// Query to get random document chunks
	// ORDER BY RANDOM() is PostgreSQL-specific for random ordering
	query := `
		SELECT content
		FROM documents
		WHERE content IS NOT NULL AND content != ''
		ORDER BY RANDOM()
		LIMIT $1
	`

	rows, err := Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query random context: %w", err)
	}
	defer rows.Close()

	var contexts []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, fmt.Errorf("failed to scan content: %w", err)
		}
		contexts = append(contexts, content)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating contexts: %w", err)
	}

	// Return empty slice if no documents found (not an error)
	if len(contexts) == 0 {
		return []string{}, nil
	}

	return contexts, nil
}

