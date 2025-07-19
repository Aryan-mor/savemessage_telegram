package services

import (
	"context"
	"log"

	"save-message/internal/ai"
)

// AIService handles all AI-related operations
type AIService struct {
	openaiClient *ai.OpenAIClient
}

// NewAIService creates a new AI service
func NewAIService(openaiKey string) *AIService {
	return &AIService{
		openaiClient: ai.NewOpenAIClient(openaiKey),
	}
}

// SuggestFolders suggests folders based on message content
func (as *AIService) SuggestFolders(ctx context.Context, messageText string, existingFolders []string) ([]string, error) {
	log.Printf("[AIService] Suggesting folders for message: %s", messageText)

	suggestions, err := as.openaiClient.SuggestFolders(ctx, messageText, existingFolders)
	if err != nil {
		log.Printf("[AIService] Error getting AI suggestions: %v", err)
		return nil, err
	}

	log.Printf("[AIService] AI suggestions: %v", suggestions)
	return suggestions, nil
}
