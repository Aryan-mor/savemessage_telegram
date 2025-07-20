package handlers

import (
	"testing"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func TestCommandHandlers_HandleHelpCommand_MainFlow(t *testing.T) {
	h := NewCommandHandlers(nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	if err := h.HandleHelpCommand(update); err != nil {
		t.Errorf("HandleHelpCommand returned error: %v", err)
	}
}

func TestCommandHandlers_HandleAddTopicCommand_MainFlow(t *testing.T) {
	h := NewCommandHandlers(nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	if err := h.HandleAddTopicCommand(update); err != nil {
		t.Errorf("HandleAddTopicCommand returned error: %v", err)
	}
}

func TestCommandHandlers_HandleBotMention_MainFlow(t *testing.T) {
	h := NewCommandHandlers(nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}, Text: "@bot"}}
	if err := h.HandleBotMention(update); err != nil {
		t.Errorf("HandleBotMention returned error: %v", err)
	}
}

func TestCommandHandlers_IsBotMention_MainFlow(t *testing.T) {
	h := NewCommandHandlers(nil, nil)
	msg := &gotgbot.Message{Text: "@bot"}
	_ = h.IsBotMention(msg.Text) // Pass string, not *gotgbot.Message
}

func TestCommandHandlers_HandleNonGeneralTopicMessage_MainFlow(t *testing.T) {
	h := NewCommandHandlers(nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	if err := h.HandleNonGeneralTopicMessage(update); err != nil {
		t.Errorf("HandleNonGeneralTopicMessage returned error: %v", err)
	}
}

func TestCommandHandlers_HandleGeneralTopicMessage_MainFlow(t *testing.T) {
	h := NewCommandHandlers(nil, nil)
	update := &gotgbot.Update{Message: &gotgbot.Message{MessageId: 1, Chat: gotgbot.Chat{Id: 123}}}
	if err := h.HandleGeneralTopicMessage(update); err != nil {
		t.Errorf("HandleGeneralTopicMessage returned error: %v", err)
	}
}
