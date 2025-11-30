package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// KeyManager manages multiple Gemini API keys with automatic rotation on rate limit errors
type KeyManager struct {
	keys        []string
	currentIndex int
	mu          sync.Mutex
	initialized bool
}

var (
	keyManagerInstance *KeyManager
	keyManagerOnce     sync.Once
)

// GetKeyManager returns the singleton instance of KeyManager
func GetKeyManager() *KeyManager {
	keyManagerOnce.Do(func() {
		keyManagerInstance = &KeyManager{
			keys:        []string{},
			currentIndex: 0,
			initialized: false,
		}
		keyManagerInstance.InitKeys()
	})
	return keyManagerInstance
}

// InitKeys initializes the KeyManager by reading GEMINI_API_KEYS from environment
// Falls back to GEMINI_API_KEY if GEMINI_API_KEYS is not set (backward compatibility)
func (km *KeyManager) InitKeys() {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Try to get GEMINI_API_KEYS (comma-separated)
	keysEnv := os.Getenv("GEMINI_API_KEYS")
	if keysEnv != "" {
		// Split by comma and trim whitespace
		keys := strings.Split(keysEnv, ",")
		for _, key := range keys {
			key = strings.TrimSpace(key)
			if key != "" {
				km.keys = append(km.keys, key)
			}
		}
	}

	// Fallback to GEMINI_API_KEY for backward compatibility
	if len(km.keys) == 0 {
		singleKey := os.Getenv("GEMINI_API_KEY")
		if singleKey != "" {
			km.keys = append(km.keys, singleKey)
		}
	}

	if len(km.keys) == 0 {
		log.Println("[KeyManager] Warning: No API keys found in GEMINI_API_KEYS or GEMINI_API_KEY")
	} else {
		log.Printf("[KeyManager] Initialized with %d API key(s)", len(km.keys))
		// Log first few characters of first key for verification (masked)
		if len(km.keys) > 0 && len(km.keys[0]) > 8 {
			maskedKey := km.keys[0][:4] + "..." + km.keys[0][len(km.keys[0])-4:]
			log.Printf("[KeyManager] First key: %s", maskedKey)
		}
		km.initialized = true
	}
}

// IsInitialized returns whether the KeyManager has been initialized with at least one key
func (km *KeyManager) IsInitialized() bool {
	km.mu.Lock()
	defer km.mu.Unlock()
	return km.initialized && len(km.keys) > 0
}

// isRateLimitError checks if an error is a rate limit (429) or quota exceeded error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	
	// Check for common rate limit indicators
	rateLimitIndicators := []string{
		"429",
		"quota exceeded",
		"rate limit",
		"resource_exhausted",
		"too many requests",
		"quota",
		"exceeded",
	}

	for _, indicator := range rateLimitIndicators {
		if strings.Contains(errStr, indicator) {
			return true
		}
	}

	return false
}

// isInvalidKeyError checks if an error is an invalid API key error
func isInvalidKeyError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	
	// Check for invalid API key indicators
	invalidKeyIndicators := []string{
		"api key not valid",
		"api_key_invalid",
		"invalid api key",
		"authentication failed",
		"unauthorized",
		"401",
	}

	for _, indicator := range invalidKeyIndicators {
		if strings.Contains(errStr, indicator) {
			return true
		}
	}

	return false
}

// getNextKey returns the next API key in rotation
func (km *KeyManager) getNextKey() (string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(km.keys) == 0 {
		return "", fmt.Errorf("no API keys available")
	}

	key := km.keys[km.currentIndex]
	km.currentIndex = (km.currentIndex + 1) % len(km.keys)
	return key, nil
}

// getCurrentKey returns the current API key without rotating
func (km *KeyManager) getCurrentKey() (string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(km.keys) == 0 {
		return "", fmt.Errorf("no API keys available")
	}

	return km.keys[km.currentIndex], nil
}

// rotateToNextKey moves to the next key in rotation
func (km *KeyManager) rotateToNextKey() {
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(km.keys) > 0 {
		oldIndex := km.currentIndex
		km.currentIndex = (km.currentIndex + 1) % len(km.keys)
		log.Printf("[KeyManager] Rotated from key index %d to %d", oldIndex, km.currentIndex)
	}
}

// ExecuteWithRetry executes a function with automatic key rotation on rate limit errors
// The callback function receives a genai.Client and should perform the operation
// Returns error if all keys are exhausted or if a non-rate-limit error occurs
func (km *KeyManager) ExecuteWithRetry(ctx context.Context, operation func(client *genai.Client) error) error {
	if !km.initialized || len(km.keys) == 0 {
		return fmt.Errorf("KeyManager not initialized or no API keys available")
	}

	// Try each key at most once
	maxAttempts := len(km.keys)
	attempts := 0

	for attempts < maxAttempts {
		// Get current key
		apiKey, err := km.getCurrentKey()
		if err != nil {
			return fmt.Errorf("failed to get API key: %w", err)
		}

		// Create client with current key
		client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
		if err != nil {
			// If client creation fails, try next key
			log.Printf("[KeyManager] Failed to create client with key index %d: %v", km.currentIndex, err)
			km.rotateToNextKey()
			attempts++
			continue
		}

		// Execute the operation
		opErr := operation(client)
		
		// Close client after operation
		client.Close()

		// If operation succeeded, return success
		if opErr == nil {
			return nil
		}

		// Check if error is invalid API key (don't retry, return immediately)
		if isInvalidKeyError(opErr) {
			log.Printf("[KeyManager] ERROR: Invalid API key at index %d. Please check your GEMINI_API_KEY or GEMINI_API_KEYS", km.currentIndex)
			return fmt.Errorf("invalid API key: %w", opErr)
		}

		// Check if error is rate limit
		if isRateLimitError(opErr) {
			log.Printf("[KeyManager] Rate limit/quota exceeded for key index %d, switching to next key...", km.currentIndex)
			km.rotateToNextKey()
			attempts++
			continue
		}

		// If it's not a rate limit error, return the error immediately
		return opErr
	}

	// All keys exhausted
	return fmt.Errorf("all API keys exhausted (tried %d keys), last error may be rate limit", maxAttempts)
}

// ExecuteWithRetryAndModel executes a function with automatic key rotation and model fallback
// This is a specialized version for operations that need model fallback (like chat generation)
func (km *KeyManager) ExecuteWithRetryAndModel(
	ctx context.Context,
	modelsToTry []string,
	operation func(client *genai.Client, modelName string) error,
) error {
	if !km.initialized || len(km.keys) == 0 {
		return fmt.Errorf("KeyManager not initialized or no API keys available")
	}

	// Try each key at most once
	maxKeyAttempts := len(km.keys)
	keyAttempts := 0

	for keyAttempts < maxKeyAttempts {
		// Get current key
		apiKey, err := km.getCurrentKey()
		if err != nil {
			return fmt.Errorf("failed to get API key: %w", err)
		}

		// Create client with current key
		client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
		if err != nil {
			log.Printf("[KeyManager] Failed to create client with key index %d: %v", km.currentIndex, err)
			km.rotateToNextKey()
			keyAttempts++
			continue
		}

		// Try each model with current key
		var lastErr error
		modelSuccess := false

		for _, modelName := range modelsToTry {
			opErr := operation(client, modelName)
			
			if opErr == nil {
				// Success! Close client and return
				client.Close()
				return nil
			}

			lastErr = opErr

			// If it's an invalid API key error, return immediately (don't retry)
			if isInvalidKeyError(opErr) {
				log.Printf("[KeyManager] ERROR: Invalid API key at index %d with model %s. Please check your API keys", km.currentIndex, modelName)
				client.Close()
				return fmt.Errorf("invalid API key: %w", opErr)
			}

			// If it's a rate limit error, break and try next key
			if isRateLimitError(opErr) {
				log.Printf("[KeyManager] Rate limit for key index %d with model %s, switching to next key...", km.currentIndex, modelName)
				break
			}

			// If it's not rate limit, try next model
			log.Printf("[KeyManager] Model %s failed (non-rate-limit), trying next model...", modelName)
		}

		// Close client before rotating key
		client.Close()

		// If we got a rate limit error, rotate to next key
		if isRateLimitError(lastErr) {
			km.rotateToNextKey()
			keyAttempts++
			continue
		}

		// If no model succeeded and it's not a rate limit, return the last error
		if !modelSuccess {
			return lastErr
		}
	}

	// All keys exhausted
	return fmt.Errorf("all API keys exhausted (tried %d keys)", maxKeyAttempts)
}

// GetClientForStreaming returns a client for streaming operations
// This is a special case because streaming iterators need the client to stay alive
// The caller is responsible for closing the client
func (km *KeyManager) GetClientForStreaming(ctx context.Context) (*genai.Client, error) {
	if !km.initialized || len(km.keys) == 0 {
		return nil, fmt.Errorf("KeyManager not initialized or no API keys available")
	}

	// Get current key
	apiKey, err := km.getCurrentKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Create and return client
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return client, nil
}

// RotateKeyOnError rotates to next key if error is rate limit
// This is useful for streaming where we can't use ExecuteWithRetry
func (km *KeyManager) RotateKeyOnError(err error) bool {
	if isRateLimitError(err) {
		km.rotateToNextKey()
		return true
	}
	return false
}

