package interfaces

import "github.com/PaulSonOfLars/gotgbot/v2"

// TopicHandlersInterface defines the interface for topic-related handlers.
type TopicHandlersInterface interface {
	HandleNewTopicCreationRequest(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleTopicSelectionCallback(update *gotgbot.Update, originalMsg *gotgbot.Message, callbackData string) error
	HandleShowAllTopicsCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleCreateTopicMenuCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleShowAllTopicsMenuCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleTopicNameEntry(update *gotgbot.Update) error
	IsRecentlyMovedMessage(messageID int64) bool
	MarkMessageAsMoved(messageID int64)
	CleanupMovedMessage(messageID int64)
	IsWaitingForTopicName(userID int64) bool
	GetMessageByCallbackData(callbackData string) *gotgbot.Message
}
