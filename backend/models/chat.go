package models

// ChatMessage represents a message in the conversation history
type ChatMessage struct {
	Role    string `json:"role"`    // "user" atau "model"
	Content string `json:"content"`
}

