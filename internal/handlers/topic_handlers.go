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

// TopicHandlers handles topic-related operations and callbacks
type TopicHandlers struct {
	messageService        *services.MessageService
	topicService          *services.TopicService
	messageStore          map[string]*gotgbot.Message
	keyboardMessageStore  map[string]int
	WaitingForTopicName   map[int64]TopicCreationContext
	originalMessageStore  map[int64]*gotgbot.Message
	recentlyMovedMessages map[int64]bool
	keyboardBuilder       *KeyboardBuilder
}

// TopicCreationContext is already defined in callback_handlers.go

// NewTopicHandlers creates a new topic handlers instance
func NewTopicHandlers(messageService *services.MessageService, topicService *services.TopicService) *TopicHandlers {
	return &TopicHandlers{
		messageService:        messageService,
		topicService:          topicService,
		messageStore:          make(map[string]*gotgbot.Message),
		keyboardMessageStore:  make(map[string]int),
		WaitingForTopicName:   make(map[int64]TopicCreationContext),
		originalMessageStore:  make(map[int64]*gotgbot.Message),
		recentlyMovedMessages: make(map[int64]bool),
		keyboardBuilder:       NewKeyboardBuilder(),
	}
}

// HandleNewTopicCreationRequest handles requests to create a new topic
func (th *TopicHandlers) HandleNewTopicCreationRequest(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	log.Printf("[TopicHandlers] Handling new topic creation request: ChatID=%d", originalMsg.Chat.Id)

	// Ask user for topic name
	_, err := th.messageService.SendMessage(originalMsg.Chat.Id, config.TopicNamePrompt, &gotgbot.SendMessageOpts{
		MessageThreadId: originalMsg.MessageThreadId,
	})
	if err != nil {
		log.Printf("[TopicHandlers] Error sending topic name prompt: %v", err)
		return err
	}

	// Store the context for topic creation
	th.WaitingForTopicName[update.CallbackQuery.From.Id] = TopicCreationContext{
		ChatId:        originalMsg.Chat.Id,
		ThreadId:      int64(originalMsg.MessageThreadId),
		OriginalMsgId: int64(originalMsg.MessageId),
	}

	// Store the original message for this user
	th.originalMessageStore[update.CallbackQuery.From.Id] = originalMsg

	// Delete the keyboard message
	if keyboardMsgId, exists := th.keyboardMessageStore[update.CallbackQuery.Data]; exists {
		th.messageService.DeleteMessage(originalMsg.Chat.Id, keyboardMsgId)
		delete(th.keyboardMessageStore, update.CallbackQuery.Data)
	}

	log.Printf("[TopicHandlers] Successfully handled new topic creation request: ChatID=%d", originalMsg.Chat.Id)
	return nil
}

// HandleTopicNameEntry handles when user provides a topic name
func (th *TopicHandlers) HandleTopicNameEntry(update *gotgbot.Update) error {
	log.Printf("[TopicHandlers] Handling topic name entry: UserID=%d", update.Message.From.Id)

	ctx := th.WaitingForTopicName[update.Message.From.Id]
	topicName := strings.TrimSpace(update.Message.Text)

	if topicName == "" {
		_, err := th.messageService.SendMessage(ctx.ChatId, config.TopicNameEmptyError, &gotgbot.SendMessageOpts{})
		if err != nil {
			log.Printf("[TopicHandlers] Error sending empty topic name error: %v", err)
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
				log.Printf("[TopicHandlers] Error sending topic exists error: %v", err)
			}
			th.cleanupTopicCreation(update.Message.From.Id)
			return nil
		}
	}

	// Create the topic
	newTopic, err := th.topicService.CreateForumTopic(ctx.ChatId, topicName)
	if err != nil {
		log.Printf("[TopicHandlers] Error creating topic: %v", err)
		_, sendErr := th.messageService.SendMessage(ctx.ChatId, config.ErrorMessageCreateFailed, &gotgbot.SendMessageOpts{})
		if sendErr != nil {
			log.Printf("[TopicHandlers] Error sending create failed message: %v", sendErr)
		}
		th.cleanupTopicCreation(update.Message.From.Id)
		return err
	}

	// Send topic name as first message in new topic
	if newTopic != nil {
		_, err := th.messageService.SendMessage(ctx.ChatId, newTopic.Name, &gotgbot.SendMessageOpts{
			MessageThreadId: int64(newTopic.MessageThreadId),
		})
		if err != nil {
			log.Printf("[TopicHandlers] Error sending topic name as first message: %v", err)
		}
	}

	// Copy the original user message to the new topic
	if origMsg, ok := th.originalMessageStore[update.Message.From.Id]; ok && newTopic != nil {
		_, err := th.messageService.CopyMessageToTopicWithResult(ctx.ChatId, origMsg.Chat.Id, int(origMsg.MessageId), newTopic.MessageThreadId)
		if err != nil {
			log.Printf("[TopicHandlers] Error copying message to new topic: %v", err)
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
			confirmMsg := config.SuccessMessageSaved + newTopic.Name + preview

			// Send confirmation message to General
			_, err = th.messageService.SendMessage(ctx.ChatId, confirmMsg, &gotgbot.SendMessageOpts{
				MessageThreadId: 0,
			})
			if err != nil {
				log.Printf("[TopicHandlers] Error sending confirmation message: %v", err)
			}

			// Delete the original message from General after a short delay
			go func(botToken string, chatID int64, messageID int) {
				time.Sleep(config.DefaultMessageAutoDeleteDelay)
				_ = th.messageService.DeleteMessage(chatID, messageID)
			}(th.messageService.BotToken, origMsg.Chat.Id, int(origMsg.MessageId))
		}
	}

	// Clean up state
	th.cleanupTopicCreation(update.Message.From.Id)
	return nil
}

// HandleTopicSelectionCallback handles when user selects an existing topic
func (th *TopicHandlers) HandleTopicSelectionCallback(update *gotgbot.Update, originalMsg *gotgbot.Message, callbackData string) error {
	log.Printf("[TopicHandlers] Handling topic selection callback: %s", callbackData)

	// Extract topic name from callback data
	parts := strings.Split(callbackData, "_")
	if len(parts) < 2 {
		log.Printf("[TopicHandlers] Invalid callback data format: %s", callbackData)
		return nil
	}

	topicName := strings.Join(parts[:len(parts)-1], "_") // Rejoin in case topic name contains underscores

	// Find the topic
	topic, err := th.topicService.FindTopicByName(originalMsg.Chat.Id, topicName)
	if err != nil {
		log.Printf("[TopicHandlers] Error finding topic: %v", err)
		_, sendErr := th.messageService.SendMessage(originalMsg.Chat.Id, config.ErrorMessageNotFound, &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
		})
		if sendErr != nil {
			log.Printf("[TopicHandlers] Error sending not found message: %v", sendErr)
		}
		return err
	}

	// Copy message to the selected topic
	_, err = th.messageService.CopyMessageToTopicWithResult(originalMsg.Chat.Id, originalMsg.Chat.Id, int(originalMsg.MessageId), topic.MessageThreadId)
	if err != nil {
		log.Printf("[TopicHandlers] Error copying message to topic: %v", err)
		_, sendErr := th.messageService.SendMessage(originalMsg.Chat.Id, "❌ Failed to save message to topic.", &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
		})
		if sendErr != nil {
			log.Printf("[TopicHandlers] Error sending copy failed message: %v", sendErr)
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
	confirmMsg := config.SuccessMessageSaved + topic.Name + preview

	// Send confirmation message
	_, err = th.messageService.SendMessage(originalMsg.Chat.Id, confirmMsg, &gotgbot.SendMessageOpts{
		MessageThreadId: originalMsg.MessageThreadId,
	})
	if err != nil {
		log.Printf("[TopicHandlers] Error sending confirmation message: %v", err)
		return err
	}

	// Delete the original message after a short delay
	go func(botToken string, chatID int64, messageID int) {
		time.Sleep(config.DefaultMessageAutoDeleteDelay)
		_ = th.messageService.DeleteMessage(chatID, messageID)
	}(th.messageService.BotToken, originalMsg.Chat.Id, int(originalMsg.MessageId))

	log.Printf("[TopicHandlers] Successfully handled topic selection: Topic=%s", topic.Name)
	return nil
}

// HandleShowAllTopicsCallback handles showing all topics from suggestions
func (th *TopicHandlers) HandleShowAllTopicsCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	log.Printf("[TopicHandlers] Handling show all topics callback: ChatID=%d", originalMsg.Chat.Id)

	topics, err := th.topicService.GetForumTopics(originalMsg.Chat.Id)
	if err != nil {
		log.Printf("[TopicHandlers] Error getting topics: %v", err)
		_, sendErr := th.messageService.SendMessage(originalMsg.Chat.Id, config.ErrorMessageFailed, &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
		})
		if sendErr != nil {
			log.Printf("[TopicHandlers] Error sending error message: %v", sendErr)
		}
		return err
	}

	if len(topics) == 0 {
		_, err = th.messageService.SendMessage(originalMsg.Chat.Id, config.NoTopicsDiscoveredMessage, &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
		})
		if err != nil {
			log.Printf("[TopicHandlers] Error sending no topics message: %v", err)
			return err
		}
	} else {
		// Build keyboard with all existing topics
		keyboard, err := th.keyboardBuilder.BuildAllTopicsKeyboard(originalMsg, topics)
		if err != nil {
			log.Printf("[TopicHandlers] Error building all topics keyboard: %v", err)
			return err
		}

		// Store message references for all topic buttons
		for _, topic := range topics {
			topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
			th.messageStore[topicCallbackData] = originalMsg
		}

		backCallbackData := config.CallbackPrefixBackToSuggestions + strconv.FormatInt(originalMsg.MessageId, 10)
		th.messageStore[backCallbackData] = originalMsg

		// Try to update existing message or send new one
		callbackData := "suggestions_" + strconv.FormatInt(originalMsg.MessageId, 10)
		if keyboardMsgId, exists := th.keyboardMessageStore[callbackData]; exists {
			_, err = th.messageService.EditMessageText(originalMsg.Chat.Id, int64(keyboardMsgId), config.ChooseFromAllTopicsMessage, &gotgbot.EditMessageTextOpts{
				ReplyMarkup: *keyboard,
			})
			if err != nil {
				log.Printf("[TopicHandlers] Error updating message with all topics: %v", err)
				// If update fails, send new message
				newMsg, err := th.messageService.SendMessage(originalMsg.Chat.Id, config.ChooseFromAllTopicsMessage, &gotgbot.SendMessageOpts{
					MessageThreadId: originalMsg.MessageThreadId,
					ReplyMarkup:     *keyboard,
				})
				if err != nil {
					log.Printf("[TopicHandlers] Error sending new message with all topics: %v", err)
				} else {
					// Store keyboard message ID for all topic buttons
					for _, topic := range topics {
						topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
						th.keyboardMessageStore[topicCallbackData] = int(newMsg.MessageId)
					}
					th.keyboardMessageStore[backCallbackData] = int(newMsg.MessageId)
				}
			} else {
				// Store keyboard message ID for all topic buttons
				for _, topic := range topics {
					topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
					th.keyboardMessageStore[topicCallbackData] = keyboardMsgId
				}
				th.keyboardMessageStore[backCallbackData] = keyboardMsgId
			}
		} else {
			// Send new message with all topics
			newMsg, err := th.messageService.SendMessage(originalMsg.Chat.Id, config.ChooseFromAllTopicsMessage, &gotgbot.SendMessageOpts{
				MessageThreadId: originalMsg.MessageThreadId,
				ReplyMarkup:     *keyboard,
			})
			if err != nil {
				log.Printf("[TopicHandlers] Error sending message with all topics: %v", err)
			} else {
				// Store keyboard message ID for all topic buttons
				for _, topic := range topics {
					topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
					th.keyboardMessageStore[topicCallbackData] = int(newMsg.MessageId)
				}
				th.keyboardMessageStore[backCallbackData] = int(newMsg.MessageId)
			}
		}
	}

	log.Printf("[TopicHandlers] Successfully handled show all topics callback: ChatID=%d", originalMsg.Chat.Id)
	return nil
}

// HandleCreateTopicMenuCallback handles the create topic menu callback
func (th *TopicHandlers) HandleCreateTopicMenuCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	log.Printf("[TopicHandlers] Handling create topic menu callback: ChatID=%d", originalMsg.Chat.Id)

	_, err := th.messageService.SendMessage(originalMsg.Chat.Id, config.TopicCreationMenuMessage, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})
	if err != nil {
		log.Printf("[TopicHandlers] Error sending topic creation menu: %v", err)
		return err
	}

	// Set flag to wait for topic name
	th.WaitingForTopicName[originalMsg.From.Id] = TopicCreationContext{
		ChatId:        originalMsg.Chat.Id,
		ThreadId:      int64(originalMsg.MessageThreadId),
		OriginalMsgId: int64(originalMsg.MessageId),
	}

	log.Printf("[TopicHandlers] Successfully handled create topic menu callback: ChatID=%d", originalMsg.Chat.Id)
	return nil
}

// HandleShowAllTopicsMenuCallback handles the show all topics menu callback
func (th *TopicHandlers) HandleShowAllTopicsMenuCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	log.Printf("[TopicHandlers] Handling show all topics menu callback: ChatID=%d", originalMsg.Chat.Id)

	topics, err := th.topicService.GetForumTopics(originalMsg.Chat.Id)
	if err != nil {
		log.Printf("[TopicHandlers] Error getting topics: %v", err)
		_, sendErr := th.messageService.SendMessage(originalMsg.Chat.Id, config.ErrorMessageFailed, &gotgbot.SendMessageOpts{})
		if sendErr != nil {
			log.Printf("[TopicHandlers] Error sending error message: %v", sendErr)
		}
		return err
	}

	if len(topics) == 0 {
		_, err = th.messageService.SendMessage(originalMsg.Chat.Id, config.ErrorMessageNoTopics, &gotgbot.SendMessageOpts{})
		if err != nil {
			log.Printf("[TopicHandlers] Error sending no topics message: %v", err)
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
			log.Printf("[TopicHandlers] Error sending topics list: %v", err)
			return err
		}
	}

	log.Printf("[TopicHandlers] Successfully handled show all topics menu callback: ChatID=%d", originalMsg.Chat.Id)
	return nil
}

// Helper methods
func (th *TopicHandlers) cleanupTopicCreation(userID int64) {
	delete(th.WaitingForTopicName, userID)
	delete(th.originalMessageStore, userID)
}

func (th *TopicHandlers) IsRecentlyMovedMessage(messageID int64) bool {
	return th.recentlyMovedMessages[messageID]
}

func (th *TopicHandlers) MarkMessageAsMoved(messageID int64) {
	th.recentlyMovedMessages[messageID] = true
}

func (th *TopicHandlers) CleanupMovedMessage(messageID int64) {
	delete(th.recentlyMovedMessages, messageID)
}
