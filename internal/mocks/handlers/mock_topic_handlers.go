package handlers

import (
	"save-message/internal/interfaces"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type MockTopicHandlers struct{}

var _ interfaces.TopicHandlersInterface = (*MockTopicHandlers)(nil)

func (m *MockTopicHandlers) HandleNewTopicCreationRequest(u *gotgbot.Update, msg *gotgbot.Message) error {
	return nil
}
func (m *MockTopicHandlers) HandleTopicSelectionCallback(u *gotgbot.Update, msg *gotgbot.Message, cb string) error {
	return nil
}
func (m *MockTopicHandlers) HandleShowAllTopicsCallback(u *gotgbot.Update, msg *gotgbot.Message) error {
	return nil
}
func (m *MockTopicHandlers) HandleCreateTopicMenuCallback(u *gotgbot.Update, msg *gotgbot.Message) error {
	return nil
}
func (m *MockTopicHandlers) HandleShowAllTopicsMenuCallback(u *gotgbot.Update, msg *gotgbot.Message) error {
	return nil
}
func (m *MockTopicHandlers) HandleTopicNameEntry(u *gotgbot.Update) error        { return nil }
func (m *MockTopicHandlers) IsRecentlyMovedMessage(messageID int64) bool         { return false }
func (m *MockTopicHandlers) MarkMessageAsMoved(messageID int64)                  {}
func (m *MockTopicHandlers) CleanupMovedMessage(messageID int64)                 {}
func (m *MockTopicHandlers) IsWaitingForTopicName(userID int64) bool             { return false }
func (m *MockTopicHandlers) GetMessageByCallbackData(cb string) *gotgbot.Message { return nil }

type MockTopicService struct{}

var _ interfaces.TopicServiceInterface = (*MockTopicService)(nil)

func (m *MockTopicService) GetForumTopics(chatID int64) ([]interfaces.ForumTopic, error) {
	return nil, nil
}
func (m *MockTopicService) CreateForumTopic(chatID int64, name string) (int64, error) { return 0, nil }
func (m *MockTopicService) TopicExists(chatID int64, topicName string) (bool, error) {
	return false, nil
}
func (m *MockTopicService) FindTopicByName(chatID int64, topicName string) (int64, error) {
	return 0, nil
}
func (m *MockTopicService) AddTopic(chatID int64, name string, messageThreadID int64, createdBy int64) error {
	return nil
}
func (m *MockTopicService) GetTopicsByChat(chatID int64) ([]interfaces.ForumTopic, error) {
	return nil, nil
}
