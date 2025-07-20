package ai

import "context"

// OpenAIClientInterface defines the interface for OpenAI client operations
type OpenAIClientInterface interface {
	SuggestFolders(ctx context.Context, message string, existingFolders []string) ([]string, error)
}
