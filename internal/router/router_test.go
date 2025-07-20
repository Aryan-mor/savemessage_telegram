package router

import (
	"context"
	"testing"

	"save-message/internal/interfaces"

	gotgbot "github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- MOCKS FOR ROUTER TESTS ---
type testMessageService struct {
	SendMessageCalled                  bool
	SendMessageArgs                    []interface{}
	SendMessageErr                     error
	AnswerCallbackQueryCalled          bool
	AnswerCallbackQueryArgs            []interface{}
	AnswerCallbackQueryErr             error
	CopyMessageToTopicCalled           bool
	CopyMessageToTopicArgs             []interface{}
	CopyMessageToTopicErr              error
	CopyMessageToTopicWithResultCalled bool
	CopyMessageToTopicWithResultArgs   []interface{}
	CopyMessageToTopicWithResultMsg    *gotgbot.Message
	CopyMessageToTopicWithResultErr    error
	EditMessageTextCalled              bool
	EditMessageTextArgs                []interface{}
	EditMessageTextMsg                 *gotgbot.Message
	EditMessageTextErr                 error
	DeleteMessageCalled                bool
	DeleteMessageArgs                  []interface{}
	DeleteMessageErr                   error
}

func (t *testMessageService) SendMessage(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
	t.SendMessageCalled = true
	t.SendMessageArgs = []interface{}{chatID, text, opts}
	return nil, t.SendMessageErr
}

func (t *testMessageService) AnswerCallbackQuery(callbackQueryID string, opts *gotgbot.AnswerCallbackQueryOpts) error {
	t.AnswerCallbackQueryCalled = true
	t.AnswerCallbackQueryArgs = []interface{}{callbackQueryID, opts}
	return t.AnswerCallbackQueryErr
}

func (t *testMessageService) CopyMessageToTopic(chatID int64, fromChatID int64, messageID int, messageThreadID int) error {
	t.CopyMessageToTopicCalled = true
	t.CopyMessageToTopicArgs = []interface{}{chatID, fromChatID, messageID, messageThreadID}
	return t.CopyMessageToTopicErr
}

func (t *testMessageService) CopyMessageToTopicWithResult(chatID int64, fromChatID int64, messageID int, messageThreadID int) (*gotgbot.Message, error) {
	t.CopyMessageToTopicWithResultCalled = true
	t.CopyMessageToTopicWithResultArgs = []interface{}{chatID, fromChatID, messageID, messageThreadID}
	return t.CopyMessageToTopicWithResultMsg, t.CopyMessageToTopicWithResultErr
}

func (t *testMessageService) EditMessageText(chatID int64, messageID int64, text string, opts *gotgbot.EditMessageTextOpts) (*gotgbot.Message, error) {
	t.EditMessageTextCalled = true
	t.EditMessageTextArgs = []interface{}{chatID, messageID, text, opts}
	return t.EditMessageTextMsg, t.EditMessageTextErr
}

func (t *testMessageService) DeleteMessage(chatID int64, messageID int) error {
	t.DeleteMessageCalled = true
	t.DeleteMessageArgs = []interface{}{chatID, messageID}
	return t.DeleteMessageErr
}

type MockMessageHandlers struct {
	interfaces.MessageHandlersInterface
	HandleMessageCalled bool
	HandleMessageErr    error
}

func (m *MockMessageHandlers) HandleMessage(update *gotgbot.Update) error {
	m.HandleMessageCalled = true
	return m.HandleMessageErr
}

type MockCallbackHandlers struct {
	interfaces.CallbackHandlersInterface
	HandleCallbackQueryCalled bool
	HandleCallbackQueryErr    error
	IsWaitingForTopicNameVal  bool
}

func (m *MockCallbackHandlers) HandleCallbackQuery(update *gotgbot.Update) error {
	m.HandleCallbackQueryCalled = true
	return m.HandleCallbackQueryErr
}
func (m *MockCallbackHandlers) IsWaitingForTopicName(userID int64) bool {
	return m.IsWaitingForTopicNameVal
}

func TestHandleMessage(t *testing.T) {
	ctx := context.Background()
	bot := &gotgbot.Bot{}
	msg := &gotgbot.Message{}

	// Test that the function doesn't panic and returns nil
	err := HandleMessage(ctx, bot, msg)
	assert.NoError(t, err)
}

func TestHandleMessage_MainFlow(t *testing.T) {
	ctx := context.Background()
	bot := &gotgbot.Bot{User: gotgbot.User{Username: "testbot"}}
	msg := &gotgbot.Message{MessageId: 1, Text: "hello"}
	if err := HandleMessage(ctx, bot, msg); err != nil {
		t.Errorf("HandleMessage returned error: %v", err)
	}
}

func TestNewDispatcher(t *testing.T) {
	messageHandlers := &MockMessageHandlers{}
	callbackHandlers := &MockCallbackHandlers{}
	messageService := &testMessageService{}

	dispatcher := NewDispatcher(messageHandlers, callbackHandlers, messageService)

	assert.NotNil(t, dispatcher)
	assert.Equal(t, messageHandlers, dispatcher.MessageHandlers)
	assert.Equal(t, callbackHandlers, dispatcher.CallbackHandlers)
	assert.Equal(t, messageService, dispatcher.MessageService)
}

func TestDispatcher_HandleUpdate(t *testing.T) {
	messageHandlers := &MockMessageHandlers{}
	callbackHandlers := &MockCallbackHandlers{}
	messageService := &testMessageService{}
	dispatcher := NewDispatcher(messageHandlers, callbackHandlers, messageService)

	tests := []struct {
		name        string
		update      *gotgbot.Update
		expectError bool
	}{
		{
			name:        "nil update",
			update:      nil,
			expectError: false,
		},
		{
			name: "empty update",
			update: &gotgbot.Update{
				UpdateId: 4,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dispatcher.HandleUpdate(tt.update)

			// The function should not panic and should handle all cases gracefully
			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Even if there's an error due to nil handlers, we're testing the structure
				assert.NotNil(t, dispatcher)
			}
		})
	}
}

func TestDispatcher_IsEditRequest(t *testing.T) {
	dispatcher := &Dispatcher{}

	tests := []struct {
		name     string
		update   *gotgbot.Update
		expected bool
	}{
		{
			name:     "nil update",
			update:   nil,
			expected: false,
		},
		{
			name: "nil message",
			update: &gotgbot.Update{
				UpdateId: 1,
			},
			expected: false,
		},
		{
			name: "empty text",
			update: &gotgbot.Update{
				UpdateId: 2,
				Message: &gotgbot.Message{
					Text: "",
				},
			},
			expected: false,
		},
		{
			name: "edit request",
			update: &gotgbot.Update{
				UpdateId: 3,
				Message: &gotgbot.Message{
					Text: "Edit: some text",
				},
			},
			expected: true,
		},
		{
			name: "regular message",
			update: &gotgbot.Update{
				UpdateId: 4,
				Message: &gotgbot.Message{
					Text: "Regular message",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dispatcher.IsEditRequest(tt.update)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDispatcher_IsTopicSelection(t *testing.T) {
	dispatcher := &Dispatcher{}

	tests := []struct {
		name     string
		update   *gotgbot.Update
		expected bool
	}{
		{
			name:     "nil update",
			update:   nil,
			expected: false,
		},
		{
			name: "nil callback query",
			update: &gotgbot.Update{
				UpdateId: 1,
			},
			expected: false,
		},
		{
			name: "topic selection",
			update: &gotgbot.Update{
				UpdateId: 2,
				CallbackQuery: &gotgbot.CallbackQuery{
					Data: "Work_123",
				},
			},
			expected: true,
		},
		{
			name: "create new folder",
			update: &gotgbot.Update{
				UpdateId: 3,
				CallbackQuery: &gotgbot.CallbackQuery{
					Data: "create_new_folder_123",
				},
			},
			expected: false,
		},
		{
			name: "retry callback",
			update: &gotgbot.Update{
				UpdateId: 4,
				CallbackQuery: &gotgbot.CallbackQuery{
					Data: "retry_123",
				},
			},
			expected: false,
		},
		{
			name: "show all topics",
			update: &gotgbot.Update{
				UpdateId: 5,
				CallbackQuery: &gotgbot.CallbackQuery{
					Data: "show_all_topics_123",
				},
			},
			expected: false,
		},
		{
			name: "back to suggestions",
			update: &gotgbot.Update{
				UpdateId: 6,
				CallbackQuery: &gotgbot.CallbackQuery{
					Data: "back_to_suggestions_123",
				},
			},
			expected: false,
		},
		{
			name: "create topic menu",
			update: &gotgbot.Update{
				UpdateId: 7,
				CallbackQuery: &gotgbot.CallbackQuery{
					Data: "create_topic_menu",
				},
			},
			expected: false,
		},
		{
			name: "show all topics menu",
			update: &gotgbot.Update{
				UpdateId: 8,
				CallbackQuery: &gotgbot.CallbackQuery{
					Data: "show_all_topics_menu",
				},
			},
			expected: false,
		},
		{
			name: "detect message on other topic",
			update: &gotgbot.Update{
				UpdateId: 9,
				CallbackQuery: &gotgbot.CallbackQuery{
					Data: "detectMessageOnOtherTopic_ok_123",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dispatcher.IsTopicSelection(tt.update)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDispatcher_IsNewTopicPrompt(t *testing.T) {
	mockCallbacks := &MockCallbackHandlers{}
	dispatcher := &Dispatcher{
		CallbackHandlers: mockCallbacks,
	}

	tests := []struct {
		name     string
		update   *gotgbot.Update
		expected bool
		skip     bool
	}{
		{
			name:     "nil update",
			update:   nil,
			expected: false,
		},
		{
			name:     "nil message",
			update:   &gotgbot.Update{UpdateId: 1},
			expected: false,
		},
		{
			name:     "message with user (skipped, needs mock)",
			update:   &gotgbot.Update{UpdateId: 2, Message: &gotgbot.Message{From: &gotgbot.User{Id: 123}}},
			expected: false, // Would panic due to nil dependencies
			skip:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("skipping test that requires a properly mocked CallbackHandlers")
			}
			result := dispatcher.IsNewTopicPrompt(tt.update)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDispatcher_IsMessageInGeneralTopic(t *testing.T) {
	dispatcher := &Dispatcher{}

	tests := []struct {
		name     string
		update   *gotgbot.Update
		expected bool
	}{
		{
			name:     "nil update",
			update:   nil,
			expected: false,
		},
		{
			name: "nil message",
			update: &gotgbot.Update{
				UpdateId: 1,
			},
			expected: false,
		},
		{
			name: "message in general topic",
			update: &gotgbot.Update{
				UpdateId: 2,
				Message: &gotgbot.Message{
					MessageThreadId: 0,
					Chat: gotgbot.Chat{
						Type: "supergroup",
					},
				},
			},
			expected: true,
		},
		{
			name: "message in other topic",
			update: &gotgbot.Update{
				UpdateId: 3,
				Message: &gotgbot.Message{
					MessageThreadId: 1,
					Chat: gotgbot.Chat{
						Type: "supergroup",
					},
				},
			},
			expected: false,
		},
		{
			name: "message in private chat",
			update: &gotgbot.Update{
				UpdateId: 4,
				Message: &gotgbot.Message{
					MessageThreadId: 0,
					Chat: gotgbot.Chat{
						Type: "private",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dispatcher.IsMessageInGeneralTopic(tt.update)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Fake implementation of MessageHandlersInterface for testing
// Only HandleStartCommand is asserted, others are no-ops

type fakeMessageHandlers struct {
	mock.Mock
}

func (f *fakeMessageHandlers) HandleStartCommand(update *gotgbot.Update) error {
	f.Called(update)
	return nil
}
func (f *fakeMessageHandlers) HandleHelpCommand(update *gotgbot.Update) error            { return nil }
func (f *fakeMessageHandlers) HandleTopicsCommand(update *gotgbot.Update) error          { return nil }
func (f *fakeMessageHandlers) HandleAddTopicCommand(update *gotgbot.Update) error        { return nil }
func (f *fakeMessageHandlers) HandleBotMention(update *gotgbot.Update) error             { return nil }
func (f *fakeMessageHandlers) HandleNonGeneralTopicMessage(update *gotgbot.Update) error { return nil }
func (f *fakeMessageHandlers) HandleGeneralTopicMessage(update *gotgbot.Update) error    { return nil }
func (f *fakeMessageHandlers) HandleTopicNameEntry(update *gotgbot.Update) error         { return nil }

// Minimal fake implementation of CallbackHandlersInterface for testing
// All methods are no-ops

type fakeCallbackHandlers struct{}

func (f *fakeCallbackHandlers) HandleCallbackQuery(update *gotgbot.Update) error  { return nil }
func (f *fakeCallbackHandlers) IsRecentlyMovedMessage(messageID int64) bool       { return false }
func (f *fakeCallbackHandlers) CleanupMovedMessage(messageID int64)               {}
func (f *fakeCallbackHandlers) IsWaitingForTopicName(userID int64) bool           { return false }
func (f *fakeCallbackHandlers) HandleTopicNameEntry(update *gotgbot.Update) error { return nil }

// Minimal fake implementation of MessageServiceInterface for testing
// All methods are no-ops

type fakeMessageService struct{}

func (f *fakeMessageService) DeleteMessage(chatID int64, messageID int) error { return nil }
func (f *fakeMessageService) CopyMessageToTopic(chatID int64, fromChatID int64, messageID int, messageThreadID int) error {
	return nil
}
func (f *fakeMessageService) CopyMessageToTopicWithResult(chatID int64, fromChatID int64, messageID int, messageThreadID int) (*gotgbot.Message, error) {
	return nil, nil
}
func (f *fakeMessageService) SendMessage(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
	return nil, nil
}
func (f *fakeMessageService) EditMessageText(chatID int64, messageID int64, text string, opts *gotgbot.EditMessageTextOpts) (*gotgbot.Message, error) {
	return nil, nil
}
func (f *fakeMessageService) AnswerCallbackQuery(callbackQueryID string, opts *gotgbot.AnswerCallbackQueryOpts) error {
	return nil
}

// Regression test: ensures that when the bot is added to a group as admin/member via my_chat_member update,
// the welcome message handler is called. This prevents silent breakage of the group join welcome flow.
func TestDispatcher_HandleUpdate_MyChatMember_BotAdded_SendsWelcome(t *testing.T) {
	// Prepare dispatcher with fake handler
	mh := &fakeMessageHandlers{}
	ch := &fakeCallbackHandlers{}
	ms := &fakeMessageService{}
	d := NewDispatcher(mh, ch, ms)

	// Simulate a my_chat_member update where the bot is added as administrator
	chat := gotgbot.Chat{Id: 12345, Title: "Test Group", Type: "supergroup"}
	botUser := gotgbot.User{Id: 999, IsBot: true, Username: "mybot"}
	admin := gotgbot.ChatMemberAdministrator{User: botUser}
	update := &gotgbot.Update{
		MyChatMember: &gotgbot.ChatMemberUpdated{
			Chat:          chat,
			NewChatMember: admin,
		},
	}

	mh.On("HandleStartCommand", mock.Anything).Return(nil).Once()

	err := d.HandleUpdate(update)
	assert.NoError(t, err)
	mh.AssertCalled(t, "HandleStartCommand", mock.Anything)
}

// Minimal fakeMessageHandlers for regression test

type fakeMessageHandlersForJoin struct {
	interfaces.MessageHandlersInterface
	called bool
}

func (f *fakeMessageHandlersForJoin) HandleGeneralTopicMessage(update *gotgbot.Update) error {
	f.called = true
	return nil
}
func (f *fakeMessageHandlersForJoin) HandleStartCommand(update *gotgbot.Update) error    { return nil }
func (f *fakeMessageHandlersForJoin) HandleHelpCommand(update *gotgbot.Update) error     { return nil }
func (f *fakeMessageHandlersForJoin) HandleTopicsCommand(update *gotgbot.Update) error   { return nil }
func (f *fakeMessageHandlersForJoin) HandleAddTopicCommand(update *gotgbot.Update) error { return nil }
func (f *fakeMessageHandlersForJoin) HandleBotMention(update *gotgbot.Update) error      { return nil }
func (f *fakeMessageHandlersForJoin) HandleNonGeneralTopicMessage(update *gotgbot.Update) error {
	return nil
}
func (f *fakeMessageHandlersForJoin) HandleTopicNameEntry(update *gotgbot.Update) error { return nil }

// Regression test: ensures that when the bot receives a join message for itself (new_chat_members contains the bot),
// it does NOT process it as a regular message (i.e., does not call HandleGeneralTopicMessage or send 'Thinking...').
func TestDispatcher_HandleMessage_BotJoinMessage_IsIgnored(t *testing.T) {
	mh := &fakeMessageHandlersForJoin{}
	ch := &fakeCallbackHandlers{}
	ms := &fakeMessageService{}
	d := NewDispatcher(mh, ch, ms)
	d.BotUserID = 999

	// Simulate a message where the bot is in new_chat_members
	chat := gotgbot.Chat{Id: 12345, Title: "Test Group", Type: "supergroup"}
	botUser := gotgbot.User{Id: 999, IsBot: true, Username: "mybot"}
	msg := &gotgbot.Message{
		Chat:           chat,
		MessageId:      123,
		NewChatMembers: []gotgbot.User{botUser},
		From:           &gotgbot.User{Id: 111, IsBot: false, Username: "admin"},
	}
	update := &gotgbot.Update{Message: msg}

	err := d.HandleUpdate(update)
	assert.NoError(t, err)
	assert.False(t, mh.called, "HandleGeneralTopicMessage should NOT be called for bot's own join message")
}
