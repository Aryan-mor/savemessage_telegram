package interfaces

import "github.com/PaulSonOfLars/gotgbot/v2"

// AIHandlersInterface defines the interface for AI-related handlers.
type AIHandlersInterface interface {
	HandleGeneralTopicMessage(update *gotgbot.Update) error
	HandleRetryCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleBackToSuggestionsCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleShowExistingFolders(update *gotgbot.Update, originalMsg *gotgbot.Message) error
}
