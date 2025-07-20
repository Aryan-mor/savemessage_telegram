package handlers

import (
	"strings"

	"save-message/internal/interfaces"
	"save-message/internal/logutils"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// MessageHandlers handles regular messages.
type MessageHandlers struct {
	CommandHandlers interfaces.MessageHandlersInterface
	AIHandlers      interfaces.AIHandlersInterface
	TopicHandlers   interfaces.TopicHandlersInterface
	WarningHandlers interfaces.WarningHandlersInterface
	MessageService  interfaces.MessageServiceInterface
	BotUsername     string
}

// NewMessageHandlers creates a new instance of MessageHandlers.
func NewMessageHandlers(
	commandHandlers interfaces.MessageHandlersInterface,
	aiHandlers interfaces.AIHandlersInterface,
	topicHandlers interfaces.TopicHandlersInterface,
	warningHandlers interfaces.WarningHandlersInterface,
	messageService interfaces.MessageServiceInterface,
	botUsername string,
) *MessageHandlers {
	return &MessageHandlers{
		CommandHandlers: commandHandlers,
		AIHandlers:      aiHandlers,
		TopicHandlers:   topicHandlers,
		WarningHandlers: warningHandlers,
		MessageService:  messageService,
		BotUsername:     botUsername,
	}
}

// HandleMessage routes messages to the appropriate handler based on context.
func (mh *MessageHandlers) HandleMessage(update *gotgbot.Update) error {
	chatID := update.Message.Chat.Id
	logutils.Info("HandleMessage", "chatID", chatID, "messageID", update.Message.MessageId)

	var err error
	switch {
	case mh.isWaitingForTopic(update):
		logutils.Info("HandleMessage: Routing to TopicNameEntry", "chatID", chatID)
		err = mh.TopicHandlers.HandleTopicNameEntry(update)
	case mh.isCommand(update):
		logutils.Info("HandleMessage: Routing to Command", "chatID", chatID)
		err = mh.handleCommand(update)
	case mh.IsBotMention(update):
		logutils.Info("HandleMessage: Routing to BotMention", "chatID", chatID)
		err = mh.CommandHandlers.HandleBotMention(update)
	case mh.isGeneralTopicMessage(update):
		logutils.Info("HandleMessage: Routing to GeneralTopicMessage", "chatID", chatID)
		err = mh.AIHandlers.HandleGeneralTopicMessage(update)
	default:
		logutils.Warn("HandleMessage: Routing to NonGeneralTopicMessage", "chatID", chatID)
		err = mh.WarningHandlers.HandleNonGeneralTopicMessage(update)
	}

	if err != nil {
		logutils.Error("HandleMessage: HandlerError", err, "chatID", chatID)
	} else {
		logutils.Success("HandleMessage", "chatID", chatID)
	}
	return err
}

// HandleStartCommand delegates to command handlers
func (mh *MessageHandlers) HandleStartCommand(update *gotgbot.Update) error {
	return mh.CommandHandlers.HandleStartCommand(update)
}

// HandleHelpCommand delegates to command handlers
func (mh *MessageHandlers) HandleHelpCommand(update *gotgbot.Update) error {
	return mh.CommandHandlers.HandleHelpCommand(update)
}

// HandleTopicsCommand delegates to command handlers
func (mh *MessageHandlers) HandleTopicsCommand(update *gotgbot.Update) error {
	return mh.CommandHandlers.HandleTopicsCommand(update)
}

// HandleAddTopicCommand delegates to command handlers
func (mh *MessageHandlers) HandleAddTopicCommand(update *gotgbot.Update) error {
	return mh.CommandHandlers.HandleAddTopicCommand(update)
}

// HandleBotMention delegates to command handlers
func (mh *MessageHandlers) HandleBotMention(update *gotgbot.Update) error {
	return mh.CommandHandlers.HandleBotMention(update)
}

// HandleNonGeneralTopicMessage delegates to warning handlers
func (mh *MessageHandlers) HandleNonGeneralTopicMessage(update *gotgbot.Update) error {
	return mh.WarningHandlers.HandleNonGeneralTopicMessage(update)
}

// HandleGeneralTopicMessage delegates to AI handlers
func (mh *MessageHandlers) HandleGeneralTopicMessage(update *gotgbot.Update) error {
	return mh.AIHandlers.HandleGeneralTopicMessage(update)
}

// IsBotMention checks if the bot is mentioned in the message.
func (mh *MessageHandlers) IsBotMention(update *gotgbot.Update) bool {
	if update.Message == nil || update.Message.Entities == nil {
		return false
	}
	for _, entity := range update.Message.Entities {
		if entity.Type == "mention" {
			if strings.Contains(update.Message.Text, "@"+mh.BotUsername) {
				return true
			}
		}
	}
	return false
}

// IsRecentlyMovedMessage checks if message was recently moved
func (mh *MessageHandlers) IsRecentlyMovedMessage(messageID int64) bool {
	return mh.TopicHandlers.IsRecentlyMovedMessage(messageID)
}

// CleanupMovedMessage cleans up moved message tracking
func (mh *MessageHandlers) CleanupMovedMessage(messageID int64) {
	mh.TopicHandlers.CleanupMovedMessage(messageID)
}

// IsWaitingForTopicName checks if user is waiting for topic name
func (mh *MessageHandlers) IsWaitingForTopicName(userID int64) bool {
	return mh.TopicHandlers.IsWaitingForTopicName(userID)
}

// HandleTopicNameEntry delegates to topic handlers
func (mh *MessageHandlers) HandleTopicNameEntry(update *gotgbot.Update) error {
	return mh.TopicHandlers.HandleTopicNameEntry(update)
}

func (mh *MessageHandlers) isWaitingForTopic(update *gotgbot.Update) bool {
	return mh.TopicHandlers.IsWaitingForTopicName(update.Message.From.Id)
}

func (mh *MessageHandlers) isCommand(update *gotgbot.Update) bool {
	return update.Message.Text != "" && update.Message.Text[0] == '/'
}

func (mh *MessageHandlers) handleCommand(update *gotgbot.Update) error {
	logutils.Info("handleCommand", "command", update.Message.Text)
	switch update.Message.Text {
	case "/start":
		return mh.CommandHandlers.HandleStartCommand(update)
	case "/help":
		return mh.CommandHandlers.HandleHelpCommand(update)
	case "/topics":
		return mh.CommandHandlers.HandleTopicsCommand(update)
	case "/addtopic":
		return mh.CommandHandlers.HandleAddTopicCommand(update)
	default:
		_, err := mh.MessageService.SendMessage(update.Message.Chat.Id, "Unknown command. Try /help", nil)
		if err != nil {
			logutils.Error("handleCommand: SendMessageError", err, "command", update.Message.Text)
		}
		return err
	}
}

func (mh *MessageHandlers) isGeneralTopicMessage(update *gotgbot.Update) bool {
	return update.Message.MessageThreadId == 0
}
