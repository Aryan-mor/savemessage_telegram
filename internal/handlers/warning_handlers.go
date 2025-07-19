package handlers

import (
	"log"
	"strconv"
	"strings"
	"time"

	"save-message/internal/config"
	"save-message/internal/services"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// WarningHandlers handles warning messages and non-General topic interactions
type WarningHandlers struct {
	messageService *services.MessageService
}

// NewWarningHandlers creates a new warning handlers instance
func NewWarningHandlers(messageService *services.MessageService) *WarningHandlers {
	return &WarningHandlers{
		messageService: messageService,
	}
}

// HandleNonGeneralTopicMessage handles messages sent in non-General topics
func (wh *WarningHandlers) HandleNonGeneralTopicMessage(update *gotgbot.Update) error {
	log.Printf("[WarningHandlers] Handling non-General topic message: ChatID=%d, ThreadID=%d, MessageID=%d",
		update.Message.Chat.Id, update.Message.MessageThreadId, update.Message.MessageId)

	// Delete the user's message immediately
	err := wh.messageService.DeleteMessage(update.Message.Chat.Id, int(update.Message.MessageId))
	if err != nil {
		log.Printf("[WarningHandlers] Error deleting message from non-General topic: %v", err)
	} else {
		log.Printf("[WarningHandlers] Successfully deleted message from non-General topic: MessageID=%d", update.Message.MessageId)
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
		log.Printf("[WarningHandlers] Error sending warning message: %v", err)
		return err
	}

	log.Printf("[WarningHandlers] Successfully sent warning message: MessageID=%d", warningMsg.MessageId)

	// Auto-delete warning message after 1 minute
	go func(botToken string, chatID int64, messageID int64, threadID int64) {
		time.Sleep(config.DefaultWarningAutoDeleteDelay)
		err := wh.messageService.DeleteMessage(chatID, int(messageID))
		if err != nil {
			log.Printf("[WarningHandlers] Error auto-deleting warning message: %v", err)
		} else {
			log.Printf("[WarningHandlers] Successfully auto-deleted warning message: MessageID=%d", messageID)
		}
	}(wh.messageService.BotToken, update.Message.Chat.Id, warningMsg.MessageId, update.Message.MessageThreadId)

	return nil
}

// HandleWarningOkCallback handles the "Ok" button for warning messages
func (wh *WarningHandlers) HandleWarningOkCallback(update *gotgbot.Update) error {
	log.Printf("[WarningHandlers] Handling warning OK callback: %s", update.CallbackQuery.Data)

	// Delete the warning message itself (the message that contains the "Ok" button)
	err := wh.messageService.DeleteMessage(update.CallbackQuery.Message.Chat.Id, int(update.CallbackQuery.Message.MessageId))
	if err != nil {
		log.Printf("[WarningHandlers] Error deleting warning message: %v", err)
	} else {
		log.Printf("[WarningHandlers] Successfully deleted warning message: MessageID=%d", update.CallbackQuery.Message.MessageId)
	}

	return nil
}

// IsWarningCallback checks if the callback is a warning OK callback
func (wh *WarningHandlers) IsWarningCallback(callbackData string) bool {
	return strings.HasPrefix(callbackData, config.CallbackPrefixDetectMessageOnOtherTopic)
}
