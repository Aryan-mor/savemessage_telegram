package services

import (
	"context"
	"net/http"

	"save-message/internal/ai"
	"save-message/internal/interfaces"
	"save-message/internal/logutils"
)

// AIService handles AI-powered folder suggestions
type AIService struct {
	openAIClient ai.OpenAIClientInterface
}

// NewAIService creates a new AI service
func NewAIService(openaiKey string, client interfaces.HTTPClient) *AIService {
	if client == nil {
		client = &http.Client{}
	}
	return &AIService{
		openAIClient: ai.NewOpenAIClient(openaiKey, client),
	}
}

// SuggestFolders suggests folders based on message content
func (as *AIService) SuggestFolders(ctx context.Context, messageText string, existingFolders []string) ([]string, error) {
	logutils.Info("SuggestFolders", "messageText", messageText, "existingFolders", existingFolders)

	suggestions, err := as.openAIClient.SuggestFolders(ctx, messageText, existingFolders)
	if err != nil {
		logutils.Error("SuggestFolders: OpenAIClientError", err, "messageText", messageText)
		return nil, err
	}

	logutils.Success("SuggestFolders", "suggestions_count", len(suggestions))
	return suggestions, nil
}
