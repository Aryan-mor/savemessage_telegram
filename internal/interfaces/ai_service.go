package interfaces

import "context"

// AIServiceInterface abstracts AI operations
// SuggestFolders returns a list of folder suggestions for a message
//go:generate mockgen -destination=ai_service_mock.go -package=interfaces . AIServiceInterface

// SuggestFolders returns a list of folder suggestions for a message
// messageText: the message to analyze
// existingFolders: the folders already present
// Returns: a list of suggested folders, or error

type AIServiceInterface interface {
	SuggestFolders(ctx context.Context, messageText string, existingFolders []string) ([]string, error)
}
