package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// RerankDocuments uses Cohere's rerank API to reorder documents based on relevance
// It returns the indices of the topN documents (relative to the input slice).
// The function supports API key rotation via COHERE_API_KEYS (comma-separated list).
func RerankDocuments(query string, documents []string, topN int) ([]int, error) {
	if len(documents) == 0 {
		return []int{}, nil
	}

	if topN <= 0 {
		topN = 5
	}

	// Read API keys from environment (support multiple keys separated by comma)
	rawKeys := os.Getenv("COHERE_API_KEYS")
	if rawKeys == "" {
		// Fallback to single key env var if provided
		rawKeys = os.Getenv("COHERE_API_KEY")
	}

	if rawKeys == "" {
		return nil, fmt.Errorf("COHERE_API_KEYS or COHERE_API_KEY is not set")
		}

	parts := strings.Split(rawKeys, ",")
	var keys []string
	for _, p := range parts {
		k := strings.TrimSpace(p)
		if k != "" {
			keys = append(keys, k)
		}
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no valid Cohere API keys found in COHERE_API_KEYS/COHERE_API_KEY")
	}

	// Ensure topN does not exceed number of documents
	if topN > len(documents) {
		topN = len(documents)
	}

	var lastErr error

	for idx, key := range keys {
		log.Printf("[Rerank] Using Cohere key %d/%d\n", idx+1, len(keys))

		// Prepare HTTP request to Cohere Rerank REST API
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		reqBody := map[string]interface{}{
			"model":     "rerank-multilingual-v3.0",
			"query":     query,
			"documents": documents,
			"top_n":     topN,
		}

		payload, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal rerank request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.cohere.com/v1/rerank", bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to create rerank request: %w", err)
		}

		httpReq.Header.Set("Authorization", "Bearer "+key)
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")

		client := &http.Client{
			Timeout: 15 * time.Second,
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			lastErr = err
			errStr := strings.ToLower(err.Error())

			// Detect rate limit / quota errors -> try next key
			if strings.Contains(errStr, "429") || 
			   strings.Contains(errStr, "rate limit") ||
				strings.Contains(errStr, "quota") {
				log.Printf("[Rerank] WARNING: Cohere rate limit/quota hit for key %d, trying next key...\n", idx+1)
				continue
			}

			log.Printf("[Rerank] ERROR calling Cohere rerank: %v\n", err)
			return nil, fmt.Errorf("cohere rerank failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
			// Rate limited or quota exceeded -> try next key
			log.Printf("[Rerank] WARNING: Cohere HTTP %d for key %d (rate limit/quota), trying next key...\n", resp.StatusCode, idx+1)
			lastErr = fmt.Errorf("cohere returned status %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// Other HTTP errors -> fail immediately
			log.Printf("[Rerank] ERROR: Cohere returned status %d\n", resp.StatusCode)
			return nil, fmt.Errorf("cohere rerank HTTP error: %d", resp.StatusCode)
		}

		var parsed struct {
			Results []struct {
				Index          int     `json:"index"`
				RelevanceScore float64 `json:"relevance_score"`
			} `json:"results"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return nil, fmt.Errorf("failed to decode Cohere rerank response: %w", err)
		}

		if len(parsed.Results) == 0 {
			log.Printf("[Rerank] WARNING: Cohere rerank returned no results, falling back to original order\n")
			// Fallback to original order indices
			indices := make([]int, 0, topN)
			for i := 0; i < topN; i++ {
				indices = append(indices, i)
			}
			return indices, nil
		}

		// Extract indices from results
		indices := make([]int, 0, len(parsed.Results))
		for _, r := range parsed.Results {
			i := r.Index
			if i >= 0 && i < len(documents) {
				indices = append(indices, i)
			}
			if len(indices) >= topN {
				break
			}
		}

		if len(indices) == 0 {
			log.Printf("[Rerank] WARNING: Cohere rerank results had no valid indices, falling back to original order\n")
			indices = make([]int, 0, topN)
			for i := 0; i < topN; i++ {
				indices = append(indices, i)
			}
		}

		return indices, nil
	}

	if lastErr != nil {
		log.Printf("[Rerank] ERROR: All Cohere keys failed or were rate-limited: %v\n", lastErr)
		return nil, fmt.Errorf("all Cohere API keys failed or were rate-limited: %w", lastErr)
}

	return nil, fmt.Errorf("unexpected error in RerankDocuments: no keys attempted")
}
