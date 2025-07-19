package handlers

import (
	"save-message/internal/services"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// MessageHandlers coordinates all message-related interactions
type MessageHandlers struct {
	CommandHandlers *CommandHandlers
	WarningHandlers *WarningHandlers
	AIHandlers      *AIHandlers
	TopicHandlers   *TopicHandlers
}

// NewMessageHandlers creates a new message handlers instance
func NewMessageHandlers(messageService *services.MessageService, topicService *services.TopicService, aiService *services.AIService) *MessageHandlers {
	return &MessageHandlers{
		CommandHandlers: NewCommandHandlers(messageService, topicService),
		WarningHandlers: NewWarningHandlers(messageService),
		AIHandlers:      NewAIHandlers(messageService, topicService, aiService),
		TopicHandlers:   NewTopicHandlers(messageService, topicService),
	}
}

// HandleStartCommand delegates to command handlers
func (mh *MessageHandlers) HandleStartCommand(update *gotgbot.Update) error {
	return mh.CommandHandlers.HandleStartCommand(update)
}

// HandleHelpCommand delegates to command handlers
func (mh *MessageHandlers) HandleHelpCommand(update *gotgbot.Update) error {
	return mh.CommandHandlers.HandleHelpCommand(update)
}

// HandleTopicsCommand delegates to command handlers
func (mh *MessageHandlers) HandleTopicsCommand(update *gotgbot.Update) error {
	return mh.CommandHandlers.HandleTopicsCommand(update)
}

// HandleAddTopicCommand delegates to command handlers
func (mh *MessageHandlers) HandleAddTopicCommand(update *gotgbot.Update) error {
	return mh.CommandHandlers.HandleAddTopicCommand(update)
}

// HandleBotMention delegates to command handlers
func (mh *MessageHandlers) HandleBotMention(update *gotgbot.Update) error {
	return mh.CommandHandlers.HandleBotMention(update)
}

// HandleNonGeneralTopicMessage delegates to warning handlers
func (mh *MessageHandlers) HandleNonGeneralTopicMessage(update *gotgbot.Update) error {
	return mh.WarningHandlers.HandleNonGeneralTopicMessage(update)
}

// HandleGeneralTopicMessage delegates to AI handlers
func (mh *MessageHandlers) HandleGeneralTopicMessage(update *gotgbot.Update) error {
	return mh.AIHandlers.HandleGeneralTopicMessage(update)
}

// IsBotMention checks if the message mentions the bot
func (mh *MessageHandlers) IsBotMention(messageText string) bool {
	return mh.CommandHandlers.IsBotMention(messageText)
}

// IsRecentlyMovedMessage checks if message was recently moved
func (mh *MessageHandlers) IsRecentlyMovedMessage(messageID int64) bool {
	return mh.TopicHandlers.IsRecentlyMovedMessage(messageID)
}

// CleanupMovedMessage cleans up moved message tracking
func (mh *MessageHandlers) CleanupMovedMessage(messageID int64) {
	mh.TopicHandlers.CleanupMovedMessage(messageID)
}

// IsWaitingForTopicName checks if user is waiting for topic name
func (mh *MessageHandlers) IsWaitingForTopicName(userID int64) bool {
	return mh.TopicHandlers.WaitingForTopicName[userID].ChatId != 0
}

// HandleTopicNameEntry delegates to topic handlers
func (mh *MessageHandlers) HandleTopicNameEntry(update *gotgbot.Update) error {
	return mh.TopicHandlers.HandleTopicNameEntry(update)
}
