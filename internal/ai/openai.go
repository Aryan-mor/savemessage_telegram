package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"save-message/internal/interfaces"
	"save-message/internal/logutils"
)

// OpenAIClient implements OpenAI API calls
type OpenAIClient struct {
	apiKey     string
	httpClient interfaces.HTTPClient
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string, client interfaces.HTTPClient) *OpenAIClient {
	return &OpenAIClient{
		apiKey:     apiKey,
		httpClient: client,
	}
}

// SuggestFolders sends a message to OpenAI and returns suggested folder names
func (c *OpenAIClient) SuggestFolders(ctx context.Context, message string, existingFolders []string) ([]string, error) {
	logutils.Info("SuggestFolders: entry")
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

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		logutils.Error("SuggestFolders: error creating request", err)
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logutils.Error("SuggestFolders: error sending request to OpenAI", err)
		return nil, fmt.Errorf("error sending request to OpenAI: %w", err)
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
		logutils.Error("SuggestFolders: OpenAI response decode error", err)
		return nil, fmt.Errorf("OpenAI response decode error: %w", err)
	}
	if len(result.Choices) == 0 {
		logutils.Error("SuggestFolders: No choices returned from OpenAI", nil)
		return nil, fmt.Errorf("No choices returned from OpenAI")
	}

	folders := parseFolders(result.Choices[0].Message.Content)
	logutils.Success("SuggestFolders: exit", "suggestion_count", len(folders))
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
