package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIClient handles requests to the OpenAI API
type OpenAIClient struct {
	APIKey     string
	HTTPClient *http.Client
}

// NewOpenAIClient creates a new OpenAIClient
func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// SuggestFolders sends a message to OpenAI and returns suggested folder names
func (c *OpenAIClient) SuggestFolders(ctx context.Context, message string, existingFolders []string) ([]string, error) {
	prompt := buildPrompt(message, existingFolders)
	requestBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "system", "content": "You are an assistant that helps organize messages into folders (topics) for a Telegram user."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 64,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("OpenAI response decode error: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("No choices returned from OpenAI")
	}

	// Parse the folder suggestions from the response (expecting a comma-separated or list format)
	// Example: "Folders: Work, Personal, Shopping"
	// You may want to improve this parsing based on your prompt
	folders := parseFolders(result.Choices[0].Message.Content)
	return folders, nil
}

// buildPrompt creates the prompt for OpenAI
func buildPrompt(message string, existingFolders []string) string {
	prompt := "Given the following message: '" + message + "'\n"

	if len(existingFolders) > 0 {
		prompt += "Existing topics: " + fmt.Sprintf("%v", existingFolders) + "\n"
		prompt += "IMPORTANT RULES:\n"
		prompt += "1. ALWAYS check if any existing topics are relevant to this message FIRST\n"
		prompt += "2. If an existing topic is relevant, include it in your suggestions\n"
		prompt += "3. Only suggest NEW topics if NO existing topics are relevant\n"
		prompt += "4. Never suggest 'General' as it's the default topic\n"
		prompt += "5. Prioritize existing topics over new ones when both are relevant\n"
		prompt += "Suggest 2-3 relevant topics for this message. Return only a comma-separated list of topic names."
	} else {
		prompt += "Suggest 2-3 relevant topic names for this message. Never suggest 'General' as it's the default topic. Return only a comma-separated list of topic names."
	}

	return prompt
}

// parseFolders parses a comma-separated list of folder names from the OpenAI response
func parseFolders(response string) []string {
	var folders []string
	for _, f := range bytes.Split([]byte(response), []byte{','}) {
		name := string(bytes.TrimSpace(f))
		if name != "" {
			folders = append(folders, name)
		}
	}
	return folders
}
