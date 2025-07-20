package interfaces

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type MessageServiceInterface interface {
	DeleteMessage(chatID int64, messageID int) error
	CopyMessageToTopic(chatID int64, fromChatID int64, messageID int, messageThreadID int) error
	CopyMessageToTopicWithResult(chatID int64, fromChatID int64, messageID int, messageThreadID int) (*gotgbot.Message, error)
	SendMessage(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error)
	EditMessageText(chatID int64, messageID int64, text string, opts *gotgbot.EditMessageTextOpts) (*gotgbot.Message, error)
	AnswerCallbackQuery(callbackQueryID string, opts *gotgbot.AnswerCallbackQueryOpts) error
}
