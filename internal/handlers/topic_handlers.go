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

// TopicHandlers handles topic-related operations and callbacks
type TopicHandlers struct {
	messageService        interfaces.MessageServiceInterface
	topicService          interfaces.TopicServiceInterface
	MessageStore          map[string]*gotgbot.Message
	KeyboardMessageStore  map[string]int
	WaitingForTopicName   map[int64]TopicCreationContext
	OriginalMessageStore  map[int64]*gotgbot.Message
	RecentlyMovedMessages map[int64]bool
	keyboardBuilder       *KeyboardBuilder

	// For testability: allow configurable delays
	MessageAutoDeleteDelay  time.Duration
	ConfirmationDeleteDelay time.Duration

	// Mockable funcs for testing
	HandleNewTopicCreationRequestFunc   func(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleTopicSelectionCallbackFunc    func(update *gotgbot.Update, originalMsg *gotgbot.Message, callbackData string) error
	HandleShowAllTopicsCallbackFunc     func(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleCreateTopicMenuCallbackFunc   func(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleShowAllTopicsMenuCallbackFunc func(update *gotgbot.Update, originalMsg *gotgbot.Message) error
	HandleTopicNameEntryFunc            func(update *gotgbot.Update) error
}

// TopicCreationContext is already defined in callback_handlers.go

// NewTopicHandlers creates a new topic handlers instance
func NewTopicHandlers(messageService interfaces.MessageServiceInterface, topicService interfaces.TopicServiceInterface) *TopicHandlers {
	return &TopicHandlers{
		messageService:        messageService,
		topicService:          topicService,
		MessageStore:          make(map[string]*gotgbot.Message),
		KeyboardMessageStore:  make(map[string]int),
		WaitingForTopicName:   make(map[int64]TopicCreationContext),
		OriginalMessageStore:  make(map[int64]*gotgbot.Message),
		RecentlyMovedMessages: make(map[int64]bool),
		keyboardBuilder:       NewKeyboardBuilder(),
	}
}

// HandleNewTopicCreationRequest handles requests to create a new topic
func (th *TopicHandlers) HandleNewTopicCreationRequest(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	if th.HandleNewTopicCreationRequestFunc != nil {
		return th.HandleNewTopicCreationRequestFunc(update, originalMsg)
	}
	logutils.Info("HandleNewTopicCreationRequest", "chatID", originalMsg.Chat.Id)

	// Ask user for topic name
	_, err := th.messageService.SendMessage(originalMsg.Chat.Id, config.TopicNamePrompt, &gotgbot.SendMessageOpts{
		MessageThreadId: originalMsg.MessageThreadId,
	})
	if err != nil {
		logutils.Error("HandleNewTopicCreationRequest: SendMessageError", err, "chatID", originalMsg.Chat.Id)
		return err
	}

	// Store the context for topic creation
	th.WaitingForTopicName[update.CallbackQuery.From.Id] = TopicCreationContext{
		ChatId:        originalMsg.Chat.Id,
		ThreadId:      int64(originalMsg.MessageThreadId),
		OriginalMsgId: int64(originalMsg.MessageId),
	}

	// Store the original message for this user
	th.OriginalMessageStore[update.CallbackQuery.From.Id] = originalMsg

	// Delete the keyboard message
	if keyboardMsgId, exists := th.KeyboardMessageStore[update.CallbackQuery.Data]; exists {
		th.messageService.DeleteMessage(originalMsg.Chat.Id, keyboardMsgId)
		delete(th.KeyboardMessageStore, update.CallbackQuery.Data)
	}

	logutils.Success("HandleNewTopicCreationRequest", "chatID", originalMsg.Chat.Id)
	return nil
}

// HandleTopicNameEntry handles when user provides a topic name
func (th *TopicHandlers) HandleTopicNameEntry(update *gotgbot.Update) error {
	if th.HandleTopicNameEntryFunc != nil {
		return th.HandleTopicNameEntryFunc(update)
	}
	logutils.Info("HandleTopicNameEntry", "userID", update.Message.From.Id)

	ctx := th.WaitingForTopicName[update.Message.From.Id]
	topicName := strings.TrimSpace(update.Message.Text)

	if topicName == "" {
		_, err := th.messageService.SendMessage(ctx.ChatId, config.TopicNameEmptyError, &gotgbot.SendMessageOpts{})
		if err != nil {
			logutils.Error("HandleTopicNameEntry: SendMessageError", err, "chatID", ctx.ChatId)
		}
		return nil
	}

	// Check if topic already exists
	topics, err := th.topicService.GetForumTopics(ctx.ChatId)
	if err == nil {
		exists := false
		for _, topic := range topics {
			if strings.EqualFold(topic.Name, topicName) {
				exists = true
				break
			}
		}
		if exists {
			_, err = th.messageService.SendMessage(ctx.ChatId, config.TopicNameExistsError, &gotgbot.SendMessageOpts{})
			if err != nil {
				logutils.Error("HandleTopicNameEntry: SendMessageError", err, "chatID", ctx.ChatId)
			}
			th.cleanupTopicCreation(update.Message.From.Id)
			return nil
		}
	}

	// Create the topic
	threadID, err := th.topicService.CreateForumTopic(ctx.ChatId, topicName)
	if err != nil {
		logutils.Error("HandleTopicNameEntry: CreateTopicError", err, "chatID", ctx.ChatId)
		_, sendErr := th.messageService.SendMessage(ctx.ChatId, config.ErrorMessageCreateFailed, &gotgbot.SendMessageOpts{})
		if sendErr != nil {
			logutils.Error("HandleTopicNameEntry: SendMessageError", sendErr, "chatID", ctx.ChatId)
		}
		th.cleanupTopicCreation(update.Message.From.Id)
		return err
	}

	// Send topic name as first message in new topic
	if threadID != 0 {
		_, err := th.messageService.SendMessage(ctx.ChatId, topicName, &gotgbot.SendMessageOpts{
			MessageThreadId: threadID,
		})
		if err != nil {
			logutils.Error("HandleTopicNameEntry: SendMessageError", err, "chatID", ctx.ChatId)
		}
	}

	// Copy the original user message to the new topic
	if origMsg, ok := th.OriginalMessageStore[update.Message.From.Id]; ok && threadID != 0 {
		_, err := th.messageService.CopyMessageToTopicWithResult(ctx.ChatId, origMsg.Chat.Id, int(origMsg.MessageId), int(threadID))
		if err != nil {
			logutils.Error("HandleTopicNameEntry: CopyMessageError", err, "chatID", ctx.ChatId)
		} else {
			// Build preview: first 2 lines of the original message
			previewLines := strings.SplitN(origMsg.Text, "\n", 3)
			preview := ""
			if len(previewLines) > 0 {
				preview += "\n\"" + previewLines[0] + "\""
			}
			if len(previewLines) > 1 {
				preview += "\n\"" + previewLines[1] + "\""
			}
			confirmMsg := config.SuccessMessageSaved + topicName + preview

			// Send confirmation message to General
			_, err = th.messageService.SendMessage(ctx.ChatId, confirmMsg, &gotgbot.SendMessageOpts{
				MessageThreadId: 0,
			})
			if err != nil {
				logutils.Error("HandleTopicNameEntry: SendMessageError", err, "chatID", ctx.ChatId)
			}

			// Delete the original message from General after a short delay
			go func(chatID int64, messageID int) {
				delay := th.MessageAutoDeleteDelay
				if delay == 0 {
					delay = config.DefaultMessageAutoDeleteDelay
				}
				time.Sleep(delay)
				_ = th.messageService.DeleteMessage(chatID, messageID)
			}(origMsg.Chat.Id, int(origMsg.MessageId))
		}
	}

	// Clean up state
	th.cleanupTopicCreation(update.Message.From.Id)
	return nil
}

// HandleTopicSelectionCallback handles when user selects an existing topic
func (th *TopicHandlers) HandleTopicSelectionCallback(update *gotgbot.Update, originalMsg *gotgbot.Message, callbackData string) error {
	if th.HandleTopicSelectionCallbackFunc != nil {
		return th.HandleTopicSelectionCallbackFunc(update, originalMsg, callbackData)
	}
	logutils.Info("HandleTopicSelectionCallback", "callbackData", callbackData)

	// Extract topic name from callback data
	parts := strings.Split(callbackData, "_")
	if len(parts) < 2 {
		logutils.Warn("HandleTopicSelectionCallback: InvalidCallbackData", "callbackData", callbackData)
		return nil
	}

	topicName := strings.Join(parts[:len(parts)-1], "_") // Rejoin in case topic name contains underscores

	// Find the topic
	threadID, err := th.topicService.FindTopicByName(originalMsg.Chat.Id, topicName)
	if err != nil {
		// If topic not found, try to create it (AI suggestion case)
		if strings.Contains(err.Error(), "topic not found") {
			logutils.Warn("HandleTopicSelectionCallback: Topic not found, creating new topic", "chatID", originalMsg.Chat.Id, "topicName", topicName)
			threadID, err = th.topicService.CreateForumTopic(originalMsg.Chat.Id, topicName)
			if err != nil || threadID == 0 {
				logutils.Error("HandleTopicSelectionCallback: CreateTopicError", err, "chatID", originalMsg.Chat.Id, "topicName", topicName)
				_, sendErr := th.messageService.SendMessage(originalMsg.Chat.Id, config.ErrorMessageCreateFailed, &gotgbot.SendMessageOpts{
					MessageThreadId: originalMsg.MessageThreadId,
				})
				if sendErr != nil {
					logutils.Error("HandleTopicSelectionCallback: SendMessageError", sendErr, "chatID", originalMsg.Chat.Id)
				}
				return err
			}
			// Optionally, send topic name as first message in new topic (like in HandleTopicNameEntry)
			_, _ = th.messageService.SendMessage(originalMsg.Chat.Id, topicName, &gotgbot.SendMessageOpts{
				MessageThreadId: threadID,
			})
		} else {
			logutils.Error("HandleTopicSelectionCallback: FindTopicError", err, "chatID", originalMsg.Chat.Id)
			_, sendErr := th.messageService.SendMessage(originalMsg.Chat.Id, config.ErrorMessageNotFound, &gotgbot.SendMessageOpts{
				MessageThreadId: originalMsg.MessageThreadId,
			})
			if sendErr != nil {
				logutils.Error("HandleTopicSelectionCallback: SendMessageError", sendErr, "chatID", originalMsg.Chat.Id)
			}
			return err
		}
	}

	// Copy message to the selected (or newly created) topic
	_, err = th.messageService.CopyMessageToTopicWithResult(originalMsg.Chat.Id, originalMsg.Chat.Id, int(originalMsg.MessageId), int(threadID))
	if err != nil {
		logutils.Error("HandleTopicSelectionCallback: CopyMessageError", err, "chatID", originalMsg.Chat.Id)
		_, sendErr := th.messageService.SendMessage(originalMsg.Chat.Id, "❌ Failed to save message to topic.", &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
		})
		if sendErr != nil {
			logutils.Error("HandleTopicSelectionCallback: SendMessageError", sendErr, "chatID", originalMsg.Chat.Id)
		}
		return err
	}

	// Mark message as moved
	th.MarkMessageAsMoved(originalMsg.MessageId)

	// Build preview: first 2 lines of the original message
	previewLines := strings.SplitN(originalMsg.Text, "\n", 3)
	preview := ""
	if len(previewLines) > 0 {
		preview += "\n\"" + previewLines[0] + "\""
	}
	if len(previewLines) > 1 {
		preview += "\n\"" + previewLines[1] + "\""
	}
	confirmMsg := config.SuccessMessageSaved + topicName + preview

	// Send confirmation message
	confirmMsgObj, err := th.messageService.SendMessage(originalMsg.Chat.Id, confirmMsg, &gotgbot.SendMessageOpts{
		MessageThreadId: originalMsg.MessageThreadId,
	})
	if err != nil {
		logutils.Error("HandleTopicSelectionCallback: SendMessageError", err, "chatID", originalMsg.Chat.Id)
		return err
	}

	// Delete the 'Choose a folder:' message (the keyboard message)
	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		_ = th.messageService.DeleteMessage(update.CallbackQuery.Message.Chat.Id, int(update.CallbackQuery.Message.MessageId))
	}

	// Delete the original message after a short delay
	go func(chatID int64, messageID int) {
		delay := th.MessageAutoDeleteDelay
		if delay == 0 {
			delay = config.DefaultMessageAutoDeleteDelay
		}
		time.Sleep(delay)
		_ = th.messageService.DeleteMessage(chatID, messageID)
	}(originalMsg.Chat.Id, int(originalMsg.MessageId))

	// Delete the confirmation message after 1 minute
	go func(chatID int64, messageID int) {
		delay := th.ConfirmationDeleteDelay
		if delay == 0 {
			delay = time.Minute
		}
		time.Sleep(delay)
		_ = th.messageService.DeleteMessage(chatID, messageID)
	}(confirmMsgObj.Chat.Id, int(confirmMsgObj.MessageId))

	logutils.Success("HandleTopicSelectionCallback", "topicName", topicName, "chatID", originalMsg.Chat.Id)
	return nil
}

// HandleShowAllTopicsCallback handles showing all topics from suggestions
func (th *TopicHandlers) HandleShowAllTopicsCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	if th.HandleShowAllTopicsCallbackFunc != nil {
		return th.HandleShowAllTopicsCallbackFunc(update, originalMsg)
	}
	logutils.Info("HandleShowAllTopicsCallback", "chatID", originalMsg.Chat.Id)

	topics, err := th.topicService.GetForumTopics(originalMsg.Chat.Id)
	if err != nil {
		logutils.Error("HandleShowAllTopicsCallback: GetTopicsError", err, "chatID", originalMsg.Chat.Id)
		_, sendErr := th.messageService.SendMessage(originalMsg.Chat.Id, config.ErrorMessageFailed, &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
		})
		if sendErr != nil {
			logutils.Error("HandleShowAllTopicsCallback: SendMessageError", sendErr, "chatID", originalMsg.Chat.Id)
		}
		return err
	}

	if len(topics) == 0 {
		_, err = th.messageService.SendMessage(originalMsg.Chat.Id, config.NoTopicsDiscoveredMessage, &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
		})
		if err != nil {
			logutils.Error("HandleShowAllTopicsCallback: SendMessageError", err, "chatID", originalMsg.Chat.Id)
			return err
		}
	} else {
		// Build keyboard with all existing topics
		keyboard, err := th.keyboardBuilder.BuildAllTopicsKeyboard(originalMsg, topics)
		if err != nil {
			logutils.Error("HandleShowAllTopicsCallback: BuildKeyboardError", err, "chatID", originalMsg.Chat.Id)
			return err
		}

		// Store message references for all topic buttons
		for _, topic := range topics {
			topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
			th.MessageStore[topicCallbackData] = originalMsg
		}

		backCallbackData := config.CallbackPrefixBackToSuggestions + strconv.FormatInt(originalMsg.MessageId, 10)
		th.MessageStore[backCallbackData] = originalMsg

		// Try to update existing message or send new one
		callbackData := "suggestions_" + strconv.FormatInt(originalMsg.MessageId, 10)
		if keyboardMsgId, exists := th.KeyboardMessageStore[callbackData]; exists {
			_, err = th.messageService.EditMessageText(originalMsg.Chat.Id, int64(keyboardMsgId), config.ChooseFromAllTopicsMessage, &gotgbot.EditMessageTextOpts{
				ReplyMarkup: *keyboard,
			})
			if err != nil {
				logutils.Error("HandleShowAllTopicsCallback: EditMessageTextError", err, "chatID", originalMsg.Chat.Id)
				// If update fails, send new message
				newMsg, err := th.messageService.SendMessage(originalMsg.Chat.Id, config.ChooseFromAllTopicsMessage, &gotgbot.SendMessageOpts{
					MessageThreadId: originalMsg.MessageThreadId,
					ReplyMarkup:     *keyboard,
				})
				if err != nil {
					logutils.Error("HandleShowAllTopicsCallback: SendMessageError", err, "chatID", originalMsg.Chat.Id)
				} else {
					// Store keyboard message ID for all topic buttons
					for _, topic := range topics {
						topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
						th.KeyboardMessageStore[topicCallbackData] = int(newMsg.MessageId)
					}
					th.KeyboardMessageStore[backCallbackData] = int(newMsg.MessageId)
				}
			} else {
				// Store keyboard message ID for all topic buttons
				for _, topic := range topics {
					topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
					th.KeyboardMessageStore[topicCallbackData] = keyboardMsgId
				}
				th.KeyboardMessageStore[backCallbackData] = keyboardMsgId
			}
		} else {
			// Send new message with all topics
			newMsg, err := th.messageService.SendMessage(originalMsg.Chat.Id, config.ChooseFromAllTopicsMessage, &gotgbot.SendMessageOpts{
				MessageThreadId: originalMsg.MessageThreadId,
				ReplyMarkup:     *keyboard,
			})
			if err != nil {
				logutils.Error("HandleShowAllTopicsCallback: SendMessageError", err, "chatID", originalMsg.Chat.Id)
			} else {
				// Store keyboard message ID for all topic buttons
				for _, topic := range topics {
					topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
					th.KeyboardMessageStore[topicCallbackData] = int(newMsg.MessageId)
				}
				th.KeyboardMessageStore[backCallbackData] = int(newMsg.MessageId)
			}
		}
	}

	logutils.Success("HandleShowAllTopicsCallback", "chatID", originalMsg.Chat.Id)
	return nil
}

// HandleCreateTopicMenuCallback handles the create topic menu callback
func (th *TopicHandlers) HandleCreateTopicMenuCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	if th.HandleCreateTopicMenuCallbackFunc != nil {
		return th.HandleCreateTopicMenuCallbackFunc(update, originalMsg)
	}
	logutils.Info("HandleCreateTopicMenuCallback", "chatID", originalMsg.Chat.Id)

	_, err := th.messageService.SendMessage(originalMsg.Chat.Id, config.TopicCreationMenuMessage, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})
	if err != nil {
		logutils.Error("HandleCreateTopicMenuCallback: SendMessageError", err, "chatID", originalMsg.Chat.Id)
		return err
	}

	// Set flag to wait for topic name
	th.WaitingForTopicName[originalMsg.From.Id] = TopicCreationContext{
		ChatId:        originalMsg.Chat.Id,
		ThreadId:      int64(originalMsg.MessageThreadId),
		OriginalMsgId: int64(originalMsg.MessageId),
	}

	logutils.Success("HandleCreateTopicMenuCallback", "chatID", originalMsg.Chat.Id)
	return nil
}

// HandleShowAllTopicsMenuCallback handles the show all topics menu callback
func (th *TopicHandlers) HandleShowAllTopicsMenuCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	if th.HandleShowAllTopicsMenuCallbackFunc != nil {
		return th.HandleShowAllTopicsMenuCallbackFunc(update, originalMsg)
	}
	logutils.Info("HandleShowAllTopicsMenuCallback", "chatID", originalMsg.Chat.Id)

	topics, err := th.topicService.GetForumTopics(originalMsg.Chat.Id)
	if err != nil {
		logutils.Error("HandleShowAllTopicsMenuCallback: GetTopicsError", err, "chatID", originalMsg.Chat.Id)
		_, sendErr := th.messageService.SendMessage(originalMsg.Chat.Id, config.ErrorMessageFailed, &gotgbot.SendMessageOpts{})
		if sendErr != nil {
			logutils.Error("HandleShowAllTopicsMenuCallback: SendMessageError", sendErr, "chatID", originalMsg.Chat.Id)
		}
		return err
	}

	if len(topics) == 0 {
		_, err = th.messageService.SendMessage(originalMsg.Chat.Id, config.ErrorMessageNoTopics, &gotgbot.SendMessageOpts{})
		if err != nil {
			logutils.Error("HandleShowAllTopicsMenuCallback: SendMessageError", err, "chatID", originalMsg.Chat.Id)
			return err
		}
	} else {
		topicList := config.TopicsListHeader
		for _, topic := range topics {
			topicList += "• " + topic.Name + "\n"
		}
		_, err = th.messageService.SendMessage(originalMsg.Chat.Id, topicList, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
		if err != nil {
			logutils.Error("HandleShowAllTopicsMenuCallback: SendMessageError", err, "chatID", originalMsg.Chat.Id)
			return err
		}
	}

	logutils.Success("HandleShowAllTopicsMenuCallback", "chatID", originalMsg.Chat.Id)
	return nil
}

// GetMessageByCallbackData retrieves the original message associated with a callback data.
func (th *TopicHandlers) GetMessageByCallbackData(callbackData string) *gotgbot.Message {
	return th.MessageStore[callbackData]
}

// IsWaitingForTopicName checks if a user is in the process of creating a new topic.
func (th *TopicHandlers) IsWaitingForTopicName(userID int64) bool {
	_, exists := th.WaitingForTopicName[userID]
	return exists
}

// Helper methods
func (th *TopicHandlers) cleanupTopicCreation(userID int64) {
	delete(th.WaitingForTopicName, userID)
	delete(th.OriginalMessageStore, userID)
}

func (th *TopicHandlers) IsRecentlyMovedMessage(messageID int64) bool {
	return th.RecentlyMovedMessages[messageID]
}

func (th *TopicHandlers) MarkMessageAsMoved(messageID int64) {
	th.RecentlyMovedMessages[messageID] = true
}

func (th *TopicHandlers) CleanupMovedMessage(messageID int64) {
	delete(th.RecentlyMovedMessages, messageID)
}
