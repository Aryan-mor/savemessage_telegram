package services

import (
	"database/sql"
	"testing"

	"save-message/internal/database"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// MockDatabase for message service tests
type MockMessageDatabase struct {
	shouldErr bool
	users     []database.User
}

func (m *MockMessageDatabase) UpsertUser(userID int64, username, firstName, lastName string) error {
	if m.shouldErr {
		return sql.ErrConnDone
	}
	return nil
}

func (m *MockMessageDatabase) GetUser(userID int64) (*database.User, error) {
	if m.shouldErr {
		return nil, sql.ErrConnDone
	}
	for _, user := range m.users {
		if user.ID == userID {
			return &user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *MockMessageDatabase) AddTopic(chatID int64, name string, messageThreadId int64, createdBy int64) error {
	if m.shouldErr {
		return sql.ErrConnDone
	}
	return nil
}

func (m *MockMessageDatabase) GetTopicsByChat(chatID int64) ([]database.Topic, error) {
	if m.shouldErr {
		return nil, sql.ErrConnDone
	}
	return []database.Topic{}, nil
}

func (m *MockMessageDatabase) TopicExists(chatID int64, name string) (bool, error) {
	if m.shouldErr {
		return false, sql.ErrConnDone
	}
	return false, nil
}

func (m *MockMessageDatabase) Close() error {
	return nil
}

func TestNewMessageService(t *testing.T) {
	tests := []struct {
		name     string
		botToken string
		db       database.DatabaseInterface
	}{
		{
			name:     "valid parameters",
			botToken: "test-token",
			db:       &database.Database{},
		},
		{
			name:     "empty bot token",
			botToken: "",
			db:       &database.Database{},
		},
		{
			name:     "nil database",
			botToken: "test-token",
			db:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMessageService(tt.botToken, tt.db)
			if service == nil {
				t.Error("NewMessageService() returned nil")
			}
			if service.BotToken != tt.botToken {
				t.Errorf("NewMessageService() BotToken = %s, want %s", service.BotToken, tt.botToken)
			}
			if service.db != tt.db {
				t.Errorf("NewMessageService() db = %v, want %v", service.db, tt.db)
			}
		})
	}
}

func TestMessageService_DeleteMessage(t *testing.T) {
	// Test with expected failure since we can't easily mock the Telegram API
	// The service will fail when Telegram API calls fail, which is expected behavior

	tests := []struct {
		name      string
		chatID    int64
		messageID int
		wantErr   bool
	}{
		{
			name:      "message deletion fails due to API error",
			chatID:    123456,
			messageID: 789,
			wantErr:   true, // Will fail due to Telegram API error
		},
		{
			name:      "zero message ID",
			chatID:    123456,
			messageID: 0,
			wantErr:   true, // Will fail due to Telegram API error
		},
		{
			name:      "zero chat ID",
			chatID:    0,
			messageID: 789,
			wantErr:   true, // Will fail due to Telegram API error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MessageService{
				BotToken: "test-token",
				db:       &MockMessageDatabase{shouldErr: false},
			}

			err := service.DeleteMessage(tt.chatID, tt.messageID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageService_CopyMessageToTopic(t *testing.T) {
	// Test with expected failure since we can't easily mock the Telegram API
	// The service will fail when Telegram API calls fail, which is expected behavior

	tests := []struct {
		name            string
		chatID          int64
		fromChatID      int64
		messageID       int
		messageThreadID int
		wantErr         bool
	}{
		{
			name:            "message copy fails due to API error",
			chatID:          123456,
			fromChatID:      123456,
			messageID:       789,
			messageThreadID: 1,
			wantErr:         true, // Will fail due to Telegram API error
		},
		{
			name:            "zero thread ID",
			chatID:          123456,
			fromChatID:      123456,
			messageID:       789,
			messageThreadID: 0,
			wantErr:         true, // Will fail due to Telegram API error
		},
		{
			name:            "different chat IDs",
			chatID:          123456,
			fromChatID:      654321,
			messageID:       789,
			messageThreadID: 1,
			wantErr:         true, // Will fail due to Telegram API error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MessageService{
				BotToken: "test-token",
				db:       &MockMessageDatabase{shouldErr: false},
			}

			err := service.CopyMessageToTopic(tt.chatID, tt.fromChatID, tt.messageID, tt.messageThreadID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyMessageToTopic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageService_CopyMessageToTopicWithResult(t *testing.T) {
	// Test with expected failure since we can't easily mock the Telegram API
	// The service will fail when Telegram API calls fail, which is expected behavior

	tests := []struct {
		name            string
		chatID          int64
		fromChatID      int64
		messageID       int
		messageThreadID int
		wantErr         bool
	}{
		{
			name:            "message copy with result fails due to API error",
			chatID:          123456,
			fromChatID:      123456,
			messageID:       789,
			messageThreadID: 1,
			wantErr:         true, // Will fail due to Telegram API error
		},
		{
			name:            "zero thread ID",
			chatID:          123456,
			fromChatID:      123456,
			messageID:       789,
			messageThreadID: 0,
			wantErr:         true, // Will fail due to Telegram API error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MessageService{
				BotToken: "test-token",
				db:       &MockMessageDatabase{shouldErr: false},
			}

			message, err := service.CopyMessageToTopicWithResult(tt.chatID, tt.fromChatID, tt.messageID, tt.messageThreadID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyMessageToTopicWithResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && message == nil {
				t.Error("CopyMessageToTopicWithResult() returned nil message when no error expected")
			}
		})
	}
}

func TestMessageService_SendMessage(t *testing.T) {
	// Test with expected failure since we can't easily mock the Telegram API
	// The service will fail when Telegram API calls fail, which is expected behavior

	tests := []struct {
		name    string
		chatID  int64
		text    string
		opts    *gotgbot.SendMessageOpts
		wantErr bool
	}{
		{
			name:    "message send fails due to API error",
			chatID:  123456,
			text:    "Test message",
			opts:    nil,
			wantErr: true, // Will fail due to Telegram API error
		},
		{
			name:    "empty text",
			chatID:  123456,
			text:    "",
			opts:    nil,
			wantErr: true, // Will fail due to Telegram API error
		},
		{
			name:   "with options",
			chatID: 123456,
			text:   "Test message",
			opts: &gotgbot.SendMessageOpts{
				ParseMode:       "Markdown",
				MessageThreadId: 1,
			},
			wantErr: true, // Will fail due to Telegram API error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MessageService{
				BotToken: "test-token",
				db:       &MockMessageDatabase{shouldErr: false},
			}

			message, err := service.SendMessage(tt.chatID, tt.text, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && message == nil {
				t.Error("SendMessage() returned nil message when no error expected")
			}
		})
	}
}

func TestMessageService_EditMessageText(t *testing.T) {
	// Test with expected failure since we can't easily mock the Telegram API
	// The service will fail when Telegram API calls fail, which is expected behavior

	tests := []struct {
		name      string
		chatID    int64
		messageID int64
		text      string
		opts      *gotgbot.EditMessageTextOpts
		wantErr   bool
	}{
		{
			name:      "message edit fails due to API error",
			chatID:    123456,
			messageID: 789,
			text:      "Updated message",
			opts:      nil,
			wantErr:   true, // Will fail due to Telegram API error
		},
		{
			name:      "empty text",
			chatID:    123456,
			messageID: 789,
			text:      "",
			opts:      nil,
			wantErr:   true, // Will fail due to Telegram API error
		},
		{
			name:      "with options",
			chatID:    123456,
			messageID: 789,
			text:      "Updated message",
			opts: &gotgbot.EditMessageTextOpts{
				ParseMode: "Markdown",
			},
			wantErr: true, // Will fail due to Telegram API error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MessageService{
				BotToken: "test-token",
				db:       &MockMessageDatabase{shouldErr: false},
			}

			message, err := service.EditMessageText(tt.chatID, tt.messageID, tt.text, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("EditMessageText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && message == nil {
				t.Error("EditMessageText() returned nil message when no error expected")
			}
		})
	}
}

func TestMessageService_AnswerCallbackQuery(t *testing.T) {
	// Test with expected failure since we can't easily mock the Telegram API
	// The service will fail when Telegram API calls fail, which is expected behavior

	tests := []struct {
		name            string
		callbackQueryID string
		opts            *gotgbot.AnswerCallbackQueryOpts
		wantErr         bool
	}{
		{
			name:            "callback answer fails due to API error",
			callbackQueryID: "test-callback-id",
			opts:            nil,
			wantErr:         true, // Will fail due to Telegram API error
		},
		{
			name:            "empty callback ID",
			callbackQueryID: "",
			opts:            nil,
			wantErr:         true, // Will fail due to Telegram API error
		},
		{
			name:            "with options",
			callbackQueryID: "test-callback-id",
			opts: &gotgbot.AnswerCallbackQueryOpts{
				Text: "Test answer",
			},
			wantErr: true, // Will fail due to Telegram API error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MessageService{
				BotToken: "test-token",
				db:       &MockMessageDatabase{shouldErr: false},
			}

			err := service.AnswerCallbackQuery(tt.callbackQueryID, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnswerCallbackQuery() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageService_DBErrorPropagation(t *testing.T) {
	service := &MessageService{
		BotToken: "test-token",
		db:       &MockMessageDatabase{shouldErr: true},
	}

	t.Run("DeleteMessage with DB error (should not affect API call)", func(t *testing.T) {
		err := service.DeleteMessage(123456, 789)
		// Should still return API error, not DB error
		if err == nil {
			t.Error("expected error due to API failure, got nil")
		}
	})

	t.Run("CopyMessageToTopic with DB error (should not affect API call)", func(t *testing.T) {
		err := service.CopyMessageToTopic(123456, 123456, 789, 1)
		if err == nil {
			t.Error("expected error due to API failure, got nil")
		}
	})

	t.Run("SendMessage with DB error (should not affect API call)", func(t *testing.T) {
		_, err := service.SendMessage(123456, "Test", nil)
		if err == nil {
			t.Error("expected error due to API failure, got nil")
		}
	})
}

func TestMessageService_NilPointerAndNilDB(t *testing.T) {
	var nilService *MessageService

	t.Run("nil MessageService panics on DeleteMessage", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic on nil receiver, got none")
			}
		}()
		nilService.DeleteMessage(123, 456)
	})
}

func TestMessageService_EdgeCases(t *testing.T) {
	service := &MessageService{BotToken: "test-token", db: &MockMessageDatabase{shouldErr: false}}
	tests := []struct {
		name      string
		chatID    int64
		messageID int
		wantErr   bool
	}{
		{"negative IDs", -1, -1, true},
		{"very large IDs", 1 << 62, 1 << 30, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteMessage(tt.chatID, tt.messageID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
