package router

import (
	"log"
	"strings"

	"save-message/internal/handlers"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// Dispatcher routes incoming updates to appropriate handlers
type Dispatcher struct {
	messageHandlers  *handlers.MessageHandlers
	callbackHandlers *handlers.CallbackHandlers
}

// NewDispatcher creates a new dispatcher
func NewDispatcher(messageHandlers *handlers.MessageHandlers, callbackHandlers *handlers.CallbackHandlers) *Dispatcher {
	return &Dispatcher{
		messageHandlers:  messageHandlers,
		callbackHandlers: callbackHandlers,
	}
}

// HandleUpdate routes an update to the appropriate handler
func (d *Dispatcher) HandleUpdate(update *gotgbot.Update) error {
	log.Printf("[Dispatcher] Handling update: UpdateID=%d", update.UpdateId)

	// Handle callback queries (button clicks)
	if update.CallbackQuery != nil {
		log.Printf("[Dispatcher] Routing to callback handler")
		return d.callbackHandlers.HandleCallbackQuery(update)
	}

	// Handle messages
	if update.Message != nil {
		return d.handleMessage(update)
	}

	log.Printf("[Dispatcher] Unknown update type")
	return nil
}

// handleMessage routes message updates to appropriate handlers
func (d *Dispatcher) handleMessage(update *gotgbot.Update) error {
	log.Printf("[Dispatcher] Handling message: ChatID=%d, MessageID=%d, ThreadID=%d",
		update.Message.Chat.Id, update.Message.MessageId, update.Message.MessageThreadId)

	// Check if this is a new chat member (bot just joined)
	if update.Message.NewChatMembers != nil {
		for _, member := range update.Message.NewChatMembers {
			if member.Id == update.Message.From.Id { // Bot joined
				log.Printf("[Dispatcher] Bot joined chat, sending welcome message")
				return d.messageHandlers.HandleStartCommand(update)
			}
		}
	}

	// Check if message is NOT in General topic (thread 0) - only allow messages in General
	if update.Message.MessageThreadId != 0 {
		log.Printf("[Dispatcher] Message detected in non-General topic, routing to non-General handler")
		return d.messageHandlers.HandleNonGeneralTopicMessage(update)
	}

	// Check if message was recently moved
	if d.callbackHandlers.IsRecentlyMovedMessage(update.Message.MessageId) {
		log.Printf("[Dispatcher] Skipping recently moved message: %d", update.Message.MessageId)
		d.callbackHandlers.CleanupMovedMessage(update.Message.MessageId)
		return nil
	}

	// Check if user is waiting to provide a topic name
	if d.callbackHandlers.IsWaitingForTopicName(update.Message.From.Id) {
		log.Printf("[Dispatcher] User is waiting for topic name, routing to topic name handler")
		return d.callbackHandlers.HandleTopicNameEntry(update)
	}

	// Handle commands
	switch update.Message.Text {
	case "/start":
		log.Printf("[Dispatcher] Routing to start command handler")
		return d.messageHandlers.HandleStartCommand(update)
	case "/help":
		log.Printf("[Dispatcher] Routing to help command handler")
		return d.messageHandlers.HandleHelpCommand(update)
	case "/topics":
		log.Printf("[Dispatcher] Routing to topics command handler")
		return d.messageHandlers.HandleTopicsCommand(update)
	case "/addtopic":
		log.Printf("[Dispatcher] Routing to add topic command handler")
		return d.messageHandlers.HandleAddTopicCommand(update)
	default:
		// Handle regular messages (not commands)
		return d.handleRegularMessage(update)
	}
}

// handleRegularMessage handles regular (non-command) messages
func (d *Dispatcher) handleRegularMessage(update *gotgbot.Update) error {
	log.Printf("[Dispatcher] Handling regular message: ChatID=%d, MessageID=%d",
		update.Message.Chat.Id, update.Message.MessageId)

	// Check if the message mentions the bot (handle both possible usernames)
	messageText := strings.ToLower(update.Message.Text)
	if update.Message.Text != "" && (strings.Contains(messageText, "@savemessagbot") || strings.Contains(messageText, "@savemessagebot")) {
		log.Printf("[Dispatcher] Message mentions bot, routing to bot mention handler")
		return d.messageHandlers.HandleBotMention(update)
	}

	// Check if this is a forum chat
	if update.Message.Chat.Type == "supergroup" {
		log.Printf("[Dispatcher] Message in supergroup, routing to General topic handler")
		return d.messageHandlers.HandleGeneralTopicMessage(update)
	}

	log.Printf("[Dispatcher] Message not in supergroup, skipping AI processing")
	return nil
}

// IsEditRequest checks if the message is an edit request
func (d *Dispatcher) IsEditRequest(update *gotgbot.Update) bool {
	if update.Message == nil || update.Message.Text == "" {
		return false
	}
	return strings.HasPrefix(update.Message.Text, "Edit:")
}

// IsTopicSelection checks if the callback is a topic selection
func (d *Dispatcher) IsTopicSelection(update *gotgbot.Update) bool {
	if update.CallbackQuery == nil {
		return false
	}
	callbackData := update.CallbackQuery.Data
	// Topic selection callbacks have format: "TopicName_MessageId"
	parts := strings.Split(callbackData, "_")
	return len(parts) >= 2 && !strings.HasPrefix(callbackData, "create_new_folder_") &&
		!strings.HasPrefix(callbackData, "retry_") &&
		!strings.HasPrefix(callbackData, "show_all_topics_") &&
		!strings.HasPrefix(callbackData, "back_to_suggestions_") &&
		callbackData != "create_topic_menu" &&
		callbackData != "show_all_topics_menu" &&
		!strings.HasPrefix(callbackData, "detectMessageOnOtherTopic_ok_")
}

// IsNewTopicPrompt checks if the user is waiting for a topic name
func (d *Dispatcher) IsNewTopicPrompt(update *gotgbot.Update) bool {
	if update.Message == nil {
		return false
	}
	return d.callbackHandlers.IsWaitingForTopicName(update.Message.From.Id)
}

// IsMessageInGeneralTopic checks if the message is in the General topic
func (d *Dispatcher) IsMessageInGeneralTopic(update *gotgbot.Update) bool {
	if update.Message == nil {
		return false
	}
	return update.Message.MessageThreadId == 0 && update.Message.Chat.Type == "supergroup"
}

// SendUnknownActionNotice sends a notice for unknown actions
func (d *Dispatcher) SendUnknownActionNotice(update *gotgbot.Update) error {
	log.Printf("[Dispatcher] Unknown action, sending notice")

	var chatID int64
	if update.Message != nil {
		chatID = update.Message.Chat.Id
	} else if update.CallbackQuery != nil {
		chatID = update.CallbackQuery.Message.Chat.Id
	} else {
		return nil
	}

	_, err := d.messageHandlers.CommandHandlers.MessageService.SendMessage(chatID, "❓ Unknown action. Please try again.", nil)
	if err != nil {
		log.Printf("[Dispatcher] Error sending unknown action notice: %v", err)
	}
	return err
}
