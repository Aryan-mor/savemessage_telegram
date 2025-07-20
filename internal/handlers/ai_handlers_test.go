package handlers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/stretchr/testify/assert"

	"save-message/internal/interfaces"
)

func TestAIHandlers_HandleGeneralTopicMessage_MainFlow(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("Skipping TestAIHandlers_HandleGeneralTopicMessage_MainFlow: not running integration test (set RUN_INTEGRATION=1 to enable)")
	}
	h := NewAIHandlers(nil, nil, nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	if err := h.HandleGeneralTopicMessage(update); err != nil {
		t.Errorf("HandleGeneralTopicMessage returned error: %v", err)
	}
}

func TestAIHandlers_HandleRetryCallback_MainFlow(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("Skipping TestAIHandlers_HandleRetryCallback_MainFlow: not running integration test (set RUN_INTEGRATION=1 to enable)")
	}
	h := NewAIHandlers(nil, nil, nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	msg := &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}
	if err := h.HandleRetryCallback(update, msg); err != nil {
		t.Errorf("HandleRetryCallback returned error: %v", err)
	}
}

func TestAIHandlers_HandleBackToSuggestionsCallback_MainFlow(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("Skipping TestAIHandlers_HandleBackToSuggestionsCallback_MainFlow: not running integration test (set RUN_INTEGRATION=1 to enable)")
	}
	h := NewAIHandlers(nil, nil, nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	msg := &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}
	if err := h.HandleBackToSuggestionsCallback(update, msg); err != nil {
		t.Errorf("HandleBackToSuggestionsCallback returned error: %v", err)
	}
}

// Minimal fakeMessageService for regression test

type fakeMessageServiceForEdit struct {
	interfaces.MessageServiceInterface
	calledDelete *bool
	calledEdit   *bool
}

func (f *fakeMessageServiceForEdit) SendMessage(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
	return &gotgbot.Message{Chat: gotgbot.Chat{Id: chatID}, MessageId: 123, Text: text}, nil
}
func (f *fakeMessageServiceForEdit) EditMessageText(chatID int64, messageID int64, text string, opts *gotgbot.EditMessageTextOpts) (*gotgbot.Message, error) {
	if f.calledEdit != nil {
		*f.calledEdit = true
	}
	return &gotgbot.Message{Chat: gotgbot.Chat{Id: chatID}, MessageId: messageID, Text: text}, nil
}
func (f *fakeMessageServiceForEdit) DeleteMessage(chatID int64, messageID int) error {
	if f.calledDelete != nil {
		*f.calledDelete = true
	}
	return nil
}
func (f *fakeMessageServiceForEdit) CopyMessageToTopic(chatID int64, fromChatID int64, messageID int, messageThreadID int) error {
	return nil
}
func (f *fakeMessageServiceForEdit) CopyMessageToTopicWithResult(chatID int64, fromChatID int64, messageID int, messageThreadID int) (*gotgbot.Message, error) {
	return nil, nil
}
func (f *fakeMessageServiceForEdit) AnswerCallbackQuery(callbackQueryID string, opts *gotgbot.AnswerCallbackQueryOpts) error {
	return nil
}

// Regression test: ensures that after a successful EditMessageText (to 'Choose a folder:'),
// the bot does NOT call DeleteMessage on the same message. This prevents the suggestion message from disappearing.
func TestHandleGeneralTopicMessage_DoesNotDeleteOnEditSuccess(t *testing.T) {
	calledDelete := false
	calledEdit := false
	fakeTopicService := &mockTopicService{}
	fakeAIService := &mockAIService{suggestions: []string{"Food", "Desserts"}}
	ms := &fakeMessageServiceForEdit{calledDelete: &calledDelete, calledEdit: &calledEdit}
	ah := NewAIHandlers(ms, fakeTopicService, fakeAIService, nil)

	msg := &gotgbot.Message{
		Chat:      gotgbot.Chat{Id: 12345},
		MessageId: 111,
		Text:      "Ice cream",
	}
	update := &gotgbot.Update{Message: msg}

	err := ah.HandleGeneralTopicMessage(update)
	assert.NoError(t, err)
	// Wait for goroutine to finish
	time.Sleep(200 * time.Millisecond)
	assert.True(t, calledEdit, "EditMessageText should be called")
	assert.False(t, calledDelete, "DeleteMessage should NOT be called after successful edit")
}

// Minimal mocks for topic/AI service

type mockTopicService struct {
	interfaces.TopicServiceInterface
}

func (m *mockTopicService) GetForumTopics(chatID int64) ([]interfaces.ForumTopic, error) {
	return nil, nil
}

// mockAIService returns fixed suggestions

type mockAIService struct {
	interfaces.AIServiceInterface
	suggestions []string
}

func (m *mockAIService) SuggestFolders(ctx context.Context, message string, existingFolders []string) ([]string, error) {
	return m.suggestions, nil
}
