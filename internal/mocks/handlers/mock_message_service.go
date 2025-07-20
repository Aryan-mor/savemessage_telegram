package handlers

import (
	"errors"
	"save-message/internal/interfaces"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type MockMessageService struct {
	DeleteMessageCalled   bool
	SendMessageCalled     bool
	SendMessageShouldFail bool
}

var _ interfaces.MessageServiceInterface = (*MockMessageService)(nil)

func (m *MockMessageService) DeleteMessage(chatID int64, messageID int) error {
	m.DeleteMessageCalled = true
	return nil
}
func (m *MockMessageService) CopyMessageToTopic(chatID int64, fromChatID int64, messageID int, messageThreadID int) error {
	return nil
}
func (m *MockMessageService) CopyMessageToTopicWithResult(chatID int64, fromChatID int64, messageID int, messageThreadID int) (*gotgbot.Message, error) {
	return nil, nil
}
func (m *MockMessageService) SendMessage(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
	m.SendMessageCalled = true
	if m.SendMessageShouldFail {
		return nil, errors.New("send failed")
	}
	return &gotgbot.Message{MessageId: 999, Chat: gotgbot.Chat{Id: chatID}}, nil
}
func (m *MockMessageService) EditMessageText(chatID int64, messageID int64, text string, opts *gotgbot.EditMessageTextOpts) (*gotgbot.Message, error) {
	return nil, nil
}
func (m *MockMessageService) AnswerCallbackQuery(callbackQueryID string, opts *gotgbot.AnswerCallbackQueryOpts) error {
	return nil
}
