package db

import (
	"context"
	"fmt"
	"backend/models"
)

// CreateSession creates a new chat session and returns its ID
func CreateSession(title string) (int, error) {
	query := `INSERT INTO chat_sessions (title) VALUES ($1) RETURNING id`
	
	var sessionID int
	err := Pool.QueryRow(context.Background(), query, title).Scan(&sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to create session: %w", err)
	}
	
	return sessionID, nil
}

// GetSessions retrieves all chat sessions ordered by created_at DESC
func GetSessions() ([]models.ChatSession, error) {
	query := `SELECT id, title, created_at FROM chat_sessions ORDER BY created_at DESC`
	
	rows, err := Pool.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()
	
	var sessions []models.ChatSession
	for rows.Next() {
		var session models.ChatSession
		if err := rows.Scan(&session.ID, &session.Title, &session.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}
	
	return sessions, nil
}

// GetSessionMessages retrieves all messages for a specific session ordered by created_at ASC
func GetSessionMessages(sessionID int) ([]models.ChatMessageDB, error) {
	query := `SELECT id, session_id, role, content, created_at 
	          FROM chat_messages 
	          WHERE session_id = $1 
	          ORDER BY created_at ASC`
	
	rows, err := Pool.Query(context.Background(), query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()
	
	var messages []models.ChatMessageDB
	for rows.Next() {
		var msg models.ChatMessageDB
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}
	
	return messages, nil
}

// SaveMessage saves a message to the database
func SaveMessage(sessionID int, role string, content string) error {
	query := `INSERT INTO chat_messages (session_id, role, content) VALUES ($1, $2, $3)`
	
	_, err := Pool.Exec(context.Background(), query, sessionID, role, content)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	
	return nil
}

// DeleteSession deletes a chat session (messages will be deleted automatically due to CASCADE)
func DeleteSession(sessionID int) error {
	query := `DELETE FROM chat_sessions WHERE id = $1`
	
	result, err := Pool.Exec(context.Background(), query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("session with id %d not found", sessionID)
	}
	
	return nil
}

