package handlers

import (
	"strconv"
	"strings"
	"time"

	"save-message/internal/config"
	"save-message/internal/interfaces"
	"save-message/internal/logutils"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// WarningHandlers handles warning messages and non-General topic interactions
type WarningHandlers struct {
	messageService interfaces.MessageServiceInterface

	// Add BotUserID for self-detection
	BotUserID int64

	// Mockable funcs for testing
	HandleNonGeneralTopicMessageFunc func(update *gotgbot.Update) error
	HandleWarningOkCallbackFunc      func(update *gotgbot.Update) error
}

// NewWarningHandlers creates a new warning handlers instance
func NewWarningHandlers(messageService interfaces.MessageServiceInterface) *WarningHandlers {
	return &WarningHandlers{
		messageService: messageService,
	}
}

// HandleNonGeneralTopicMessage handles messages sent in non-General topics
func (wh *WarningHandlers) HandleNonGeneralTopicMessage(update *gotgbot.Update) error {
	if wh.HandleNonGeneralTopicMessageFunc != nil {
		return wh.HandleNonGeneralTopicMessageFunc(update)
	}
	logutils.Warn("HandleNonGeneralTopicMessage", "chatID", update.Message.Chat.Id, "threadID", update.Message.MessageThreadId, "messageID", update.Message.MessageId)

	// Skip deleting if message is from the bot itself
	if update.Message.From != nil && update.Message.From.IsBot && wh.BotUserID != 0 && update.Message.From.Id == wh.BotUserID {
		logutils.Info("HandleNonGeneralTopicMessage: Skipping delete for bot's own message", "chatID", update.Message.Chat.Id, "messageID", update.Message.MessageId)
		return nil
	}

	// Delete the user's message immediately
	err := wh.messageService.DeleteMessage(update.Message.Chat.Id, int(update.Message.MessageId))
	if err != nil {
		logutils.Error("HandleNonGeneralTopicMessage: DeleteMessageError", err, "chatID", update.Message.Chat.Id, "messageID", update.Message.MessageId)
	} else {
		logutils.Success("HandleNonGeneralTopicMessage", "chatID", update.Message.Chat.Id, "messageID", update.Message.MessageId)
	}

	// Send warning message with "Ok" button
	callbackData := config.CallbackPrefixDetectMessageOnOtherTopic + strconv.FormatInt(update.Message.MessageId, 10)
	keyboard := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextOk, CallbackData: callbackData}},
		},
	}

	warningMsg, err := wh.messageService.SendMessage(update.Message.Chat.Id,
		config.WarningNonGeneralTopic,
		&gotgbot.SendMessageOpts{
			MessageThreadId: update.Message.MessageThreadId,
			ParseMode:       "Markdown",
			ReplyMarkup:     *keyboard,
		})

	if err != nil {
		logutils.Error("HandleNonGeneralTopicMessage: SendMessageError", err, "chatID", update.Message.Chat.Id, "messageID", update.Message.MessageId)
		return err
	}

	logutils.Success("HandleNonGeneralTopicMessage", "chatID", update.Message.Chat.Id, "messageID", warningMsg.MessageId)

	// Auto-delete warning message after 1 minute
	go func(chatID int64, messageID int64, threadID int64) {
		time.Sleep(config.DefaultWarningAutoDeleteDelay)
		err := wh.messageService.DeleteMessage(chatID, int(messageID))
		if err != nil {
			logutils.Error("HandleNonGeneralTopicMessage: AutoDeleteMessageError", err, "chatID", chatID, "messageID", messageID)
		} else {
			logutils.Success("HandleNonGeneralTopicMessage", "chatID", chatID, "messageID", messageID)
		}
	}(update.Message.Chat.Id, warningMsg.MessageId, update.Message.MessageThreadId)

	return nil
}

// HandleWarningOkCallback handles the "Ok" button for warning messages
func (wh *WarningHandlers) HandleWarningOkCallback(update *gotgbot.Update) error {
	if wh.HandleWarningOkCallbackFunc != nil {
		return wh.HandleWarningOkCallbackFunc(update)
	}
	logutils.Warn("HandleWarningOkCallback", "callbackData", update.CallbackQuery.Data)

	// Delete the warning message itself (the message that contains the "Ok" button)
	err := wh.messageService.DeleteMessage(update.CallbackQuery.Message.Chat.Id, int(update.CallbackQuery.Message.MessageId))
	if err != nil {
		logutils.Error("HandleWarningOkCallback: DeleteMessageError", err, "chatID", update.CallbackQuery.Message.Chat.Id, "messageID", update.CallbackQuery.Message.MessageId)
	} else {
		logutils.Success("HandleWarningOkCallback", "chatID", update.CallbackQuery.Message.Chat.Id, "messageID", update.CallbackQuery.Message.MessageId)
	}

	return nil
}

// IsWarningCallback checks if a callback is a warning confirmation.
func (wh *WarningHandlers) IsWarningCallback(callbackData string) bool {
	return strings.HasPrefix(callbackData, config.CallbackPrefixDetectMessageOnOtherTopic)
}
