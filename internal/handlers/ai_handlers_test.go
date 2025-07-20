package handlers

import (
	"os"
	"testing"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func TestAIHandlers_HandleGeneralTopicMessage_MainFlow(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("Skipping TestAIHandlers_HandleGeneralTopicMessage_MainFlow: not running integration test (set RUN_INTEGRATION=1 to enable)")
	}
	h := NewAIHandlers(nil, nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	if err := h.HandleGeneralTopicMessage(update); err != nil {
		t.Errorf("HandleGeneralTopicMessage returned error: %v", err)
	}
}

func TestAIHandlers_HandleRetryCallback_MainFlow(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("Skipping TestAIHandlers_HandleRetryCallback_MainFlow: not running integration test (set RUN_INTEGRATION=1 to enable)")
	}
	h := NewAIHandlers(nil, nil, nil)
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
	h := NewAIHandlers(nil, nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	msg := &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}
	if err := h.HandleBackToSuggestionsCallback(update, msg); err != nil {
		t.Errorf("HandleBackToSuggestionsCallback returned error: %v", err)
	}
}
