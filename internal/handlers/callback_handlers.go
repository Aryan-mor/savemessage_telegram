package handlers

import (
	"strings"

	"save-message/internal/config"
	"save-message/internal/interfaces"
	"save-message/internal/logutils"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// CallbackHandlers coordinates all callback-related interactions
type CallbackHandlers struct {
	TopicHandlers   interfaces.TopicHandlersInterface
	WarningHandlers interfaces.WarningHandlersInterface
	AIHandlers      interfaces.AIHandlersInterface
	MessageService  interfaces.MessageServiceInterface
}

// TopicCreationContext stores context for topic creation
type TopicCreationContext struct {
	ChatId        int64
	ThreadId      int64
	OriginalMsgId int64
}

// NewCallbackHandlers creates a new callback handlers instance
func NewCallbackHandlers(
	messageService interfaces.MessageServiceInterface,
	topicHandlers interfaces.TopicHandlersInterface,
	aiHandlers interfaces.AIHandlersInterface,
	warningHandlers interfaces.WarningHandlersInterface,
) *CallbackHandlers {
	return &CallbackHandlers{
		TopicHandlers:   topicHandlers,
		WarningHandlers: warningHandlers,
		AIHandlers:      aiHandlers,
		MessageService:  messageService,
	}
}

// HandleCallbackQuery routes callback queries to appropriate handlers
func (ch *CallbackHandlers) HandleCallbackQuery(update *gotgbot.Update) error {
	callbackData := update.CallbackQuery.Data
	chatID := update.CallbackQuery.Message.Chat.Id
	logutils.Info("HandleCallbackQuery", "chatID", chatID, "callbackData", callbackData)

	// Answer the callback query to remove the loading state
	err := ch.MessageService.AnswerCallbackQuery(update.CallbackQuery.Id, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Processing...",
	})
	if err != nil {
		logutils.Error("HandleCallbackQuery: Error answering callback query", err, "chatID", chatID, "callbackData", callbackData)
	} else {
		logutils.Success("HandleCallbackQuery: Callback query answered", "chatID", chatID, "callbackData", callbackData)
	}

	// Special handling for warning callbacks
	if ch.WarningHandlers.IsWarningCallback(callbackData) {
		logutils.Info("HandleCallbackQuery: Handling warning callback", "chatID", chatID, "callbackData", callbackData)
		err = ch.WarningHandlers.HandleWarningOkCallback(update)
		if err != nil {
			logutils.Error("HandleCallbackQuery: Error handling warning callback", err, "chatID", chatID, "callbackData", callbackData)
		} else {
			logutils.Success("HandleCallbackQuery: Warning callback handled", "chatID", chatID, "callbackData", callbackData)
		}
		return err
	}

	// Get original message from topic handlers
	originalMsg := ch.TopicHandlers.GetMessageByCallbackData(callbackData)
	if originalMsg == nil {
		logutils.Warn("HandleCallbackQuery: Original message not found", "chatID", chatID, "callbackData", callbackData)
		_, err := ch.MessageService.SendMessage(update.CallbackQuery.From.Id, config.ErrorMessageNotFound, nil)
		if err != nil {
			logutils.Error("HandleCallbackQuery: Error sending error message", err, "chatID", chatID, "callbackData", callbackData)
		} else {
			logutils.Success("HandleCallbackQuery: Error message sent", "chatID", chatID, "callbackData", callbackData)
		}
		return nil
	}

	// Route to appropriate handler based on callback data
	switch {
	case strings.HasPrefix(callbackData, config.CallbackPrefixCreateNewFolder):
		logutils.Info("HandleCallbackQuery: Routing to NewTopicCreationRequest", "chatID", chatID, "callbackData", callbackData)
		err = ch.TopicHandlers.HandleNewTopicCreationRequest(update, originalMsg)
	case strings.HasPrefix(callbackData, config.CallbackPrefixRetry):
		logutils.Info("HandleCallbackQuery: Routing to RetryCallback", "chatID", chatID, "callbackData", callbackData)
		err = ch.AIHandlers.HandleRetryCallback(update, originalMsg)
	case strings.HasPrefix(callbackData, config.CallbackPrefixShowAllTopics):
		logutils.Info("HandleCallbackQuery: Routing to ShowAllTopicsCallback", "chatID", chatID, "callbackData", callbackData)
		err = ch.TopicHandlers.HandleShowAllTopicsCallback(update, originalMsg)
	case callbackData == config.CallbackDataCreateTopicMenu:
		logutils.Info("HandleCallbackQuery: Routing to HandleCreateTopicMenuCallback", "chatID", chatID, "callbackData", callbackData)
		err = ch.TopicHandlers.HandleCreateTopicMenuCallback(update, originalMsg)
	case callbackData == config.CallbackDataShowAllTopicsMenu:
		logutils.Info("HandleCallbackQuery: Routing to HandleShowAllTopicsMenuCallback", "chatID", chatID, "callbackData", callbackData)
		err = ch.TopicHandlers.HandleShowAllTopicsMenuCallback(update, originalMsg)
	case strings.HasPrefix(callbackData, config.CallbackPrefixBackToSuggestions):
		logutils.Info("HandleCallbackQuery: Routing to HandleBackToSuggestionsCallback", "chatID", chatID, "callbackData", callbackData)
		err = ch.AIHandlers.HandleBackToSuggestionsCallback(update, originalMsg)
	default:
		logutils.Warn("HandleCallbackQuery: Routing to HandleTopicSelectionCallback", "chatID", chatID, "callbackData", callbackData)
		err = ch.TopicHandlers.HandleTopicSelectionCallback(update, originalMsg, callbackData)
	}
	if err != nil {
		logutils.Error("HandleCallbackQuery: HandlerError", err, "chatID", chatID, "callbackData", callbackData)
	} else {
		logutils.Success("HandleCallbackQuery", "chatID", chatID, "callbackData", callbackData)
	}

	return err
}

// IsRecentlyMovedMessage checks if message was recently moved
func (ch *CallbackHandlers) IsRecentlyMovedMessage(messageID int64) bool {
	return ch.TopicHandlers.IsRecentlyMovedMessage(messageID)
}

// MarkMessageAsMoved marks message as moved
func (ch *CallbackHandlers) MarkMessageAsMoved(messageID int64) {
	ch.TopicHandlers.MarkMessageAsMoved(messageID)
}

// CleanupMovedMessage cleans up moved message tracking
func (ch *CallbackHandlers) CleanupMovedMessage(messageID int64) {
	ch.TopicHandlers.CleanupMovedMessage(messageID)
}

// IsWaitingForTopicName checks if user is waiting for topic name
func (ch *CallbackHandlers) IsWaitingForTopicName(userID int64) bool {
	return ch.TopicHandlers.IsWaitingForTopicName(userID)
}

// HandleTopicNameEntry delegates to topic handlers
func (ch *CallbackHandlers) HandleTopicNameEntry(update *gotgbot.Update) error {
	return ch.TopicHandlers.HandleTopicNameEntry(update)
}
