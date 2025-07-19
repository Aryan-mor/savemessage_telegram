package handlers

import (
	"log"
	"strings"

	"save-message/internal/config"
	"save-message/internal/services"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// CallbackHandlers coordinates all callback-related interactions
type CallbackHandlers struct {
	topicHandlers   *TopicHandlers
	warningHandlers *WarningHandlers
	aiHandlers      *AIHandlers
	messageService  *services.MessageService
}

// TopicCreationContext stores context for topic creation
type TopicCreationContext struct {
	ChatId        int64
	ThreadId      int64
	OriginalMsgId int64
}

// NewCallbackHandlers creates a new callback handlers instance
func NewCallbackHandlers(messageService *services.MessageService, topicService *services.TopicService, aiService *services.AIService) *CallbackHandlers {
	return &CallbackHandlers{
		topicHandlers:   NewTopicHandlers(messageService, topicService),
		warningHandlers: NewWarningHandlers(messageService),
		aiHandlers:      NewAIHandlers(messageService, topicService, aiService),
		messageService:  messageService,
	}
}

// HandleCallbackQuery routes callback queries to appropriate handlers
func (ch *CallbackHandlers) HandleCallbackQuery(update *gotgbot.Update) error {
	log.Printf("[CallbackHandlers] Handling callback query: %s", update.CallbackQuery.Data)

	callbackData := update.CallbackQuery.Data

	// Answer the callback query to remove the loading state
	err := ch.messageService.AnswerCallbackQuery(update.CallbackQuery.Id, &gotgbot.AnswerCallbackQueryOpts{
		Text: "Processing...",
	})
	if err != nil {
		log.Printf("[CallbackHandlers] Error answering callback query: %v", err)
	}

	// Special handling for warning callbacks
	if ch.warningHandlers.IsWarningCallback(callbackData) {
		return ch.warningHandlers.HandleWarningOkCallback(update)
	}

	// Get original message from topic handlers
	originalMsg := ch.topicHandlers.messageStore[callbackData]
	if originalMsg == nil {
		_, err := ch.messageService.SendMessage(update.CallbackQuery.From.Id, config.ErrorMessageNotFound, nil)
		if err != nil {
			log.Printf("[CallbackHandlers] Error sending error message: %v", err)
		}
		return nil
	}

	// Route to appropriate handler based on callback data
	switch {
	case strings.HasPrefix(callbackData, config.CallbackPrefixCreateNewFolder):
		return ch.topicHandlers.HandleNewTopicCreationRequest(update, originalMsg)
	case strings.HasPrefix(callbackData, config.CallbackPrefixRetry):
		return ch.aiHandlers.HandleRetryCallback(update, originalMsg)
	case strings.HasPrefix(callbackData, config.CallbackPrefixShowAllTopics):
		return ch.topicHandlers.HandleShowAllTopicsCallback(update, originalMsg)
	case callbackData == config.CallbackDataCreateTopicMenu:
		return ch.topicHandlers.HandleCreateTopicMenuCallback(update, originalMsg)
	case callbackData == config.CallbackDataShowAllTopicsMenu:
		return ch.topicHandlers.HandleShowAllTopicsMenuCallback(update, originalMsg)
	case strings.HasPrefix(callbackData, config.CallbackPrefixBackToSuggestions):
		return ch.aiHandlers.HandleBackToSuggestionsCallback(update, originalMsg)
	default:
		return ch.topicHandlers.HandleTopicSelectionCallback(update, originalMsg, callbackData)
	}
}

// IsRecentlyMovedMessage checks if message was recently moved
func (ch *CallbackHandlers) IsRecentlyMovedMessage(messageID int64) bool {
	return ch.topicHandlers.IsRecentlyMovedMessage(messageID)
}

// MarkMessageAsMoved marks message as moved
func (ch *CallbackHandlers) MarkMessageAsMoved(messageID int64) {
	ch.topicHandlers.MarkMessageAsMoved(messageID)
}

// CleanupMovedMessage cleans up moved message tracking
func (ch *CallbackHandlers) CleanupMovedMessage(messageID int64) {
	ch.topicHandlers.CleanupMovedMessage(messageID)
}

// IsWaitingForTopicName checks if user is waiting for topic name
func (ch *CallbackHandlers) IsWaitingForTopicName(userID int64) bool {
	return ch.topicHandlers.WaitingForTopicName[userID].ChatId != 0
}

// HandleTopicNameEntry delegates to topic handlers
func (ch *CallbackHandlers) HandleTopicNameEntry(update *gotgbot.Update) error {
	return ch.topicHandlers.HandleTopicNameEntry(update)
}
