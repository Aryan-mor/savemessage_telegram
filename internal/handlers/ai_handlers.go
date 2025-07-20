package handlers

import (
	"context"
	"strconv"
	"strings"

	"save-message/internal/config"
	"save-message/internal/interfaces"
	"save-message/internal/logutils"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// AIHandlers handles AI-related operations and suggestions
type AIHandlers struct {
	messageService       interfaces.MessageServiceInterface
	topicService         interfaces.TopicServiceInterface
	aiService            interfaces.AIServiceInterface
	messageStore         map[string]*gotgbot.Message
	keyboardMessageStore map[string]int
	keyboardBuilder      *KeyboardBuilder

	// Add reference to TopicHandlers for cross-storage
	TopicHandlers *TopicHandlers

	// Mockable funcs for testing
	HandleGeneralTopicMessageFunc       func(update *gotgbot.Update) error
	HandleRetryCallbackFunc             func(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleBackToSuggestionsCallbackFunc func(update *gotgbot.Update, originalMsg *gotgbot.Message) error
}

// NewAIHandlers creates a new AI handlers instance
func NewAIHandlers(messageService interfaces.MessageServiceInterface, topicService interfaces.TopicServiceInterface, aiService interfaces.AIServiceInterface, topicHandlers *TopicHandlers) *AIHandlers {
	return &AIHandlers{
		messageService:       messageService,
		topicService:         topicService,
		aiService:            aiService,
		messageStore:         make(map[string]*gotgbot.Message),
		keyboardMessageStore: make(map[string]int),
		keyboardBuilder:      NewKeyboardBuilder(),
		TopicHandlers:        topicHandlers,
	}
}

// HandleGeneralTopicMessage handles messages in General topic with AI suggestions
func (ah *AIHandlers) HandleGeneralTopicMessage(update *gotgbot.Update) error {
	if ah.HandleGeneralTopicMessageFunc != nil {
		return ah.HandleGeneralTopicMessageFunc(update)
	}
	logutils.Info("HandleGeneralTopicMessage", "chatID", update.Message.Chat.Id, "messageID", update.Message.MessageId)

	// Send waiting message
	waitingMsg, err := ah.messageService.SendMessage(update.Message.Chat.Id, config.AIProcessingMessage, &gotgbot.SendMessageOpts{
		MessageThreadId: update.Message.MessageThreadId,
	})
	if err != nil {
		logutils.Error("HandleGeneralTopicMessage: SendMessageError", err, "chatID", update.Message.Chat.Id)
		return err
	}

	// Store the waiting message ID
	callbackData := "suggestions_" + strconv.FormatInt(update.Message.MessageId, 10)
	ah.keyboardMessageStore[callbackData] = int(waitingMsg.MessageId)

	// Process AI suggestions in a goroutine
	go func(msg *gotgbot.Message) {
		// Get existing topics
		topics, err := ah.topicService.GetForumTopics(msg.Chat.Id)
		if err != nil {
			logutils.Error("HandleGeneralTopicMessage: GetForumTopicsError", err, "chatID", msg.Chat.Id)
			ah.handleAIError(msg, waitingMsg)
			return
		}

		// Get AI suggestions
		ctx := context.Background()
		suggestions, err := ah.aiService.SuggestFolders(ctx, msg.Text, ah.getTopicNames(topics))
		if err != nil {
			logutils.Error("HandleGeneralTopicMessage: SuggestFoldersError", err, "chatID", msg.Chat.Id)
			ah.handleAIError(msg, waitingMsg)
			return
		}

		logutils.Info("HandleGeneralTopicMessage: AI suggestions", "suggestions", suggestions)

		// Build keyboard
		keyboard, err := ah.keyboardBuilder.BuildSuggestionKeyboard(msg, suggestions, topics)
		if err != nil {
			logutils.Error("HandleGeneralTopicMessage: BuildSuggestionKeyboardError", err, "chatID", msg.Chat.Id)
			ah.handleAIError(msg, waitingMsg)
			return
		}

		// Store message references for all suggestion buttons
		ah.storeMessageReferences(msg, suggestions, topics)

		// Also store in TopicHandlers.messageStore for callback lookup
		if ah.TopicHandlers != nil {
			for _, folder := range suggestions {
				callbackData := strings.TrimSpace(folder) + "_" + strconv.FormatInt(msg.MessageId, 10)
				ah.TopicHandlers.MessageStore[callbackData] = msg
			}
			// Also store for create new topic button
			createCallbackData := config.CallbackPrefixCreateNewFolder + strconv.FormatInt(msg.MessageId, 10)
			ah.TopicHandlers.MessageStore[createCallbackData] = msg
		}

		// Update the waiting message with suggestions
		logutils.Info("HandleGeneralTopicMessage: Updating waiting message", "chatID", msg.Chat.Id, "messageID", waitingMsg.MessageId, "text", config.ChooseFolderMessage)
		_, err = ah.messageService.EditMessageText(msg.Chat.Id, int64(waitingMsg.MessageId), config.ChooseFolderMessage, &gotgbot.EditMessageTextOpts{
			ReplyMarkup: *keyboard,
		})
		if err != nil {
			logutils.Error("HandleGeneralTopicMessage: EditMessageTextError", err, "chatID", msg.Chat.Id, "messageID", waitingMsg.MessageId)
			// If update fails, try to find the message by searching through all stored keyboard messages
			ah.tryUpdateExistingMessage(msg, keyboard)
			// Only delete the waitingMsg if the edit failed (i.e., a new message will be sent)
			if waitingMsg != nil {
				logutils.Info("HandleGeneralTopicMessage: Attempting to delete 'Thinking...' message after edit failure", "chatID", msg.Chat.Id, "messageID", waitingMsg.MessageId, "text", waitingMsg.Text)
				err := ah.messageService.DeleteMessage(msg.Chat.Id, int(waitingMsg.MessageId))
				if err != nil {
					logutils.Error("HandleGeneralTopicMessage: Failed to delete 'Thinking...' message", err, "chatID", msg.Chat.Id, "messageID", waitingMsg.MessageId)
				} else {
					logutils.Success("HandleGeneralTopicMessage: Deleted 'Thinking...' message", "chatID", msg.Chat.Id, "messageID", waitingMsg.MessageId)
				}
			}
		} else {
			logutils.Success("HandleGeneralTopicMessage: Successfully updated waiting message with keyboard", "chatID", msg.Chat.Id)
			// Store keyboard message ID for all suggestion buttons
			ah.storeKeyboardMessageIDs(msg, suggestions, topics, int(waitingMsg.MessageId))
			// Do NOT delete the message if edit succeeded
		}
	}(update.Message)

	return nil
}

// HandleRetryCallback handles retry button clicks
func (ah *AIHandlers) HandleRetryCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	if ah.HandleRetryCallbackFunc != nil {
		return ah.HandleRetryCallbackFunc(update, originalMsg)
	}
	logutils.Info("HandleRetryCallback", "chatID", originalMsg.Chat.Id)

	_, err := ah.messageService.SendMessage(originalMsg.Chat.Id, config.SuccessMessageRetry, &gotgbot.SendMessageOpts{
		MessageThreadId: originalMsg.MessageThreadId,
	})
	if err != nil {
		logutils.Error("HandleRetryCallback: SendMessageError", err, "chatID", originalMsg.Chat.Id)
		return err
	}

	logutils.Success("HandleRetryCallback", "chatID", originalMsg.Chat.Id)
	return nil
}

// HandleBackToSuggestionsCallback handles back to suggestions button
func (ah *AIHandlers) HandleBackToSuggestionsCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	if ah.HandleBackToSuggestionsCallbackFunc != nil {
		return ah.HandleBackToSuggestionsCallbackFunc(update, originalMsg)
	}
	logutils.Info("HandleBackToSuggestionsCallback", "chatID", originalMsg.Chat.Id)

	// Get existing topics
	topics, err := ah.topicService.GetForumTopics(originalMsg.Chat.Id)
	if err != nil {
		logutils.Error("HandleBackToSuggestionsCallback: GetForumTopicsError", err, "chatID", originalMsg.Chat.Id)
		_, sendErr := ah.messageService.SendMessage(originalMsg.Chat.Id, config.ErrorMessageFailed, &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
		})
		if sendErr != nil {
			logutils.Error("HandleBackToSuggestionsCallback: SendMessageError", sendErr, "chatID", originalMsg.Chat.Id)
		}
		return err
	}

	// Get AI suggestions again
	ctx := context.Background()
	suggestions, err := ah.aiService.SuggestFolders(ctx, originalMsg.Text, ah.getTopicNames(topics))
	if err != nil {
		logutils.Error("HandleBackToSuggestionsCallback: SuggestFoldersError", err, "chatID", originalMsg.Chat.Id)
		ah.handleAIError(originalMsg, nil)
		return err
	}

	// Build keyboard
	keyboard, err := ah.keyboardBuilder.BuildSuggestionKeyboard(originalMsg, suggestions, topics)
	if err != nil {
		logutils.Error("HandleBackToSuggestionsCallback: BuildSuggestionKeyboardError", err, "chatID", originalMsg.Chat.Id)
		return err
	}

	// Store message references for all suggestion buttons
	ah.storeMessageReferences(originalMsg, suggestions, topics)

	// Try to update existing message or send new one
	callbackData := "suggestions_" + strconv.FormatInt(originalMsg.MessageId, 10)
	if keyboardMsgId, exists := ah.keyboardMessageStore[callbackData]; exists {
		_, err = ah.messageService.EditMessageText(originalMsg.Chat.Id, int64(keyboardMsgId), config.ChooseFolderMessage, &gotgbot.EditMessageTextOpts{
			ReplyMarkup: *keyboard,
		})
		if err != nil {
			logutils.Error("HandleBackToSuggestionsCallback: EditMessageTextError", err, "chatID", originalMsg.Chat.Id, "messageID", keyboardMsgId)
			// If update fails, send new message
			newMsg, err := ah.messageService.SendMessage(originalMsg.Chat.Id, config.ChooseFolderMessage, &gotgbot.SendMessageOpts{
				MessageThreadId: originalMsg.MessageThreadId,
				ReplyMarkup:     *keyboard,
			})
			if err != nil {
				logutils.Error("HandleBackToSuggestionsCallback: SendMessageError", err, "chatID", originalMsg.Chat.Id)
			} else {
				ah.storeKeyboardMessageIDs(originalMsg, suggestions, topics, int(newMsg.MessageId))
			}
		} else {
			ah.storeKeyboardMessageIDs(originalMsg, suggestions, topics, keyboardMsgId)
		}
	} else {
		// Send new message with suggestions
		newMsg, err := ah.messageService.SendMessage(originalMsg.Chat.Id, config.ChooseFolderMessage, &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
			ReplyMarkup:     *keyboard,
		})
		if err != nil {
			logutils.Error("HandleBackToSuggestionsCallback: SendMessageError", err, "chatID", originalMsg.Chat.Id)
		} else {
			ah.storeKeyboardMessageIDs(originalMsg, suggestions, topics, int(newMsg.MessageId))
		}
	}

	logutils.Success("HandleBackToSuggestionsCallback", "chatID", originalMsg.Chat.Id)
	return nil
}

// Helper methods
func (ah *AIHandlers) getTopicNames(topics []interfaces.ForumTopic) []string {
	var names []string
	for _, topic := range topics {
		names = append(names, topic.Name)
	}
	return names
}

func (ah *AIHandlers) handleAIError(msg *gotgbot.Message, waitingMsg *gotgbot.Message) {
	retryKeyboard := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextTryAgain, CallbackData: config.CallbackPrefixRetry + strconv.FormatInt(msg.MessageId, 10)}},
		},
	}

	if waitingMsg != nil {
		_, err := ah.messageService.EditMessageText(msg.Chat.Id, waitingMsg.MessageId, config.AIFailedMessage, &gotgbot.EditMessageTextOpts{
			ReplyMarkup: *retryKeyboard,
		})
		if err != nil {
			logutils.Error("handleAIError: EditMessageTextError", err, "chatID", msg.Chat.Id, "messageID", waitingMsg.MessageId)
		}
	}
}

func (ah *AIHandlers) storeMessageReferences(msg *gotgbot.Message, suggestions []string, topics []interfaces.ForumTopic) {
	// Store for existing topics
	for _, folder := range suggestions {
		for _, topic := range topics {
			if strings.EqualFold(topic.Name, folder) {
				callbackData := topic.Name + "_" + strconv.FormatInt(msg.MessageId, 10)
				ah.messageStore[callbackData] = msg
				break
			}
		}
	}

	// Store for new topics
	for _, folder := range suggestions {
		cleanFolder := strings.TrimSpace(folder)
		if len(cleanFolder) > 0 && len(cleanFolder) <= 50 && !strings.Contains(cleanFolder, "\n") {
			callbackData := cleanFolder + "_" + strconv.FormatInt(msg.MessageId, 10)
			ah.messageStore[callbackData] = msg
		}
	}

	// Store for other buttons
	createCallbackData := config.CallbackPrefixCreateNewFolder + strconv.FormatInt(msg.MessageId, 10)
	ah.messageStore[createCallbackData] = msg

	if len(topics) > 0 {
		showAllCallbackData := config.CallbackPrefixShowAllTopics + strconv.FormatInt(msg.MessageId, 10)
		ah.messageStore[showAllCallbackData] = msg
	}

	retryCallbackData := config.CallbackPrefixRetry + strconv.FormatInt(msg.MessageId, 10)
	ah.messageStore[retryCallbackData] = msg
}

func (ah *AIHandlers) storeKeyboardMessageIDs(msg *gotgbot.Message, suggestions []string, topics []interfaces.ForumTopic, keyboardMsgID int) {
	// Store for existing topics
	for _, folder := range suggestions {
		for _, topic := range topics {
			if strings.EqualFold(topic.Name, folder) {
				callbackData := topic.Name + "_" + strconv.FormatInt(msg.MessageId, 10)
				ah.keyboardMessageStore[callbackData] = keyboardMsgID
				break
			}
		}
	}

	// Store for new topics
	for _, folder := range suggestions {
		cleanFolder := strings.TrimSpace(folder)
		if len(cleanFolder) > 0 && len(cleanFolder) <= 50 && !strings.Contains(cleanFolder, "\n") {
			callbackData := cleanFolder + "_" + strconv.FormatInt(msg.MessageId, 10)
			ah.keyboardMessageStore[callbackData] = keyboardMsgID
		}
	}

	// Store for other buttons
	createCallbackData := config.CallbackPrefixCreateNewFolder + strconv.FormatInt(msg.MessageId, 10)
	ah.keyboardMessageStore[createCallbackData] = keyboardMsgID

	if len(topics) > 0 {
		showAllCallbackData := config.CallbackPrefixShowAllTopics + strconv.FormatInt(msg.MessageId, 10)
		ah.keyboardMessageStore[showAllCallbackData] = keyboardMsgID
	}

	retryCallbackData := config.CallbackPrefixRetry + strconv.FormatInt(msg.MessageId, 10)
	ah.keyboardMessageStore[retryCallbackData] = keyboardMsgID
}

func (ah *AIHandlers) tryUpdateExistingMessage(msg *gotgbot.Message, keyboard *gotgbot.InlineKeyboardMarkup) {
	for storedCallback, storedMsgID := range ah.keyboardMessageStore {
		if strings.Contains(storedCallback, strconv.FormatInt(msg.MessageId, 10)) {
			_, updateErr := ah.messageService.EditMessageText(msg.Chat.Id, int64(storedMsgID), config.ChooseFolderMessage, &gotgbot.EditMessageTextOpts{
				ReplyMarkup: *keyboard,
			})
			if updateErr == nil {
				break
			}
		}
	}
}
