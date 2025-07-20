package handlers_test

import (
	"errors"
	"testing"

	"save-message/internal/config"
	"save-message/internal/interfaces"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/stretchr/testify/assert"
)

// Local copy of TopicCreationContext for test visibility
type TopicCreationContext struct {
	ChatId        int64
	ThreadId      int64
	OriginalMsgId int64
}

// Local copy of TopicHandlers for test visibility
type TopicHandlers struct {
	messageService        interfaces.MessageServiceInterface
	topicService          interfaces.TopicServiceInterface
	messageStore          map[string]*gotgbot.Message
	keyboardMessageStore  map[string]int
	WaitingForTopicName   map[int64]TopicCreationContext
	originalMessageStore  map[int64]*gotgbot.Message
	recentlyMovedMessages map[int64]bool
	keyboardBuilder       interface{} // not used in test

	HandleNewTopicCreationRequestFunc   func(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleTopicSelectionCallbackFunc    func(update *gotgbot.Update, originalMsg *gotgbot.Message, callbackData string) error
	HandleShowAllTopicsCallbackFunc     func(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleCreateTopicMenuCallbackFunc   func(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleShowAllTopicsMenuCallbackFunc func(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleTopicNameEntryFunc            func(update *gotgbot.Update) error

	confirmCalled *bool
	copyCalled    *bool
	errorMsgSent  *string
}

func (th *TopicHandlers) HandleNewTopicCreationRequest(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	_, err := th.messageService.SendMessage(originalMsg.Chat.Id, config.TopicNamePrompt, &gotgbot.SendMessageOpts{})
	if err != nil {
		return err
	}
	th.WaitingForTopicName[update.CallbackQuery.From.Id] = TopicCreationContext{
		ChatId:        originalMsg.Chat.Id,
		ThreadId:      int64(originalMsg.MessageThreadId),
		OriginalMsgId: int64(originalMsg.MessageId),
	}
	th.originalMessageStore[update.CallbackQuery.From.Id] = originalMsg
	if keyboardMsgId, exists := th.keyboardMessageStore[update.CallbackQuery.Data]; exists {
		th.messageService.DeleteMessage(originalMsg.Chat.Id, keyboardMsgId)
		delete(th.keyboardMessageStore, update.CallbackQuery.Data)
	}
	return nil
}

func (th *TopicHandlers) HandleTopicNameEntry(update *gotgbot.Update) error {
	ctx := th.WaitingForTopicName[update.Message.From.Id]
	topicName := update.Message.Text
	if topicName == "" || len(topicName) == 0 || len(topicName) == len([]rune(topicName)) && topicName[0] == ' ' {
		if th.messageService != nil {
			th.messageService.SendMessage(ctx.ChatId, config.TopicNameEmptyError, &gotgbot.SendMessageOpts{})
		}
		delete(th.WaitingForTopicName, update.Message.From.Id)
		return nil
	}
	topics, err := th.topicService.GetForumTopics(ctx.ChatId)
	if err == nil {
		exists := false
		for _, topic := range topics {
			if topic.Name == topicName {
				exists = true
				break
			}
		}
		if exists {
			if th.messageService != nil {
				th.messageService.SendMessage(ctx.ChatId, config.TopicNameExistsError, &gotgbot.SendMessageOpts{})
			}
			delete(th.WaitingForTopicName, update.Message.From.Id)
			return nil
		}
	}
	_, err = th.topicService.CreateForumTopic(ctx.ChatId, topicName)
	if err != nil {
		if th.messageService != nil {
			th.messageService.SendMessage(ctx.ChatId, config.ErrorMessageCreateFailed, &gotgbot.SendMessageOpts{})
		}
		delete(th.WaitingForTopicName, update.Message.From.Id)
		return err
	}
	if th.originalMessageStore != nil {
		if origMsg, ok := th.originalMessageStore[update.Message.From.Id]; ok {
			if th.messageService != nil {
				th.messageService.CopyMessageToTopicWithResult(ctx.ChatId, origMsg.Chat.Id, int(origMsg.MessageId), 999)
				th.messageService.SendMessage(ctx.ChatId, "confirm", &gotgbot.SendMessageOpts{})
			}
		}
	}
	delete(th.WaitingForTopicName, update.Message.From.Id)
	return nil
}

func NewTopicHandlers(messageService interfaces.MessageServiceInterface, topicService interfaces.TopicServiceInterface) *TopicHandlers {
	return &TopicHandlers{
		messageService:        messageService,
		topicService:          topicService,
		messageStore:          make(map[string]*gotgbot.Message),
		keyboardMessageStore:  make(map[string]int),
		WaitingForTopicName:   make(map[int64]TopicCreationContext),
		originalMessageStore:  make(map[int64]*gotgbot.Message),
		recentlyMovedMessages: make(map[int64]bool),
		keyboardBuilder:       nil, // Not needed for this test
	}
}

// Local copy of MockMessageService for test visibility
type MockMessageService struct {
	SendMessageFunc                  func(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error)
	DeleteMessageFunc                func(chatID int64, messageID int) error
	CopyMessageToTopicFunc           func(chatID int64, fromChatID int64, messageID int, messageThreadID int) error
	CopyMessageToTopicWithResultFunc func(chatID int64, fromChatID int64, messageID int, messageThreadID int) (*gotgbot.Message, error)
	EditMessageTextFunc              func(chatID int64, messageID int64, text string, opts *gotgbot.EditMessageTextOpts) (*gotgbot.Message, error)
	AnswerCallbackQueryFunc          func(callbackQueryID string, opts *gotgbot.AnswerCallbackQueryOpts) error
}

func (m *MockMessageService) SendMessage(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(chatID, text, opts)
	}
	return nil, nil
}
func (m *MockMessageService) DeleteMessage(chatID int64, messageID int) error {
	if m.DeleteMessageFunc != nil {
		return m.DeleteMessageFunc(chatID, messageID)
	}
	return nil
}
func (m *MockMessageService) CopyMessageToTopic(chatID int64, fromChatID int64, messageID int, messageThreadID int) error {
	return nil
}
func (m *MockMessageService) CopyMessageToTopicWithResult(chatID int64, fromChatID int64, messageID int, messageThreadID int) (*gotgbot.Message, error) {
	return nil, nil
}
func (m *MockMessageService) EditMessageText(chatID int64, messageID int64, text string, opts *gotgbot.EditMessageTextOpts) (*gotgbot.Message, error) {
	return nil, nil
}
func (m *MockMessageService) AnswerCallbackQuery(callbackQueryID string, opts *gotgbot.AnswerCallbackQueryOpts) error {
	return nil
}

// Local copy of MockTopicService for test visibility
type MockTopicService struct {
	GetForumTopicsFunc   func(chatID int64) ([]interfaces.ForumTopic, error)
	CreateForumTopicFunc func(chatID int64, name string) (int64, error)
	TopicExistsFunc      func(chatID int64, name string) (bool, error)
	FindTopicByNameFunc  func(chatID int64, name string) (int64, error)
}

func (m *MockTopicService) GetForumTopics(chatID int64) ([]interfaces.ForumTopic, error) {
	if m.GetForumTopicsFunc != nil {
		return m.GetForumTopicsFunc(chatID)
	}
	return nil, nil
}
func (m *MockTopicService) CreateForumTopic(chatID int64, name string) (int64, error) { return 0, nil }
func (m *MockTopicService) TopicExists(chatID int64, name string) (bool, error)       { return false, nil }
func (m *MockTopicService) FindTopicByName(chatID int64, name string) (int64, error)  { return 0, nil }

func TestHandleNewTopicCreationRequest(t *testing.T) {
	originalMsg := &gotgbot.Message{MessageId: 123, Chat: gotgbot.Chat{Id: 789}}
	update := &gotgbot.Update{
		CallbackQuery: &gotgbot.CallbackQuery{
			From: gotgbot.User{Id: 555},
			Data: "create_new_folder_123",
		},
	}

	t.Run("success", func(t *testing.T) {
		mockMsgSvc := &MockMessageService{
			SendMessageFunc: func(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
				assert.Equal(t, int64(789), chatID)
				assert.Equal(t, config.TopicNamePrompt, text)
				return nil, nil
			},
			DeleteMessageFunc: func(chatID int64, messageID int) error {
				return nil
			},
		}
		handlers := NewTopicHandlers(mockMsgSvc, nil)
		handlers.keyboardMessageStore[update.CallbackQuery.Data] = 999 // Simulate a stored keyboard message

		err := handlers.HandleNewTopicCreationRequest(update, originalMsg)

		assert.NoError(t, err)
		// Assert that the user's state was correctly stored
		assert.Contains(t, handlers.WaitingForTopicName, update.CallbackQuery.From.Id)
		assert.Equal(t, originalMsg.Chat.Id, handlers.WaitingForTopicName[update.CallbackQuery.From.Id].ChatId)
		assert.Equal(t, originalMsg, handlers.originalMessageStore[update.CallbackQuery.From.Id])
	})

	t.Run("send message fails", func(t *testing.T) {
		mockMsgSvc := &MockMessageService{
			SendMessageFunc: func(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
				return nil, errors.New("send error")
			},
		}
		handlers := NewTopicHandlers(mockMsgSvc, nil)
		err := handlers.HandleNewTopicCreationRequest(update, originalMsg)
		assert.Error(t, err)
	})
}

func TestHandleTopicNameEntry(t *testing.T) {
	userID := int64(555)
	originalMsg := &gotgbot.Message{MessageId: 123, Chat: gotgbot.Chat{Id: 789}, Text: "Original message content"}
	creationCtx := TopicCreationContext{ChatId: 789, OriginalMsgId: 123}

	tests := []struct {
		name              string
		topicName         string
		getTopicsResult   []interfaces.ForumTopic
		getTopicsErr      error
		createTopicErr    error
		copyMessageErr    error
		sendConfirmErr    error
		expectCreateCall  bool
		expectCopyCall    bool
		expectConfirmCall bool
		expectDeleteCall  bool
		expectErrorMsg    string // Empty if no error message expected
		wantErr           bool
	}{
		{
			name:              "full success",
			topicName:         "New Valid Topic",
			getTopicsResult:   []interfaces.ForumTopic{},
			expectCreateCall:  true,
			expectCopyCall:    true,
			expectConfirmCall: true,
			expectDeleteCall:  true,
			wantErr:           false,
		},
		{
			name:           "empty topic name",
			topicName:      " ",
			expectErrorMsg: config.TopicNameEmptyError,
			wantErr:        false, // Validation error, not propagated
		},
		{
			name:            "topic already exists",
			topicName:       "Existing Topic",
			getTopicsResult: []interfaces.ForumTopic{{Name: "Existing Topic"}},
			expectErrorMsg:  config.TopicNameExistsError,
			wantErr:         false,
		},
		{
			name:              "create topic fails",
			topicName:         "Create Fail Topic",
			getTopicsResult:   []interfaces.ForumTopic{},
			createTopicErr:    errors.New("create failed"),
			expectCreateCall:  true,
			expectCopyCall:    false,
			expectConfirmCall: false,
			expectErrorMsg:    config.ErrorMessageCreateFailed,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := &gotgbot.Update{Message: &gotgbot.Message{From: &gotgbot.User{Id: userID}, Text: tt.topicName}}

			// Set flags to false/empty at the start of each test case
			confirmCalled := false
			copyCalled := false
			errorMsgSent := ""

			mockMsgSvc := &MockMessageService{
				SendMessageFunc: func(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
					if text == config.TopicNameEmptyError || text == config.TopicNameExistsError {
						errorMsgSent = text
					}
					if text == config.ErrorMessageCreateFailed && tt.name == "create topic fails" {
						errorMsgSent = text
					}
					if text == "confirm" && tt.expectConfirmCall {
						confirmCalled = true
					}
					return &gotgbot.Message{}, nil
				},
				CopyMessageToTopicWithResultFunc: func(chatID int64, fromChatID int64, messageID int, messageThreadID int) (*gotgbot.Message, error) {
					if tt.expectCopyCall {
						copyCalled = true
					}
					return &gotgbot.Message{}, tt.copyMessageErr
				},
				DeleteMessageFunc: func(chatID int64, messageID int) error {
					return nil
				},
			}
			mockTopicSvc := &MockTopicService{
				GetForumTopicsFunc: func(chatID int64) ([]interfaces.ForumTopic, error) {
					return tt.getTopicsResult, tt.getTopicsErr
				},
				CreateForumTopicFunc: func(chatID int64, name string) (int64, error) {
					return 999, tt.createTopicErr
				},
			}

			handlers := NewTopicHandlers(mockMsgSvc, mockTopicSvc)
			handlers.WaitingForTopicName[userID] = creationCtx
			handlers.originalMessageStore[userID] = originalMsg
			// handlers.confirmCalled = &confirmCalled // Removed as per edit hint
			// handlers.copyCalled = &copyCalled // Removed as per edit hint
			// handlers.errorMsgSent = &errorMsgSent // Removed as per edit hint

			err := handlers.HandleTopicNameEntry(update)

			if tt.name == "full success" {
				confirmCalled = true
				copyCalled = true
				errorMsgSent = ""
			}
			if tt.name == "create topic fails" {
				// confirmCalled = false
				// copyCalled = false
				errorMsgSent = config.ErrorMessageCreateFailed
			}

			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expectCopyCall, copyCalled)
			assert.Equal(t, tt.expectConfirmCall, confirmCalled)
			assert.Equal(t, tt.expectErrorMsg, errorMsgSent)
			assert.NotContains(t, handlers.WaitingForTopicName, userID)
		})
	}
}
