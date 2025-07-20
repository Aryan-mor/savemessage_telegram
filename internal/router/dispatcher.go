package router

import (
	"strings"

	"save-message/internal/config"
	"save-message/internal/interfaces"
	"save-message/internal/logutils"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// Dispatcher routes incoming updates to the appropriate handlers.
type Dispatcher struct {
	MessageHandlers  interfaces.MessageHandlersInterface
	CallbackHandlers interfaces.CallbackHandlersInterface
	MessageService   interfaces.MessageServiceInterface
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(
	mh interfaces.MessageHandlersInterface,
	ch interfaces.CallbackHandlersInterface,
	ms interfaces.MessageServiceInterface,
) *Dispatcher {
	return &Dispatcher{
		MessageHandlers:  mh,
		CallbackHandlers: ch,
		MessageService:   ms,
	}
}

// HandleUpdate routes an update to the appropriate handler
func (d *Dispatcher) HandleUpdate(update *gotgbot.Update) error {
	logutils.Info("Raw update received", "update", update)
	// Handle nil updates gracefully
	if update == nil {
		logutils.Info("HandleUpdate: Received nil update, ignoring")
		return nil
	}

	logutils.Info("HandleUpdate", "updateID", update.UpdateId)

	// Handle my_chat_member (bot added/removed or admin status changed)
	if update.MyChatMember != nil {
		chat := update.MyChatMember.Chat
		from := update.MyChatMember.From
		logutils.Info("HandleUpdate: my_chat_member event", "chatID", chat.Id, "chatTitle", chat.Title, "fromUser", from.Username, "oldChatMember", update.MyChatMember.OldChatMember, "newChatMember", update.MyChatMember.NewChatMember)
		// (Welcome message logic temporarily removed until correct status field is confirmed)
		logutils.Success("HandleUpdate: my_chat_member event processed", "chatID", chat.Id)
		return nil
	}

	// Handle chat_member (user added/removed or admin status changed)
	if update.ChatMember != nil {
		chat := update.ChatMember.Chat
		from := update.ChatMember.From
		logutils.Info("HandleUpdate: chat_member event", "chatID", chat.Id, "chatTitle", chat.Title, "fromUser", from.Username, "oldChatMember", update.ChatMember.OldChatMember, "newChatMember", update.ChatMember.NewChatMember)
		// TODO: Add any additional handling for user added/removed/admin changes here
		logutils.Success("HandleUpdate: chat_member event processed", "chatID", chat.Id)
		return nil
	}

	// Handle callback queries (button clicks)
	if update.CallbackQuery != nil {
		logutils.Info("HandleUpdate: Routing to callback handler")
		return d.CallbackHandlers.HandleCallbackQuery(update)
	}

	// Handle messages
	if update.Message != nil {
		return d.handleMessage(update)
	}

	logutils.Warn("HandleUpdate: Unknown update type")
	return nil
}

// handleMessage routes message updates to appropriate handlers
func (d *Dispatcher) handleMessage(update *gotgbot.Update) error {
	logutils.Info("handleMessage", "chatID", update.Message.Chat.Id, "messageID", update.Message.MessageId, "threadID", update.Message.MessageThreadId)

	// Check if this is a new chat member (bot just joined)
	if update.Message.NewChatMembers != nil {
		for _, member := range update.Message.NewChatMembers {
			if member.Id == update.Message.From.Id { // Bot joined
				logutils.Info("handleMessage: Bot joined chat, sending welcome message")
				return d.MessageHandlers.HandleStartCommand(update)
			}
		}
	}

	// Check if message is NOT in General topic (thread 0) - only allow messages in General
	if update.Message.MessageThreadId != 0 {
		logutils.Info("handleMessage: Message detected in non-General topic, routing to non-General handler")
		return d.MessageHandlers.HandleNonGeneralTopicMessage(update)
	}

	// Check if message was recently moved
	if d.CallbackHandlers.IsRecentlyMovedMessage(update.Message.MessageId) {
		logutils.Info("handleMessage: Skipping recently moved message: %d", update.Message.MessageId)
		d.CallbackHandlers.CleanupMovedMessage(update.Message.MessageId)
		return nil
	}

	// Check if user is waiting to provide a topic name
	if d.CallbackHandlers.IsWaitingForTopicName(update.Message.From.Id) {
		logutils.Info("handleMessage: User is waiting for topic name, routing to topic name handler")
		return d.CallbackHandlers.HandleTopicNameEntry(update)
	}

	// Handle commands
	switch update.Message.Text {
	case "/start":
		logutils.Info("handleMessage: Routing to start command handler")
		return d.MessageHandlers.HandleStartCommand(update)
	case "/help":
		logutils.Info("handleMessage: Routing to help command handler")
		return d.MessageHandlers.HandleHelpCommand(update)
	case "/topics":
		logutils.Info("handleMessage: Routing to topics command handler")
		return d.MessageHandlers.HandleTopicsCommand(update)
	case "/addtopic":
		logutils.Info("handleMessage: Routing to add topic command handler")
		return d.MessageHandlers.HandleAddTopicCommand(update)
	default:
		// Handle regular messages (not commands)
		return d.handleRegularMessage(update)
	}
}

// handleRegularMessage handles regular (non-command) messages
func (d *Dispatcher) handleRegularMessage(update *gotgbot.Update) error {
	logutils.Info("handleRegularMessage", "chatID", update.Message.Chat.Id, "messageID", update.Message.MessageId)

	// Check if the message mentions the bot (handle both possible usernames)
	messageText := strings.ToLower(update.Message.Text)
	if update.Message.Text != "" && (strings.Contains(messageText, "@savemessagbot") || strings.Contains(messageText, "@savemessagebot")) {
		logutils.Info("handleRegularMessage: Message mentions bot, routing to bot mention handler")
		return d.MessageHandlers.HandleBotMention(update)
	}

	// Check if this is a forum chat
	if update.Message.Chat.Type == "supergroup" {
		logutils.Info("handleRegularMessage: Message in supergroup, routing to General topic handler")
		return d.MessageHandlers.HandleGeneralTopicMessage(update)
	}

	logutils.Info("handleRegularMessage: Message not in supergroup, skipping AI processing")
	return nil
}

// IsEditRequest checks if the message is an edit request
func (d *Dispatcher) IsEditRequest(update *gotgbot.Update) bool {
	if update == nil || update.Message == nil || update.Message.Text == "" {
		return false
	}
	return strings.HasPrefix(update.Message.Text, "Edit:")
}

// IsTopicSelection checks if the callback is a topic selection
func (d *Dispatcher) IsTopicSelection(update *gotgbot.Update) bool {
	if update == nil || update.CallbackQuery == nil {
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
	if update == nil || update.Message == nil || d.CallbackHandlers == nil {
		return false
	}
	return d.CallbackHandlers.IsWaitingForTopicName(update.Message.From.Id)
}

// IsMessageInGeneralTopic checks if the message is in the General topic
func (d *Dispatcher) IsMessageInGeneralTopic(update *gotgbot.Update) bool {
	if update == nil || update.Message == nil {
		return false
	}
	return update.Message.MessageThreadId == 0 && update.Message.Chat.Type == "supergroup"
}

func (d *Dispatcher) sendError(update *gotgbot.Update, err error) {
	chatID := getChatID(update)
	if chatID == 0 {
		logutils.Error("sendError: Could not determine chat ID", err)
		return
	}

	logutils.Error("sendError: Sending error message to user", err, "chatID", chatID)
	_, sendErr := d.MessageService.SendMessage(chatID, config.ErrorMessageFailed, nil)
	if sendErr != nil {
		logutils.Error("sendError: Failed to send error message", sendErr, "chatID", chatID)
	}
}

func getChatID(update *gotgbot.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.Id
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Chat.Id
	}
	return 0
}
