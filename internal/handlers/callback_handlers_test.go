package handlers

import (
	"save-message/internal/mocks/handlers"
	mocks "save-message/internal/mocks/handlers"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleCallbackQuery(t *testing.T) {
	tests := []struct {
		name            string
		callbackData    string
		messageInStore  bool
		expectedHandler string // "topic", "warning", "ai"
		wantErr         bool
	}{
		{"warning callback", "warning_123", true, "warning", false},
		{"message not in store", "some_other_callback", false, "", false},
		{"create new folder", "create_new_folder_123", true, "topic", false},
		{"retry", "retry_123", true, "ai", false},
		{"show all topics", "show_all_topics_123", true, "topic", false},
		{"create topic menu", "create_topic_menu", true, "topic", false},
		{"show all topics menu", "show_all_topics_menu", true, "topic", false},
		{"back to suggestions", "back_to_suggestions_123", true, "ai", false},
		{"topic selection", "Work_123", true, "topic", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMsgSvc := &mocks.MockMessageService{}
			mockTopic := &handlers.MockTopicHandlers{}
			mockAI := &handlers.MockAIHandlers{}
			mockWarning := &handlers.MockWarningHandlers{}

			callbackHandlers := NewCallbackHandlers(mockMsgSvc, mockTopic, mockAI, mockWarning)
			_ = callbackHandlers // No-op for now
		})
	}
}

func TestCallbackHandlers_StateManagement(t *testing.T) {
	mockTopicH := &handlers.MockTopicHandlers{}
	ch := NewCallbackHandlers(nil, mockTopicH, &handlers.MockAIHandlers{}, &handlers.MockWarningHandlers{})
	userID := int64(123)
	msgID := int64(456)

	// Test IsWaitingForTopicName
	assert.False(t, ch.IsWaitingForTopicName(userID))

	// Test IsRecentlyMovedMessage
	assert.False(t, ch.IsRecentlyMovedMessage(msgID))
	ch.MarkMessageAsMoved(msgID)
	ch.CleanupMovedMessage(msgID)
}
