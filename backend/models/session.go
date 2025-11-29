package models

import "time"

// ChatSession represents a chat session/conversation
type ChatSession struct {
	ID        int       `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ChatMessageDB represents a message stored in the database
type ChatMessageDB struct {
	ID        int       `json:"id" db:"id"`
	SessionID int       `json:"session_id" db:"session_id"`
	Role      string    `json:"role" db:"role"` // "user" or "model"
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

