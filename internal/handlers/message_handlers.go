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

// MessageHandlers handles all message-related interactions
type MessageHandlers struct {
	MessageService *services.MessageService
	topicService   *services.TopicService
	aiService      *services.AIService
}

// NewMessageHandlers creates a new message handlers instance
func NewMessageHandlers(messageService *services.MessageService, topicService *services.TopicService, aiService *services.AIService) *MessageHandlers {
	return &MessageHandlers{
		MessageService: messageService,
		topicService:   topicService,
		aiService:      aiService,
	}
}

// HandleStartCommand handles the /start command
func (mh *MessageHandlers) HandleStartCommand(update *gotgbot.Update) error {
	log.Printf("[MessageHandlers] Handling /start command: ChatID=%d", update.Message.Chat.Id)

	welcome := "Save Message is your personal assistant inside Telegram.\n\nIt helps you organize your saved messages using Topics and smart suggestions — without using any commands.\nYou can categorize, edit, and retrieve your notes easily with inline buttons.\n\n🛡️ 100% private: all your content stays inside Telegram.\n\nJust write — we'll handle the rest.\n\nFor more info, send /help."

	_, err := mh.MessageService.SendMessage(update.Message.Chat.Id, welcome, nil)
	if err != nil {
		log.Printf("[MessageHandlers] Error sending start message: %v", err)
		return err
	}

	log.Printf("[MessageHandlers] Successfully sent start message: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// HandleHelpCommand handles the /help command
func (mh *MessageHandlers) HandleHelpCommand(update *gotgbot.Update) error {
	log.Printf("[MessageHandlers] Handling /help command: ChatID=%d", update.Message.Chat.Id)

	helpText := `🤖 **Save Message Bot Help**

**How to use:**
• Simply send any message and the bot will suggest relevant folders
• Click on a suggested folder to save your message there
• Use "📁 Show All Topics" to browse all existing topics

**Commands:**
• /start - Start the bot
• /help - Show this help message
• /topics - List all your topics
• /addtopic - Create a new topic manually

**Important:** ⚠️ **Don't create topics manually in Save message group!** Let the bot create them automatically when you save messages. This ensures proper organization and prevents confusion.

**Tips:**
• The bot uses AI to suggest relevant folders
• Existing topics show with 📁 icon, new ones with ➕
• Messages are automatically cleaned from General topic after saving
• Success messages auto-delete after 1 minute`

	_, err := mh.MessageService.SendMessage(update.Message.Chat.Id, helpText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})
	if err != nil {
		log.Printf("[MessageHandlers] Error sending help message: %v", err)
		return err
	}

	log.Printf("[MessageHandlers] Successfully sent help message: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// HandleTopicsCommand handles the /topics command
func (mh *MessageHandlers) HandleTopicsCommand(update *gotgbot.Update) error {
	log.Printf("[MessageHandlers] Handling /topics command: ChatID=%d", update.Message.Chat.Id)

	topics, err := mh.topicService.GetForumTopics(update.Message.Chat.Id)
	if err != nil {
		log.Printf("[MessageHandlers] Error getting topics: %v", err)
		_, sendErr := mh.MessageService.SendMessage(update.Message.Chat.Id, "❌ Failed to get topics.", &gotgbot.SendMessageOpts{})
		if sendErr != nil {
			log.Printf("[MessageHandlers] Error sending error message: %v", sendErr)
		}
		return err
	}

	if len(topics) == 0 {
		_, err = mh.MessageService.SendMessage(update.Message.Chat.Id, "📁 No topics found yet. Send a message to create your first topic!", &gotgbot.SendMessageOpts{})
		if err != nil {
			log.Printf("[MessageHandlers] Error sending no topics message: %v", err)
			return err
		}
	} else {
		topicList := "📁 **Your Topics:**\n"
		for _, topic := range topics {
			topicList += "• " + topic.Name + "\n"
		}
		_, err = mh.MessageService.SendMessage(update.Message.Chat.Id, topicList, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
		if err != nil {
			log.Printf("[MessageHandlers] Error sending topics list: %v", err)
			return err
		}
	}

	log.Printf("[MessageHandlers] Successfully handled /topics command: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// HandleAddTopicCommand handles the /addtopic command
func (mh *MessageHandlers) HandleAddTopicCommand(update *gotgbot.Update) error {
	log.Printf("[MessageHandlers] Handling /addtopic command: ChatID=%d", update.Message.Chat.Id)

	keyboard := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: "📝 Create New Topic", CallbackData: "create_topic_menu"}},
		},
	}

	_, err := mh.MessageService.SendMessage(update.Message.Chat.Id, "Choose an option:", &gotgbot.SendMessageOpts{
		ReplyMarkup: *keyboard,
	})
	if err != nil {
		log.Printf("[MessageHandlers] Error sending add topic menu: %v", err)
		return err
	}

	log.Printf("[MessageHandlers] Successfully handled /addtopic command: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// HandleBotMention handles when the bot is mentioned
func (mh *MessageHandlers) HandleBotMention(update *gotgbot.Update) error {
	log.Printf("[MessageHandlers] Handling bot mention: ChatID=%d", update.Message.Chat.Id)

	keyboard := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: "📝 Create New Topic", CallbackData: "create_topic_menu"}},
			{{Text: "📁 Show All Topics", CallbackData: "show_all_topics_menu"}},
		},
	}

	_, err := mh.MessageService.SendMessage(update.Message.Chat.Id, "🤖 **Bot Menu**\n\nWhat would you like to do?", &gotgbot.SendMessageOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: *keyboard,
	})
	if err != nil {
		log.Printf("[MessageHandlers] Error sending bot menu: %v", err)
		return err
	}

	log.Printf("[MessageHandlers] Successfully handled bot mention: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// HandleNonGeneralTopicMessage handles messages sent in non-General topics
func (mh *MessageHandlers) HandleNonGeneralTopicMessage(update *gotgbot.Update) error {
	log.Printf("[MessageHandlers] Handling non-General topic message: ChatID=%d, ThreadID=%d, MessageID=%d",
		update.Message.Chat.Id, update.Message.MessageThreadId, update.Message.MessageId)

	// Delete the user's message immediately
	err := mh.MessageService.DeleteMessage(update.Message.Chat.Id, int(update.Message.MessageId))
	if err != nil {
		log.Printf("[MessageHandlers] Error deleting message from non-General topic: %v", err)
	} else {
		log.Printf("[MessageHandlers] Successfully deleted message from non-General topic: MessageID=%d", update.Message.MessageId)
	}

	// Send warning message with "Ok" button
	callbackData := "detectMessageOnOtherTopic_ok_" + strconv.FormatInt(update.Message.MessageId, 10)
	keyboard := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: "Ok", CallbackData: callbackData}},
		},
	}

	warningMsg, err := mh.MessageService.SendMessage(update.Message.Chat.Id,
		"⚠️ **Please send messages only in the General topic!**\n\nThis message will be removed automatically in 1 minute.",
		&gotgbot.SendMessageOpts{
			MessageThreadId: update.Message.MessageThreadId,
			ParseMode:       "Markdown",
			ReplyMarkup:     *keyboard,
		})

	if err != nil {
		log.Printf("[MessageHandlers] Error sending warning message: %v", err)
		return err
	}

	log.Printf("[MessageHandlers] Successfully sent warning message: MessageID=%d", warningMsg.MessageId)

	// Auto-delete warning message after 1 minute
	go func(botToken string, chatID int64, messageID int64, threadID int64) {
		time.Sleep(60 * time.Second)
		err := mh.MessageService.DeleteMessage(chatID, int(messageID))
		if err != nil {
			log.Printf("[MessageHandlers] Error auto-deleting warning message: %v", err)
		} else {
			log.Printf("[MessageHandlers] Successfully auto-deleted warning message: MessageID=%d", messageID)
		}
	}(mh.MessageService.BotToken, update.Message.Chat.Id, warningMsg.MessageId, update.Message.MessageThreadId)

	log.Printf("[MessageHandlers] Successfully handled non-General topic message: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// HandleGeneralTopicMessage handles messages sent in the General topic
func (mh *MessageHandlers) HandleGeneralTopicMessage(update *gotgbot.Update) error {
	log.Printf("[MessageHandlers] Handling General topic message: ChatID=%d, MessageID=%d",
		update.Message.Chat.Id, update.Message.MessageId)

	// Check if message mentions the bot
	messageText := strings.ToLower(update.Message.Text)
	if strings.Contains(messageText, "@savemessagbot") || strings.Contains(messageText, "@savemessagebot") {
		return mh.HandleBotMention(update)
	}

	// Process the message asynchronously for AI suggestions
	go func(msg *gotgbot.Message) {
		// Send waiting message first
		waitingMsg, err := mh.MessageService.SendMessage(msg.Chat.Id, "🤔 Thinking...", &gotgbot.SendMessageOpts{
			MessageThreadId: msg.MessageThreadId,
		})
		if err != nil {
			log.Printf("[MessageHandlers] Error sending waiting message: %v", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get existing topics from database
		topics, err := mh.topicService.GetForumTopics(msg.Chat.Id)
		existingFolders := []string{}
		if err == nil {
			for _, topic := range topics {
				existingFolders = append(existingFolders, topic.Name)
			}
		}

		suggestions, err := mh.aiService.SuggestFolders(ctx, msg.Text, existingFolders)
		if err != nil {
			log.Printf("[MessageHandlers] AI error: %v", err)
			// Update waiting message with error and retry button
			retryKeyboard := &gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
					{{Text: "🔄 Try Again", CallbackData: "retry_" + strconv.FormatInt(msg.MessageId, 10)}},
				},
			}
			_, err = mh.MessageService.EditMessageText(msg.Chat.Id, waitingMsg.MessageId, "Sorry, I couldn't suggest folders right now.", &gotgbot.EditMessageTextOpts{
				ReplyMarkup: *retryKeyboard,
			})
			if err != nil {
				log.Printf("[MessageHandlers] Error updating waiting message: %v", err)
			}
			return
		}

		log.Printf("[MessageHandlers] AI suggestions: %v", suggestions)

		// Build suggestion keyboard
		keyboard, err := mh.buildSuggestionKeyboard(msg, suggestions, topics)
		if err != nil {
			log.Printf("[MessageHandlers] Error building suggestion keyboard: %v", err)
			return
		}

		// Update waiting message with keyboard
		_, err = mh.MessageService.EditMessageText(msg.Chat.Id, waitingMsg.MessageId, "Choose a folder:", &gotgbot.EditMessageTextOpts{
			ReplyMarkup: *keyboard,
		})
		if err != nil {
			log.Printf("[MessageHandlers] Error updating message with keyboard: %v", err)
		}
	}(update.Message)

	log.Printf("[MessageHandlers] Successfully handled General topic message: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// buildSuggestionKeyboard builds the keyboard for AI suggestions
func (mh *MessageHandlers) buildSuggestionKeyboard(msg *gotgbot.Message, suggestions []string, topics []services.ForumTopic) (*gotgbot.InlineKeyboardMarkup, error) {
	var rows [][]gotgbot.InlineKeyboardButton

	// Separate existing and new topics
	var existingTopics []string
	var newTopics []string

	log.Printf("[MessageHandlers] Available topics: %v", topics)
	log.Printf("[MessageHandlers] AI suggestions: %v", suggestions)

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
			log.Printf("[MessageHandlers] Skipping General topic")
			continue
		}

		if isExisting {
			log.Printf("[MessageHandlers] Found existing topic: %s (original: %s)", folder, existingTopicName)
			existingTopics = append(existingTopics, existingTopicName) // Use exact name
		} else {
			log.Printf("[MessageHandlers] New topic suggested: %s", folder)
			newTopics = append(newTopics, folder)
		}
	}

	log.Printf("[MessageHandlers] Existing topics to show: %v", existingTopics)
	log.Printf("[MessageHandlers] New topics to show: %v", newTopics)

	// Add existing topics with folder icon
	for _, folder := range existingTopics {
		callbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
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
		rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "➕ " + cleanFolder, CallbackData: callbackData}})
	}

	// Add create new folder option
	createCallbackData := "create_new_folder_" + strconv.FormatInt(msg.MessageId, 10)
	createBtn := gotgbot.InlineKeyboardButton{Text: "📝 Create Custom Topic", CallbackData: createCallbackData}
	rows = append(rows, []gotgbot.InlineKeyboardButton{createBtn})

	// Add show all topics button if there are existing topics
	if len(topics) > 0 {
		showAllCallbackData := "show_all_topics_" + strconv.FormatInt(msg.MessageId, 10)
		showAllBtn := gotgbot.InlineKeyboardButton{Text: "📁 Show All Topics", CallbackData: showAllCallbackData}
		rows = append(rows, []gotgbot.InlineKeyboardButton{showAllBtn})
	}

	// Add retry button
	retryCallbackData := "retry_" + strconv.FormatInt(msg.MessageId, 10)
	retryBtn := gotgbot.InlineKeyboardButton{Text: "🔄 Try Again", CallbackData: retryCallbackData}
	rows = append(rows, []gotgbot.InlineKeyboardButton{retryBtn})

	return &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}, nil
}
