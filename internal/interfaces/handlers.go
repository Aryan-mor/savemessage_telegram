package interfaces

import "github.com/PaulSonOfLars/gotgbot/v2"

type MessageHandlersInterface interface {
	HandleStartCommand(update *gotgbot.Update) error
	HandleHelpCommand(update *gotgbot.Update) error
	HandleTopicsCommand(update *gotgbot.Update) error
	HandleAddTopicCommand(update *gotgbot.Update) error
	HandleBotMention(update *gotgbot.Update) error
	HandleNonGeneralTopicMessage(update *gotgbot.Update) error
	HandleGeneralTopicMessage(update *gotgbot.Update) error
}

type CallbackHandlersInterface interface {
	HandleCallbackQuery(update *gotgbot.Update) error
	IsRecentlyMovedMessage(messageID int64) bool
	CleanupMovedMessage(messageID int64)
	IsWaitingForTopicName(userID int64) bool
	HandleTopicNameEntry(update *gotgbot.Update) error
}
