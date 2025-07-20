package handlers

import (
	"context"
	"save-message/internal/interfaces"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type MockAIHandlers struct{}

var _ interfaces.AIHandlersInterface = (*MockAIHandlers)(nil)

func (m *MockAIHandlers) HandleGeneralTopicMessage(u *gotgbot.Update) error { return nil }
func (m *MockAIHandlers) HandleRetryCallback(u *gotgbot.Update, msg *gotgbot.Message) error {
	return nil
}
func (m *MockAIHandlers) HandleBackToSuggestionsCallback(u *gotgbot.Update, msg *gotgbot.Message) error {
	return nil
}
func (m *MockAIHandlers) HandleShowExistingFolders(u *gotgbot.Update, msg *gotgbot.Message) error {
	return nil
}

type MockAIService struct{}

var _ interfaces.AIServiceInterface = (*MockAIService)(nil)

func (m *MockAIService) SuggestCategories(messageText string) ([]string, error) { return nil, nil }
func (m *MockAIService) SuggestFolders(ctx context.Context, messageText string, existingFolders []string) ([]string, error) {
	return nil, nil
}
