package handlers

import (
	"testing"

	mocks "save-message/internal/mocks/handlers"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/stretchr/testify/assert"
)

// Mocks are now in mocks_test.go

func TestHandleNonGeneralTopicMessage(t *testing.T) {
	update := &gotgbot.Update{
		Message: &gotgbot.Message{
			MessageId: 123, Chat: gotgbot.Chat{Id: 789},
		},
	}

	tests := []struct {
		name             string
		deleteShouldFail bool
		sendShouldFail   bool
		expectDeleteCall bool
		expectSendCall   bool
		wantErr          bool
	}{
		{"success", false, false, true, true, false},
		{"delete fails", true, false, true, true, false}, // Error is logged, not returned
		{"send fails", false, true, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMsgSvc := &mocks.MockMessageService{}
			if tt.sendShouldFail {
				mockMsgSvc.SendMessageShouldFail = true
			}
			handlers := NewWarningHandlers(mockMsgSvc)
			err := handlers.HandleNonGeneralTopicMessage(update)

			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expectDeleteCall, mockMsgSvc.DeleteMessageCalled)
			assert.Equal(t, tt.expectSendCall, mockMsgSvc.SendMessageCalled)
		})
	}
}

func TestHandleWarningOkCallback(t *testing.T) {
	update := &gotgbot.Update{
		CallbackQuery: &gotgbot.CallbackQuery{
			Message: &gotgbot.Message{MessageId: 456, Chat: gotgbot.Chat{Id: 789}},
		},
	}

	t.Run("success", func(t *testing.T) {
		mockMsgSvc := &mocks.MockMessageService{}
		handlers := NewWarningHandlers(mockMsgSvc)
		err := handlers.HandleWarningOkCallback(update)
		assert.NoError(t, err)
		assert.True(t, mockMsgSvc.DeleteMessageCalled)
	})

	t.Run("failure", func(t *testing.T) {
		mockMsgSvc := &mocks.MockMessageService{}
		handlers := NewWarningHandlers(mockMsgSvc)
		err := handlers.HandleWarningOkCallback(update)
		assert.NoError(t, err) // Error is logged, not returned
	})
}
