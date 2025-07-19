package handlers

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"save-message/internal/services"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// CallbackHandlers handles all callback-related interactions
type CallbackHandlers struct {
	messageService        *services.MessageService
	topicService          *services.TopicService
	aiService             *services.AIService
	messageStore          map[string]*gotgbot.Message
	keyboardMessageStore  map[string]int
	WaitingForTopicName   map[int64]TopicCreationContext
	originalMessageStore  map[int64]*gotgbot.Message
	recentlyMovedMessages map[int64]bool
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
		messageService:        messageService,
		topicService:          topicService,
		aiService:             aiService,
		messageStore:          make(map[string]*gotgbot.Message),
		keyboardMessageStore:  make(map[string]int),
		WaitingForTopicName:   make(map[int64]TopicCreationContext),
		originalMessageStore:  make(map[int64]*gotgbot.Message),
		recentlyMovedMessages: make(map[int64]bool),
	}
}

// HandleCallbackQuery handles all callback queries
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

	// Special handling for detectMessageOnOtherTopic_ok_ callback
	if strings.HasPrefix(callbackData, "detectMessageOnOtherTopic_ok_") {
		return ch.HandleWarningOkCallback(update)
	}

	originalMsg := ch.messageStore[callbackData]
	if originalMsg == nil {
		_, err := ch.messageService.SendMessage(update.CallbackQuery.From.Id, "❌ Error: Message not found. Please try again.", nil)
		if err != nil {
			log.Printf("[CallbackHandlers] Error sending error message: %v", err)
		}
		return nil
	}

	// Route to appropriate handler based on callback data
	switch {
	case strings.HasPrefix(callbackData, "create_new_folder_"):
		return ch.HandleNewTopicCreationRequest(update, originalMsg)
	case strings.HasPrefix(callbackData, "retry_"):
		return ch.HandleRetryCallback(update, originalMsg)
	case strings.HasPrefix(callbackData, "show_all_topics_"):
		return ch.HandleShowAllTopicsCallback(update, originalMsg)
	case callbackData == "create_topic_menu":
		return ch.HandleCreateTopicMenuCallback(update, originalMsg)
	case callbackData == "show_all_topics_menu":
		return ch.HandleShowAllTopicsMenuCallback(update, originalMsg)
	case strings.HasPrefix(callbackData, "back_to_suggestions_"):
		return ch.HandleBackToSuggestionsCallback(update, originalMsg)
	default:
		return ch.HandleTopicSelectionCallback(update, originalMsg, callbackData)
	}
}

// HandleWarningOkCallback handles the "Ok" button for warning messages
func (ch *CallbackHandlers) HandleWarningOkCallback(update *gotgbot.Update) error {
	log.Printf("[CallbackHandlers] Handling warning OK callback: %s", update.CallbackQuery.Data)

	// Delete the warning message itself (the message that contains the "Ok" button)
	err := ch.messageService.DeleteMessage(update.CallbackQuery.Message.Chat.Id, int(update.CallbackQuery.Message.MessageId))
	if err != nil {
		log.Printf("[CallbackHandlers] Error deleting warning message: %v", err)
	} else {
		log.Printf("[CallbackHandlers] Successfully deleted warning message: MessageId=%d", update.CallbackQuery.Message.MessageId)
	}

	return nil
}

// HandleNewTopicCreationRequest handles requests to create a new topic
func (ch *CallbackHandlers) HandleNewTopicCreationRequest(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	log.Printf("[CallbackHandlers] Handling new topic creation request: ChatID=%d", originalMsg.Chat.Id)

	// Ask user for topic name
	_, err := ch.messageService.SendMessage(originalMsg.Chat.Id, "📝 Please enter the name for your new topic:", &gotgbot.SendMessageOpts{
		MessageThreadId: originalMsg.MessageThreadId,
	})
	if err != nil {
		log.Printf("[CallbackHandlers] Error sending topic name prompt: %v", err)
		return err
	}

	// Store the context for topic creation
	ch.WaitingForTopicName[update.CallbackQuery.From.Id] = TopicCreationContext{
		ChatId:        originalMsg.Chat.Id,
		ThreadId:      int64(originalMsg.MessageThreadId),
		OriginalMsgId: int64(originalMsg.MessageId),
	}

	// Store the original message for this user
	ch.originalMessageStore[update.CallbackQuery.From.Id] = originalMsg

	// Delete the keyboard message
	if keyboardMsgId, exists := ch.keyboardMessageStore[update.CallbackQuery.Data]; exists {
		ch.messageService.DeleteMessage(originalMsg.Chat.Id, keyboardMsgId)
		delete(ch.keyboardMessageStore, update.CallbackQuery.Data)
	}

	log.Printf("[CallbackHandlers] Successfully handled new topic creation request: ChatID=%d", originalMsg.Chat.Id)
	return nil
}

// HandleRetryCallback handles retry button clicks
func (ch *CallbackHandlers) HandleRetryCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	log.Printf("[CallbackHandlers] Handling retry callback: ChatID=%d", originalMsg.Chat.Id)

	// Send a simple retry message
	_, err := ch.messageService.SendMessage(originalMsg.Chat.Id, "🔄 Retrying... Please send your message again.", &gotgbot.SendMessageOpts{
		MessageThreadId: originalMsg.MessageThreadId,
	})
	if err != nil {
		log.Printf("[CallbackHandlers] Error sending retry message: %v", err)
		return err
	}

	log.Printf("[CallbackHandlers] Successfully handled retry callback: ChatID=%d", originalMsg.Chat.Id)
	return nil
}

// HandleShowAllTopicsCallback handles showing all topics from suggestions
func (ch *CallbackHandlers) HandleShowAllTopicsCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	log.Printf("[CallbackHandlers] Handling show all topics callback: ChatID=%d", originalMsg.Chat.Id)

	topics, err := ch.topicService.GetForumTopics(originalMsg.Chat.Id)
	if err != nil {
		log.Printf("[CallbackHandlers] Error getting topics: %v", err)
		_, sendErr := ch.messageService.SendMessage(originalMsg.Chat.Id, "❌ Failed to get topics. Please try again.", &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
		})
		if sendErr != nil {
			log.Printf("[CallbackHandlers] Error sending error message: %v", sendErr)
		}
		return err
	}

	if len(topics) == 0 {
		_, err = ch.messageService.SendMessage(originalMsg.Chat.Id, "📁 No topics discovered yet. Create some topics and the bot will remember them!", &gotgbot.SendMessageOpts{
			MessageThreadId: originalMsg.MessageThreadId,
		})
		if err != nil {
			log.Printf("[CallbackHandlers] Error sending no topics message: %v", err)
			return err
		}
	} else {
		// Build keyboard with all existing topics
		var rows [][]gotgbot.InlineKeyboardButton

		// Add all existing topics as buttons
		for _, topic := range topics {
			callbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
			ch.messageStore[callbackData] = originalMsg
			rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "📁 " + topic.Name, CallbackData: callbackData}})
		}

		// Add back button
		backCallbackData := "back_to_suggestions_" + strconv.FormatInt(originalMsg.MessageId, 10)
		ch.messageStore[backCallbackData] = originalMsg
		backBtn := gotgbot.InlineKeyboardButton{Text: "⬅️ Back to Suggestions", CallbackData: backCallbackData}
		rows = append(rows, []gotgbot.InlineKeyboardButton{backBtn})

		keyboard := &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}

		// Always try to update the existing message first
		if keyboardMsgId, exists := ch.keyboardMessageStore[update.CallbackQuery.Data]; exists {
			_, err = ch.messageService.EditMessageText(originalMsg.Chat.Id, int64(keyboardMsgId), "Choose from all existing topics:", &gotgbot.EditMessageTextOpts{
				ReplyMarkup: *keyboard,
			})
			if err != nil {
				log.Printf("[CallbackHandlers] Error updating message with all topics: %v", err)
				// If update fails, try to find the message by searching through all stored keyboard messages
				for storedCallback, storedMsgId := range ch.keyboardMessageStore {
					if strings.Contains(storedCallback, strconv.FormatInt(originalMsg.MessageId, 10)) {
						_, updateErr := ch.messageService.EditMessageText(originalMsg.Chat.Id, int64(storedMsgId), "Choose from all existing topics:", &gotgbot.EditMessageTextOpts{
							ReplyMarkup: *keyboard,
						})
						if updateErr == nil {
							// Store the keyboard message ID for all topic buttons
							for _, topic := range topics {
								topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
								ch.keyboardMessageStore[topicCallbackData] = storedMsgId
							}
							ch.keyboardMessageStore[backCallbackData] = storedMsgId
							break
						}
					}
				}
			} else {
				// Store the keyboard message ID for all topic buttons
				for _, topic := range topics {
					topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
					ch.keyboardMessageStore[topicCallbackData] = keyboardMsgId
				}
				ch.keyboardMessageStore[backCallbackData] = keyboardMsgId
			}
		} else {
			// Send new message with all topics
			newMsg, err := ch.messageService.SendMessage(originalMsg.Chat.Id, "Choose from all existing topics:", &gotgbot.SendMessageOpts{
				MessageThreadId: originalMsg.MessageThreadId,
				ReplyMarkup:     *keyboard,
			})
			if err != nil {
				log.Printf("[CallbackHandlers] Error sending message with all topics: %v", err)
			} else {
				// Store the keyboard message ID for all topic buttons
				for _, topic := range topics {
					topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
					ch.keyboardMessageStore[topicCallbackData] = int(newMsg.MessageId)
				}
				ch.keyboardMessageStore[backCallbackData] = int(newMsg.MessageId)
			}
		}
	}

	log.Printf("[CallbackHandlers] Successfully handled show all topics callback: ChatID=%d", originalMsg.Chat.Id)
	return nil
}

// HandleCreateTopicMenuCallback handles the create topic menu callback
func (ch *CallbackHandlers) HandleCreateTopicMenuCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	log.Printf("[CallbackHandlers] Handling create topic menu callback: ChatID=%d", originalMsg.Chat.Id)

	// Show topic creation input prompt
	_, err := ch.messageService.SendMessage(originalMsg.Chat.Id, "📝 **Create New Topic**\n\nPlease send the name of the topic you want to create:", &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})
	if err != nil {
		log.Printf("[CallbackHandlers] Error sending topic creation prompt: %v", err)
		return err
	}

	// Set flag to wait for topic name
	ch.WaitingForTopicName[originalMsg.From.Id] = TopicCreationContext{
		ChatId:        originalMsg.Chat.Id,
		ThreadId:      int64(originalMsg.MessageThreadId),
		OriginalMsgId: int64(originalMsg.MessageId),
	}

	log.Printf("[CallbackHandlers] Successfully handled create topic menu callback: ChatID=%d", originalMsg.Chat.Id)
	return nil
}

// HandleShowAllTopicsMenuCallback handles the show all topics menu callback
func (ch *CallbackHandlers) HandleShowAllTopicsMenuCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	log.Printf("[CallbackHandlers] Handling show all topics menu callback: ChatID=%d", originalMsg.Chat.Id)

	// Show all topics from database
	topics, err := ch.topicService.GetForumTopics(originalMsg.Chat.Id)
	if err != nil {
		log.Printf("[CallbackHandlers] Error getting topics: %v", err)
		_, sendErr := ch.messageService.SendMessage(originalMsg.Chat.Id, "❌ Failed to get topics.", &gotgbot.SendMessageOpts{})
		if sendErr != nil {
			log.Printf("[CallbackHandlers] Error sending error message: %v", sendErr)
		}
		return err
	}

	if len(topics) == 0 {
		_, err = ch.messageService.SendMessage(originalMsg.Chat.Id, "📁 No topics found yet. Send a message to create your first topic!", &gotgbot.SendMessageOpts{})
		if err != nil {
			log.Printf("[CallbackHandlers] Error sending no topics message: %v", err)
			return err
		}
	} else {
		topicList := "📁 **Your Topics:**\n"
		for _, topic := range topics {
			topicList += "• " + topic.Name + "\n"
		}
		_, err = ch.messageService.SendMessage(originalMsg.Chat.Id, topicList, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
		if err != nil {
			log.Printf("[CallbackHandlers] Error sending topics list: %v", err)
			return err
		}
	}

	log.Printf("[CallbackHandlers] Successfully handled show all topics menu callback: ChatID=%d", originalMsg.Chat.Id)
	return nil
}

// HandleBackToSuggestionsCallback handles going back to AI suggestions
func (ch *CallbackHandlers) HandleBackToSuggestionsCallback(update *gotgbot.Update, originalMsg *gotgbot.Message) error {
	log.Printf("[CallbackHandlers] Handling back to suggestions callback: ChatID=%d", originalMsg.Chat.Id)

	// Go back to AI suggestions
	parts := strings.Split(update.CallbackQuery.Data, "_")
	if len(parts) >= 4 {
		messageId, err := strconv.ParseInt(parts[3], 10, 64)
		if err == nil {
			// Find the original message and reprocess it
			for _, storedMsg := range ch.messageStore {
				if storedMsg.MessageId == messageId {
					// Reprocess the original message to show AI suggestions
					go func(msg *gotgbot.Message) {
						// Send waiting message first
						waitingMsg, err := ch.messageService.SendMessage(msg.Chat.Id, "🤔 Thinking...", &gotgbot.SendMessageOpts{
							MessageThreadId: msg.MessageThreadId,
						})
						if err != nil {
							log.Printf("[CallbackHandlers] Error sending waiting message: %v", err)
							return
						}

						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()

						// Get existing topics from database
						topics, err := ch.topicService.GetForumTopics(msg.Chat.Id)
						existingFolders := []string{}
						if err == nil {
							for _, topic := range topics {
								existingFolders = append(existingFolders, topic.Name)
							}
						}

						suggestions, err := ch.aiService.SuggestFolders(ctx, msg.Text, existingFolders)
						if err != nil {
							log.Printf("[CallbackHandlers] AI error: %v", err)
							// Update waiting message with error and retry button
							retryKeyboard := &gotgbot.InlineKeyboardMarkup{
								InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
									{{Text: "🔄 Try Again", CallbackData: "retry_" + strconv.FormatInt(msg.MessageId, 10)}},
								},
							}
							_, err = ch.messageService.EditMessageText(msg.Chat.Id, waitingMsg.MessageId, "Sorry, I couldn't suggest folders right now.", &gotgbot.EditMessageTextOpts{
								ReplyMarkup: *retryKeyboard,
							})
							if err != nil {
								log.Printf("[CallbackHandlers] Error updating waiting message: %v", err)
							}
							return
						}
						log.Printf("[CallbackHandlers] AI suggestions: %v", suggestions)

						// Build inline keyboard
						var rows [][]gotgbot.InlineKeyboardButton

						// Separate existing and new topics
						var existingTopics []string
						var newTopics []string

						log.Printf("[CallbackHandlers] Available topics: %v", topics)
						log.Printf("[CallbackHandlers] AI suggestions: %v", suggestions)

						for _, folder := range suggestions {
							// Check if this is an existing topic (case-insensitive)
							isExisting := false
							var existingTopicName string
							for _, topic := range topics {
								if strings.EqualFold(topic.Name, folder) {
									isExisting = true
									existingTopicName = topic.Name // Use the exact name from the topic
									break
								}
							}

							// Skip General topic
							if strings.EqualFold(folder, "General") {
								log.Printf("[CallbackHandlers] Skipping General topic")
								continue
							}

							if isExisting {
								log.Printf("[CallbackHandlers] Found existing topic: %s (original: %s)", folder, existingTopicName)
								existingTopics = append(existingTopics, existingTopicName) // Use exact name
							} else {
								log.Printf("[CallbackHandlers] New topic suggested: %s", folder)
								newTopics = append(newTopics, folder)
							}
						}

						log.Printf("[CallbackHandlers] Existing topics to show: %v", existingTopics)
						log.Printf("[CallbackHandlers] New topics to show: %v", newTopics)

						// Add existing topics with folder icon
						for _, folder := range existingTopics {
							callbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
							ch.messageStore[callbackData] = msg
							rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "📁 " + folder, CallbackData: callbackData}})
						}

						// Add new topics with plus icon
						for _, folder := range newTopics {
							cleanFolder := strings.TrimSpace(folder)
							// Skip suggestions that are too long or contain newlines
							if len(cleanFolder) == 0 || len(cleanFolder) > 50 || strings.Contains(cleanFolder, "\n") {
								continue
							}
							callbackData := cleanFolder + "_" + strconv.FormatInt(msg.MessageId, 10)
							ch.messageStore[callbackData] = msg
							rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "➕ " + cleanFolder, CallbackData: callbackData}})
						}

						// Add create new folder option
						createCallbackData := "create_new_folder_" + strconv.FormatInt(msg.MessageId, 10)
						ch.messageStore[createCallbackData] = msg
						createBtn := gotgbot.InlineKeyboardButton{Text: "📝 Create Custom Topic", CallbackData: createCallbackData}
						rows = append(rows, []gotgbot.InlineKeyboardButton{createBtn})

						// Add show all topics button if there are existing topics
						topics, err = ch.topicService.GetForumTopics(msg.Chat.Id)
						if err == nil && len(topics) > 0 {
							showAllCallbackData := "show_all_topics_" + strconv.FormatInt(msg.MessageId, 10)
							ch.messageStore[showAllCallbackData] = msg
							showAllBtn := gotgbot.InlineKeyboardButton{Text: "📁 Show All Topics", CallbackData: showAllCallbackData}
							rows = append(rows, []gotgbot.InlineKeyboardButton{showAllBtn})
						}

						keyboard := &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}

						// Update waiting message with keyboard
						_, err = ch.messageService.EditMessageText(msg.Chat.Id, waitingMsg.MessageId, "Choose a folder:", &gotgbot.EditMessageTextOpts{
							ReplyMarkup: *keyboard,
						})
						if err != nil {
							log.Printf("[CallbackHandlers] Error updating message with keyboard: %v", err)
						} else {
							// Store the keyboard message ID for each callback data
							keyboardMsgId := int(waitingMsg.MessageId)
							for _, folder := range suggestions {
								callbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
								ch.keyboardMessageStore[callbackData] = keyboardMsgId
							}
							createCallbackData := "create_new_folder_" + strconv.FormatInt(msg.MessageId, 10)
							ch.keyboardMessageStore[createCallbackData] = keyboardMsgId
						}
					}(storedMsg)
					break
				}
			}
		}
	}

	log.Printf("[CallbackHandlers] Successfully handled back to suggestions callback: ChatID=%d", originalMsg.Chat.Id)
	return nil
}

// HandleTopicSelectionCallback handles topic selection callbacks
func (ch *CallbackHandlers) HandleTopicSelectionCallback(update *gotgbot.Update, originalMsg *gotgbot.Message, callbackData string) error {
	log.Printf("[CallbackHandlers] Handling topic selection callback: ChatID=%d, CallbackData=%s", originalMsg.Chat.Id, callbackData)

	// Check if this is a topic selection callback (format: "TopicName_MessageId")
	parts := strings.Split(callbackData, "_")
	if len(parts) >= 2 {
		// Try to find the original message by searching through all stored messages
		var messageId int64

		// Extract message ID from the last part
		if msgId, err := strconv.ParseInt(parts[len(parts)-1], 10, 64); err == nil {
			messageId = msgId
			// Find the original message by message ID
			for _, storedMsg := range ch.messageStore {
				if storedMsg.MessageId == messageId {
					originalMsg = storedMsg
					break
				}
			}
		}

		if originalMsg == nil {
			_, err := ch.messageService.SendMessage(update.CallbackQuery.From.Id, "❌ Error: Original message not found. Please try again.", nil)
			if err != nil {
				log.Printf("[CallbackHandlers] Error sending error message: %v", err)
			}
			return nil
		}

		// Extract topic name (everything except the last part which is message ID)
		topicName := strings.Join(parts[:len(parts)-1], "_")
		log.Printf("[CallbackHandlers] Topic selection callback: topicName='%s', callbackData='%s'", topicName, callbackData)

		// Find the topic in the database
		topics, err := ch.topicService.GetForumTopics(originalMsg.Chat.Id)
		if err != nil {
			log.Printf("[CallbackHandlers] Error getting topics: %v", err)
			_, sendErr := ch.messageService.SendMessage(originalMsg.Chat.Id, "❌ Failed to get topics.", &gotgbot.SendMessageOpts{
				MessageThreadId: originalMsg.MessageThreadId,
			})
			if sendErr != nil {
				log.Printf("[CallbackHandlers] Error sending error message: %v", sendErr)
			}
			return err
		}
		log.Printf("[CallbackHandlers] Available topics in database: %v", topics)

		var targetTopic *services.ForumTopic
		for _, topic := range topics {
			if strings.EqualFold(topic.Name, topicName) {
				targetTopic = &topic
				log.Printf("[CallbackHandlers] Found existing topic: %s (MessageThreadId: %d)", topic.Name, topic.MessageThreadId)
				break
			}
		}

		// If topic doesn't exist, create it
		if targetTopic == nil {
			log.Printf("[CallbackHandlers] Topic '%s' not found in database, creating new topic", topicName)

			// Create the new topic
			newTopic, err := ch.topicService.CreateForumTopic(originalMsg.Chat.Id, topicName)
			if err != nil {
				log.Printf("[CallbackHandlers] Error creating new topic '%s': %v", topicName, err)
				_, sendErr := ch.messageService.SendMessage(originalMsg.Chat.Id, "❌ Failed to create new topic.", &gotgbot.SendMessageOpts{
					MessageThreadId: originalMsg.MessageThreadId,
				})
				if sendErr != nil {
					log.Printf("[CallbackHandlers] Error sending error message: %v", sendErr)
				}
				return err
			}

			log.Printf("[CallbackHandlers] Successfully created new topic: %s (MessageThreadId: %d)", newTopic.Name, newTopic.MessageThreadId)
			targetTopic = newTopic
		}

		// Copy message to the selected topic
		log.Printf("[CallbackHandlers] Copying message to topic: MessageId=%d, TopicName=%s, MessageThreadId=%d",
			originalMsg.MessageId, targetTopic.Name, targetTopic.MessageThreadId)

		err = ch.messageService.CopyMessageToTopic(originalMsg.Chat.Id, originalMsg.Chat.Id, int(originalMsg.MessageId), targetTopic.MessageThreadId)
		if err != nil {
			log.Printf("[CallbackHandlers] Error copying message to topic: %v", err)
			_, sendErr := ch.messageService.SendMessage(originalMsg.Chat.Id, "❌ Failed to move message to topic.", &gotgbot.SendMessageOpts{
				MessageThreadId: originalMsg.MessageThreadId,
			})
			if sendErr != nil {
				log.Printf("[CallbackHandlers] Error sending error message: %v", sendErr)
			}
			return err
		}

		// Build preview: first 2 lines of the original message
		previewLines := strings.SplitN(originalMsg.Text, "\n", 3)
		preview := ""
		if len(previewLines) > 0 {
			preview += "\n\"" + previewLines[0] + "\""
		}
		if len(previewLines) > 1 {
			preview += "\n\"" + previewLines[1] + "\""
		}
		confirmMsg := "✅ Message saved to topic: " + targetTopic.Name + preview

		// Send confirmation message to General
		_, err = ch.messageService.SendMessage(originalMsg.Chat.Id, confirmMsg, &gotgbot.SendMessageOpts{
			MessageThreadId: 0,
		})
		if err != nil {
			log.Printf("[CallbackHandlers] Error sending confirmation message: %v", err)
		}

		// Delete the original message from General after a short delay
		go func(botToken string, chatID int64, messageID int) {
			time.Sleep(1 * time.Second)
			_ = ch.messageService.DeleteMessage(chatID, messageID)
		}(ch.messageService.BotToken, originalMsg.Chat.Id, int(originalMsg.MessageId))

		// Mark message as recently moved to prevent reprocessing
		ch.recentlyMovedMessages[originalMsg.MessageId] = true

		// Delete the keyboard message
		if keyboardMsgId, exists := ch.keyboardMessageStore[callbackData]; exists {
			ch.messageService.DeleteMessage(originalMsg.Chat.Id, keyboardMsgId)
			delete(ch.keyboardMessageStore, callbackData)
		}

		log.Printf("[CallbackHandlers] Successfully handled topic selection callback: ChatID=%d, Topic=%s", originalMsg.Chat.Id, targetTopic.Name)
	}

	return nil
}

// HandleTopicNameEntry handles when a user enters a topic name
func (ch *CallbackHandlers) HandleTopicNameEntry(update *gotgbot.Update) error {
	log.Printf("[CallbackHandlers] Handling topic name entry: ChatID=%d, UserID=%d", update.Message.Chat.Id, update.Message.From.Id)

	ctx, waiting := ch.WaitingForTopicName[update.Message.From.Id]
	if !waiting {
		return nil // Not waiting for topic name
	}

	topicName := strings.TrimSpace(update.Message.Text)
	if topicName == "" {
		_, err := ch.messageService.SendMessage(update.Message.Chat.Id, "❌ Topic name cannot be empty. Please try again.", &gotgbot.SendMessageOpts{})
		if err != nil {
			log.Printf("[CallbackHandlers] Error sending empty topic name error: %v", err)
		}
		return nil
	}

	// Check if topic already exists
	exists, err := ch.topicService.TopicExists(ctx.ChatId, topicName)
	if err != nil {
		log.Printf("[CallbackHandlers] Error checking if topic exists: %v", err)
		_, sendErr := ch.messageService.SendMessage(ctx.ChatId, "❌ Error checking topic existence. Please try again.", &gotgbot.SendMessageOpts{})
		if sendErr != nil {
			log.Printf("[CallbackHandlers] Error sending error message: %v", sendErr)
		}
		return err
	}

	if exists {
		_, err := ch.messageService.SendMessage(ctx.ChatId, "❌ A topic with this name already exists. Please choose a different name.", &gotgbot.SendMessageOpts{})
		if err != nil {
			log.Printf("[CallbackHandlers] Error sending topic exists error: %v", err)
		}
		ch.cleanupTopicCreation(update.Message.From.Id)
		return nil
	}

	// Create the topic in the correct chat/thread
	newTopic, err := ch.topicService.CreateForumTopic(ctx.ChatId, topicName)
	if err != nil {
		log.Printf("[CallbackHandlers] Error creating topic: %v", err)
		_, sendErr := ch.messageService.SendMessage(ctx.ChatId, "❌ Failed to create topic. Please try again.", &gotgbot.SendMessageOpts{})
		if sendErr != nil {
			log.Printf("[CallbackHandlers] Error sending error message: %v", sendErr)
		}
		ch.cleanupTopicCreation(update.Message.From.Id)
		return err
	}

	// After creating the topic and before copying the original user message, send the topic name as the first message in the new topic
	if newTopic != nil {
		_, err := ch.messageService.SendMessage(ctx.ChatId, newTopic.Name, &gotgbot.SendMessageOpts{
			MessageThreadId: int64(newTopic.MessageThreadId),
		})
		if err != nil {
			log.Printf("[CallbackHandlers] Error sending topic name as first message: %v", err)
		}
	}

	// Copy the original user message to the new topic
	if origMsg, ok := ch.originalMessageStore[update.Message.From.Id]; ok && newTopic != nil {
		// Copy the message and get the new message ID
		_, err := ch.messageService.CopyMessageToTopicWithResult(ctx.ChatId, origMsg.Chat.Id, int(origMsg.MessageId), newTopic.MessageThreadId)
		if err != nil {
			log.Printf("[CallbackHandlers] Error copying message to new topic: %v", err)
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
			confirmMsg := "✅ Message saved to topic: " + newTopic.Name + preview
			// Send confirmation message to General
			_, err = ch.messageService.SendMessage(ctx.ChatId, confirmMsg, &gotgbot.SendMessageOpts{
				MessageThreadId: 0,
			})
			if err != nil {
				log.Printf("[CallbackHandlers] Error sending confirmation message: %v", err)
			}
			// Delete the original message from General after a short delay
			go func(botToken string, chatID int64, messageID int) {
				time.Sleep(1 * time.Second)
				_ = ch.messageService.DeleteMessage(chatID, messageID)
			}(ch.messageService.BotToken, origMsg.Chat.Id, int(origMsg.MessageId))
		}
	}

	// Clean up state
	ch.cleanupTopicCreation(update.Message.From.Id)

	log.Printf("[CallbackHandlers] Successfully handled topic name entry: ChatID=%d, Topic=%s", update.Message.Chat.Id, topicName)
	return nil
}

// cleanupTopicCreation cleans up topic creation state
func (ch *CallbackHandlers) cleanupTopicCreation(userID int64) {
	delete(ch.WaitingForTopicName, userID)
	delete(ch.originalMessageStore, userID)
}

// IsRecentlyMovedMessage checks if a message was recently moved
func (ch *CallbackHandlers) IsRecentlyMovedMessage(messageID int64) bool {
	return ch.recentlyMovedMessages[messageID]
}

// MarkMessageAsMoved marks a message as recently moved
func (ch *CallbackHandlers) MarkMessageAsMoved(messageID int64) {
	ch.recentlyMovedMessages[messageID] = true
}

// CleanupMovedMessage removes a message from the recently moved list
func (ch *CallbackHandlers) CleanupMovedMessage(messageID int64) {
	delete(ch.recentlyMovedMessages, messageID)
}
