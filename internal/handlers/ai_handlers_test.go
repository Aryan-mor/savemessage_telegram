package handlers

import (
	"testing"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func TestAIHandlers_HandleGeneralTopicMessage_MainFlow(t *testing.T) {
	h := NewAIHandlers(nil, nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	if err := h.HandleGeneralTopicMessage(update); err != nil {
		t.Errorf("HandleGeneralTopicMessage returned error: %v", err)
	}
}

func TestAIHandlers_HandleRetryCallback_MainFlow(t *testing.T) {
	h := NewAIHandlers(nil, nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	msg := &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}
	if err := h.HandleRetryCallback(update, msg); err != nil {
		t.Errorf("HandleRetryCallback returned error: %v", err)
	}
}

func TestAIHandlers_HandleBackToSuggestionsCallback_MainFlow(t *testing.T) {
	h := NewAIHandlers(nil, nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	msg := &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}
	if err := h.HandleBackToSuggestionsCallback(update, msg); err != nil {
		t.Errorf("HandleBackToSuggestionsCallback returned error: %v", err)
	}
}
