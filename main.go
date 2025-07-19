package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"save-message/internal/ai"
	"save-message/internal/database"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/joho/godotenv"
)

// Store original message info for callback handling
var messageStore = make(map[string]*gotgbot.Message) // callbackData -> original message
var keyboardMessageStore = make(map[string]int)      // callbackData -> keyboard message ID
// Refactor: waitingForTopicName now stores context
// user_id -> struct with chat ID, thread ID, and original message ID

type TopicCreationContext struct {
	ChatId        int64
	ThreadId      int64
	OriginalMsgId int64
}

var waitingForTopicName = make(map[int64]TopicCreationContext) // user_id -> context
var originalMessageStore = make(map[int64]*gotgbot.Message)    // user_id -> original message for topic creation

// Add a map to track recently moved messages
var recentlyMovedMessages = make(map[int64]bool)

// Global database instance
var db *database.Database

// ForumTopic represents a Telegram forum topic
type ForumTopic struct {
	MessageThreadId int    `json:"message_thread_id"`
	Name            string `json:"name"`
}

// GetForumTopics fetches all topics in a forum
func GetForumTopics(botToken string, chatId int64) ([]ForumTopic, error) {
	// First, check if this is a forum chat
	chatUrl := fmt.Sprintf("https://api.telegram.org/bot%s/getChat?chat_id=%d", botToken, chatId)
	chatResp, err := http.Get(chatUrl)
	if err == nil {
		defer chatResp.Body.Close()
		chatBody, _ := io.ReadAll(chatResp.Body)
		log.Printf("getChat response: %s", string(chatBody))

		var chatResult struct {
			Ok     bool `json:"ok"`
			Result struct {
				Type    string `json:"type"`
				IsForum bool   `json:"is_forum"`
			} `json:"result"`
		}

		if err := json.Unmarshal(chatBody, &chatResult); err == nil && chatResult.Ok {
			log.Printf("Chat type: %s, Is forum: %v", chatResult.Result.Type, chatResult.Result.IsForum)
		}
	}

	// Try different methods to get forum topics
	methods := []string{
		"getForumTopics",
		"getForumTopicByID",
	}

	for _, method := range methods {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/%s?chat_id=%d", botToken, method, chatId)
		resp, err := http.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		log.Printf("%s response: %s", method, string(body))

		var result struct {
			Ok     bool `json:"ok"`
			Result struct {
				Topics []ForumTopic `json:"topics"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}

		if result.Ok {
			log.Printf("Successfully got topics using %s: %v", method, result.Result.Topics)
			// Update database with found topics
			for _, topic := range result.Result.Topics {
				err := db.AddTopic(chatId, topic.Name, int64(topic.MessageThreadId), 0) // 0 for system-created topics
				if err != nil {
					log.Printf("Error adding topic to database: %v", err)
				}
			}
			return result.Result.Topics, nil
		}
	}

	// If all methods fail, use database
	log.Printf("All forum topic methods failed, using database")
	dbTopics, err := db.GetTopicsByChat(chatId)
	if err != nil {
		log.Printf("Error getting topics from database: %v", err)
		return []ForumTopic{}, nil
	}

	var topics []ForumTopic
	for _, dbTopic := range dbTopics {
		topics = append(topics, ForumTopic{
			MessageThreadId: int(dbTopic.MessageThreadId), // Convert int64 to int for Telegram API
			Name:            dbTopic.Name,
		})
	}
	log.Printf("Using database topics: %v", topics)
	return topics, nil
}

// CreateForumTopic creates a new topic in a forum
func CreateForumTopic(botToken string, chatId int64, name string) (*ForumTopic, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/createForumTopic", botToken)

	requestBody := map[string]interface{}{
		"chat_id": chatId,
		"name":    name,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok     bool       `json:"ok"`
		Result ForumTopic `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.Ok {
		return nil, fmt.Errorf("failed to create topic: %s", string(body))
	}

	// Add topic to database
	err = db.AddTopic(chatId, name, int64(result.Result.MessageThreadId), 0) // 0 for system-created topics
	if err != nil {
		log.Printf("Error adding topic to database: %v", err)
	} else {
		log.Printf("Added topic '%s' to database for chat %d", name, chatId)
	}

	return &result.Result, nil
}

// CopyMessageToTopic copies a message to a specific topic
func CopyMessageToTopic(botToken string, chatId int64, fromChatId int64, messageId int, messageThreadId int) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/copyMessage", botToken)

	requestBody := map[string]interface{}{
		"chat_id":           chatId,
		"from_chat_id":      fromChatId,
		"message_id":        messageId,
		"message_thread_id": messageThreadId,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok bool `json:"ok"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if !result.Ok {
		return fmt.Errorf("failed to copy message: %s", string(body))
	}

	return nil
}

// DeleteMessage deletes a message from a chat
func DeleteMessage(botToken string, chatId int64, messageId int) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteMessage", botToken)

	requestBody := map[string]interface{}{
		"chat_id":    chatId,
		"message_id": messageId,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok bool `json:"ok"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if !result.Ok {
		return fmt.Errorf("failed to delete message: %s", string(body))
	}

	return nil
}

// Add a helper to copy a message and return the new message ID
func CopyMessageToTopicWithResult(botToken string, chatId int64, fromChatId int64, messageId int, messageThreadId int) (*gotgbot.Message, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/copyMessage", botToken)
	requestBody := map[string]interface{}{
		"chat_id":           chatId,
		"from_chat_id":      fromChatId,
		"message_id":        messageId,
		"message_thread_id": messageThreadId,
	}
	bodyBytes, _ := json.Marshal(requestBody)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Ok     bool            `json:"ok"`
		Result gotgbot.Message `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if !result.Ok {
		return nil, fmt.Errorf("failed to copy message: %s", string(body))
	}
	return &result.Result, nil
}

func main() {
	_ = godotenv.Load()
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set in .env")
	}
	if openaiKey == "" {
		log.Fatal("OPENAI_API_KEY is not set in .env")
	}

	// Initialize database
	var err error
	db, err = database.NewDatabase("bot.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	bot, err := gotgbot.NewBot(botToken, nil)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	log.Printf("Authorized on account %s", bot.User.Username)

	openaiClient := ai.NewOpenAIClient(openaiKey)
	var offset int64 = 0
	for {
		updates, err := bot.GetUpdates(&gotgbot.GetUpdatesOpts{
			Offset:  offset,
			Timeout: 10,
		})
		if err != nil {
			if !strings.Contains(err.Error(), "context deadline exceeded") {
				log.Printf("GetUpdates error: %v", err)
			}
			time.Sleep(2 * time.Second)
			continue
		}
		for _, update := range updates {
			// Always increment offset for each update to prevent infinite loops
			if update.UpdateId >= offset {
				offset = update.UpdateId + 1
			}

			// Handle callback queries (button clicks)
			if update.CallbackQuery != nil {
				log.Printf("Received callback query: %s", update.CallbackQuery.Data)
				callbackData := update.CallbackQuery.Data

				// Answer the callback query to remove the loading state
				bot.AnswerCallbackQuery(update.CallbackQuery.Id, &gotgbot.AnswerCallbackQueryOpts{
					Text: "Processing...",
				})

				// Special handling for detectMessageOnOtherTopic_ok_ callback
				if strings.HasPrefix(callbackData, "detectMessageOnOtherTopic_ok_") {
					// Handle "Ok" button for warning message about posting in non-General topics
					log.Printf("[DEBUG] Handling detectMessageOnOtherTopic_ok_ callback: %s", callbackData)

					// Delete the warning message itself (the message that contains the "Ok" button)
					err := DeleteMessage(botToken, update.CallbackQuery.Message.Chat.Id, int(update.CallbackQuery.Message.MessageId))
					if err != nil {
						log.Printf("[DEBUG] Error deleting warning message: %v", err)
					} else {
						log.Printf("[DEBUG] Successfully deleted warning message: MessageId=%d", update.CallbackQuery.Message.MessageId)
					}
					continue
				}

				originalMsg := messageStore[callbackData]
				if originalMsg == nil {
					bot.SendMessage(update.CallbackQuery.From.Id, "❌ Error: Message not found. Please try again.", nil)
					continue
				}

				if strings.HasPrefix(callbackData, "create_new_folder_") {
					// Ask user for topic name
					bot.SendMessage(originalMsg.Chat.Id, "📝 Please enter the name for your new topic:", &gotgbot.SendMessageOpts{
						MessageThreadId: originalMsg.MessageThreadId,
					})

					// Store the context for topic creation
					waitingForTopicName[update.CallbackQuery.From.Id] = TopicCreationContext{
						ChatId:        originalMsg.Chat.Id,
						ThreadId:      int64(originalMsg.MessageThreadId),
						OriginalMsgId: int64(originalMsg.MessageId),
					}
					// Store the original message for this user
					originalMessageStore[update.CallbackQuery.From.Id] = originalMsg
					// Delete the keyboard message
					if keyboardMsgId, exists := keyboardMessageStore[callbackData]; exists {
						DeleteMessage(botToken, originalMsg.Chat.Id, keyboardMsgId)
						delete(keyboardMessageStore, callbackData)
					}
				} else if strings.HasPrefix(callbackData, "retry_") {
					// Handle retry button - just send a simple retry message for now
					bot.SendMessage(originalMsg.Chat.Id, "🔄 Retrying... Please send your message again.", &gotgbot.SendMessageOpts{
						MessageThreadId: originalMsg.MessageThreadId,
					})
				} else if strings.HasPrefix(callbackData, "show_all_topics_") {
					// Show all topics from database as clickable buttons
					topics, err := GetForumTopics(botToken, originalMsg.Chat.Id)
					if err != nil {
						log.Printf("Error getting topics: %v", err)
						bot.SendMessage(originalMsg.Chat.Id, "❌ Failed to get topics. Please try again.", &gotgbot.SendMessageOpts{
							MessageThreadId: originalMsg.MessageThreadId,
						})
						continue
					}

					if len(topics) == 0 {
						bot.SendMessage(originalMsg.Chat.Id, "📁 No topics discovered yet. Create some topics and the bot will remember them!", &gotgbot.SendMessageOpts{
							MessageThreadId: originalMsg.MessageThreadId,
						})
					} else {
						// Build keyboard with all existing topics
						var rows [][]gotgbot.InlineKeyboardButton

						// Add all existing topics as buttons
						for _, topic := range topics {
							callbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
							messageStore[callbackData] = originalMsg
							rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "📁 " + topic.Name, CallbackData: callbackData}})
						}

						// Add back button
						backCallbackData := "back_to_suggestions_" + strconv.FormatInt(originalMsg.MessageId, 10)
						messageStore[backCallbackData] = originalMsg
						backBtn := gotgbot.InlineKeyboardButton{Text: "⬅️ Back to Suggestions", CallbackData: backCallbackData}
						rows = append(rows, []gotgbot.InlineKeyboardButton{backBtn})

						keyboard := &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}

						// Always try to update the existing message first
						if keyboardMsgId, exists := keyboardMessageStore[callbackData]; exists {
							_, _, err = bot.EditMessageText("Choose from all existing topics:", &gotgbot.EditMessageTextOpts{
								ChatId:      originalMsg.Chat.Id,
								MessageId:   int64(keyboardMsgId),
								ReplyMarkup: *keyboard,
							})
							if err != nil {
								log.Printf("Error updating message with all topics: %v", err)
								// If update fails, try to find the message by searching through all stored keyboard messages
								for storedCallback, storedMsgId := range keyboardMessageStore {
									if strings.Contains(storedCallback, strconv.FormatInt(originalMsg.MessageId, 10)) {
										_, _, updateErr := bot.EditMessageText("Choose from all existing topics:", &gotgbot.EditMessageTextOpts{
											ChatId:      originalMsg.Chat.Id,
											MessageId:   int64(storedMsgId),
											ReplyMarkup: *keyboard,
										})
										if updateErr == nil {
											// Store the keyboard message ID for all topic buttons
											for _, topic := range topics {
												topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
												keyboardMessageStore[topicCallbackData] = storedMsgId
											}
											keyboardMessageStore[backCallbackData] = storedMsgId
											break
										}
									}
								}
							} else {
								// Store the keyboard message ID for all topic buttons
								for _, topic := range topics {
									topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
									keyboardMessageStore[topicCallbackData] = keyboardMsgId
								}
								keyboardMessageStore[backCallbackData] = keyboardMsgId
							}
						} else {
							// Send new message with all topics
							newMsg, err := bot.SendMessage(originalMsg.Chat.Id, "Choose from all existing topics:", &gotgbot.SendMessageOpts{
								MessageThreadId: originalMsg.MessageThreadId,
								ReplyMarkup:     *keyboard,
							})
							if err != nil {
								log.Printf("Error sending message with all topics: %v", err)
							} else {
								// Store the keyboard message ID for all topic buttons
								for _, topic := range topics {
									topicCallbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
									keyboardMessageStore[topicCallbackData] = int(newMsg.MessageId)
								}
								keyboardMessageStore[backCallbackData] = int(newMsg.MessageId)
							}
						}
					}
				} else if callbackData == "create_topic_menu" {
					// Show topic creation input prompt
					bot.SendMessage(originalMsg.Chat.Id, "📝 **Create New Topic**\n\nPlease send the name of the topic you want to create:", &gotgbot.SendMessageOpts{
						ParseMode: "Markdown",
					})
					// Set flag to wait for topic name
					waitingForTopicName[originalMsg.From.Id] = TopicCreationContext{
						ChatId:        originalMsg.Chat.Id,
						ThreadId:      int64(originalMsg.MessageThreadId),
						OriginalMsgId: int64(originalMsg.MessageId),
					}
				} else if callbackData == "show_all_topics_menu" {
					// Show all topics from database
					topics, err := GetForumTopics(botToken, originalMsg.Chat.Id)
					if err != nil {
						log.Printf("Error getting topics: %v", err)
						bot.SendMessage(originalMsg.Chat.Id, "❌ Failed to get topics.", &gotgbot.SendMessageOpts{})
						return
					}

					if len(topics) == 0 {
						bot.SendMessage(originalMsg.Chat.Id, "📁 No topics found yet. Send a message to create your first topic!", &gotgbot.SendMessageOpts{})
					} else {
						topicList := "📁 **Your Topics:**\n"
						for _, topic := range topics {
							topicList += "• " + topic.Name + "\n"
						}
						bot.SendMessage(originalMsg.Chat.Id, topicList, &gotgbot.SendMessageOpts{
							ParseMode: "Markdown",
						})
					}
				} else if strings.HasPrefix(callbackData, "back_to_suggestions_") {
					// Go back to AI suggestions
					parts := strings.Split(callbackData, "_")
					if len(parts) >= 4 {
						messageId, err := strconv.ParseInt(parts[3], 10, 64)
						if err == nil {
							// Find the original message and reprocess it
							for _, storedMsg := range messageStore {
								if storedMsg.MessageId == messageId {
									// Reprocess the original message to show AI suggestions
									go func(msg *gotgbot.Message) {
										// Send waiting message first
										waitingMsg, err := bot.SendMessage(msg.Chat.Id, "🤔 Thinking...", &gotgbot.SendMessageOpts{
											MessageThreadId: msg.MessageThreadId,
										})
										if err != nil {
											log.Printf("Error sending waiting message: %v", err)
											return
										}

										ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
										defer cancel()

										// Get existing topics from database
										topics, err := GetForumTopics(botToken, msg.Chat.Id)
										existingFolders := []string{}
										if err == nil {
											for _, topic := range topics {
												existingFolders = append(existingFolders, topic.Name)
											}
										}

										suggestions, err := openaiClient.SuggestFolders(ctx, msg.Text, existingFolders)
										if err != nil {
											log.Printf("OpenAI error: %v", err)
											// Update waiting message with error and retry button
											retryKeyboard := &gotgbot.InlineKeyboardMarkup{
												InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
													{{Text: "🔄 Try Again", CallbackData: "retry_" + strconv.FormatInt(msg.MessageId, 10)}},
												},
											}
											_, _, err = bot.EditMessageText("Sorry, I couldn't suggest folders right now.", &gotgbot.EditMessageTextOpts{
												ChatId:      msg.Chat.Id,
												MessageId:   waitingMsg.MessageId,
												ReplyMarkup: *retryKeyboard,
											})
											if err != nil {
												log.Printf("Error updating waiting message: %v", err)
											}
											return
										}
										log.Printf("OpenAI suggestions: %v", suggestions)

										// Build inline keyboard
										var rows [][]gotgbot.InlineKeyboardButton

										// Separate existing and new topics
										var existingTopics []string
										var newTopics []string

										log.Printf("Available topics: %v", topics)
										log.Printf("AI suggestions: %v", suggestions)

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
												log.Printf("Skipping General topic")
												continue
											}

											if isExisting {
												log.Printf("Found existing topic: %s (original: %s)", folder, existingTopicName)
												existingTopics = append(existingTopics, existingTopicName) // Use exact name
											} else {
												log.Printf("New topic suggested: %s", folder)
												newTopics = append(newTopics, folder)
											}
										}

										log.Printf("Existing topics to show: %v", existingTopics)
										log.Printf("New topics to show: %v", newTopics)

										// Add existing topics with folder icon
										for _, folder := range existingTopics {
											callbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
											messageStore[callbackData] = msg
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
											messageStore[callbackData] = msg
											rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "➕ " + cleanFolder, CallbackData: callbackData}})
										}

										// Add create new folder option
										createCallbackData := "create_new_folder_" + strconv.FormatInt(msg.MessageId, 10)
										messageStore[createCallbackData] = msg
										createBtn := gotgbot.InlineKeyboardButton{Text: "📝 Create Custom Topic", CallbackData: createCallbackData}
										rows = append(rows, []gotgbot.InlineKeyboardButton{createBtn})

										// Add show all topics button if there are existing topics
										topics, err = GetForumTopics(botToken, msg.Chat.Id)
										if err == nil && len(topics) > 0 {
											showAllCallbackData := "show_all_topics_" + strconv.FormatInt(msg.MessageId, 10)
											messageStore[showAllCallbackData] = msg
											showAllBtn := gotgbot.InlineKeyboardButton{Text: "📁 Show All Topics", CallbackData: showAllCallbackData}
											rows = append(rows, []gotgbot.InlineKeyboardButton{showAllBtn})
										}

										keyboard := &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}

										// Update waiting message with keyboard
										_, _, err = bot.EditMessageText("Choose a folder:", &gotgbot.EditMessageTextOpts{
											ChatId:      msg.Chat.Id,
											MessageId:   waitingMsg.MessageId,
											ReplyMarkup: *keyboard,
										})
										if err != nil {
											log.Printf("Error updating message with keyboard: %v", err)
										} else {
											// Store the keyboard message ID for each callback data
											keyboardMsgId := int(waitingMsg.MessageId)
											for _, folder := range suggestions {
												callbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
												keyboardMessageStore[callbackData] = keyboardMsgId
											}
											createCallbackData := "create_new_folder_" + strconv.FormatInt(msg.MessageId, 10)
											keyboardMessageStore[createCallbackData] = keyboardMsgId
										}
									}(storedMsg)
									break
								}
							}
						}
					}
				} else {
					// Handle topic selection (both from AI suggestions and Show All Topics)
					// Check if this is a topic selection callback (format: "TopicName_MessageId")
					parts := strings.Split(callbackData, "_")
					if len(parts) >= 2 {
						// Try to find the original message by searching through all stored messages
						var originalMsg *gotgbot.Message
						var messageId int64

						// Extract message ID from the last part
						if msgId, err := strconv.ParseInt(parts[len(parts)-1], 10, 64); err == nil {
							messageId = msgId
							// Find the original message by message ID
							for _, storedMsg := range messageStore {
								if storedMsg.MessageId == messageId {
									originalMsg = storedMsg
									break
								}
							}
						}

						if originalMsg == nil {
							bot.SendMessage(update.CallbackQuery.From.Id, "❌ Error: Original message not found. Please try again.", nil)
							continue
						}

						// Extract topic name (everything except the last part which is message ID)
						topicName := strings.Join(parts[:len(parts)-1], "_")
						log.Printf("[DEBUG] Topic selection callback: topicName='%s', callbackData='%s'", topicName, callbackData)

						// Find the topic in the database
						topics, err := GetForumTopics(botToken, originalMsg.Chat.Id)
						if err != nil {
							log.Printf("[DEBUG] Error getting topics: %v", err)
							bot.SendMessage(originalMsg.Chat.Id, "❌ Failed to get topics.", &gotgbot.SendMessageOpts{
								MessageThreadId: originalMsg.MessageThreadId,
							})
							continue
						}
						log.Printf("[DEBUG] Available topics in database: %v", topics)

						var targetTopic *ForumTopic
						for _, topic := range topics {
							if strings.EqualFold(topic.Name, topicName) {
								targetTopic = &topic
								log.Printf("[DEBUG] Found existing topic: %s (MessageThreadId: %d)", topic.Name, topic.MessageThreadId)
								break
							}
						}

						// If topic doesn't exist, create it
						if targetTopic == nil {
							log.Printf("[DEBUG] Topic '%s' not found in database, creating new topic", topicName)

							// Create the new topic
							newTopic, err := CreateForumTopic(botToken, originalMsg.Chat.Id, topicName)
							if err != nil {
								log.Printf("[DEBUG] Error creating new topic '%s': %v", topicName, err)
								bot.SendMessage(originalMsg.Chat.Id, "❌ Failed to create new topic.", &gotgbot.SendMessageOpts{
									MessageThreadId: originalMsg.MessageThreadId,
								})
								continue
							}

							log.Printf("[DEBUG] Successfully created new topic: %s (MessageThreadId: %d)", newTopic.Name, newTopic.MessageThreadId)
							targetTopic = newTopic
						}

						// Copy message to the selected topic
						log.Printf("[DEBUG] Copying message to topic: MessageId=%d, TopicName=%s, MessageThreadId=%d",
							originalMsg.MessageId, targetTopic.Name, targetTopic.MessageThreadId)

						err = CopyMessageToTopic(botToken, originalMsg.Chat.Id, originalMsg.Chat.Id, int(originalMsg.MessageId), targetTopic.MessageThreadId)
						if err != nil {
							log.Printf("[DEBUG] Error copying message to topic: %v", err)
							bot.SendMessage(originalMsg.Chat.Id, "❌ Failed to move message to topic.", &gotgbot.SendMessageOpts{
								MessageThreadId: originalMsg.MessageThreadId,
							})
							continue
						}
						log.Printf("[DEBUG] Successfully copied message to topic: MessageId=%d, TopicName=%s", originalMsg.MessageId, targetTopic.Name)

						// Delete the original user message from General topic
						DeleteMessage(botToken, originalMsg.Chat.Id, int(originalMsg.MessageId))

						// Show success message with message preview
						messagePreview := originalMsg.Text
						if len(messagePreview) > 100 {
							messagePreview = messagePreview[:100] + "..."
						}

						successMsg := fmt.Sprintf("✅ **Message moved to '%s'!**\n\n📝 Preview: %s", targetTopic.Name, messagePreview)
						successResponse, err := bot.SendMessage(originalMsg.Chat.Id, successMsg, &gotgbot.SendMessageOpts{
							MessageThreadId: originalMsg.MessageThreadId,
							ParseMode:       "Markdown",
						})

						// Auto-delete success message after 1 minute
						if err == nil {
							go func(msgId int, chatId int64) {
								time.Sleep(60 * time.Second)
								DeleteMessage(botToken, chatId, msgId)
							}(int(successResponse.MessageId), originalMsg.Chat.Id)
						}

						// Delete the keyboard message
						if keyboardMsgId, exists := keyboardMessageStore[callbackData]; exists {
							DeleteMessage(botToken, originalMsg.Chat.Id, keyboardMsgId)
							delete(keyboardMessageStore, callbackData)
						}

						// Clean up message store
						delete(messageStore, callbackData)
					}
				}

				// Clean up the stored message
				delete(messageStore, callbackData)
				continue
			}

			// Handle messages
			if update.Message != nil {
				// Check if the bot was just added to a group
				botJustJoined := false
				if update.Message.NewChatMembers != nil && len(update.Message.NewChatMembers) > 0 {
					for _, member := range update.Message.NewChatMembers {
						if member.Id == bot.User.Id {
							botJustJoined = true
							break
						}
					}
				}
				if botJustJoined {
					welcome := "It helps you organize your saved messages using Topics and smart suggestions — without using any commands.\nYou can categorize, edit, and retrieve your notes easily with inline buttons.\n\n🛡️ 100% private: all your content stays inside Telegram.\n\nJust write — we'll handle the rest.\n\nFor more info, send /help."
					bot.SendMessage(update.Message.Chat.Id, welcome, &gotgbot.SendMessageOpts{})
					continue
				}

				// Check if message is NOT in General topic (thread 0) - only allow messages in General
				if update.Message.MessageThreadId != 0 {
					log.Printf("[DEBUG] Message detected in non-General topic: ThreadId=%d, MessageId=%d, Text=%s",
						update.Message.MessageThreadId, update.Message.MessageId, update.Message.Text)

					// Delete the user's message immediately
					err := DeleteMessage(botToken, update.Message.Chat.Id, int(update.Message.MessageId))
					if err != nil {
						log.Printf("[DEBUG] Error deleting message from non-General topic: %v", err)
					} else {
						log.Printf("[DEBUG] Successfully deleted message from non-General topic: MessageId=%d", update.Message.MessageId)
					}

					// Send warning message with "Ok" button
					callbackData := "detectMessageOnOtherTopic_ok_" + strconv.FormatInt(update.Message.MessageId, 10)
					keyboard := &gotgbot.InlineKeyboardMarkup{
						InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
							{{Text: "Ok", CallbackData: callbackData}},
						},
					}

					warningMsg, err := bot.SendMessage(update.Message.Chat.Id,
						"⚠️ **Please send messages only in the General topic!**\n\nThis message will be removed automatically in 1 minute.",
						&gotgbot.SendMessageOpts{
							MessageThreadId: update.Message.MessageThreadId,
							ParseMode:       "Markdown",
							ReplyMarkup:     *keyboard,
						})

					if err != nil {
						log.Printf("[DEBUG] Error sending warning message: %v", err)
					} else {
						log.Printf("[DEBUG] Successfully sent warning message: MessageId=%d", warningMsg.MessageId)

						// Auto-delete warning message after 1 minute
						go func(botToken string, chatId int64, messageId int64, threadId int64) {
							time.Sleep(60 * time.Second)
							err := DeleteMessage(botToken, chatId, int(messageId))
							if err != nil {
								log.Printf("[DEBUG] Error auto-deleting warning message: %v", err)
							} else {
								log.Printf("[DEBUG] Successfully auto-deleted warning message: MessageId=%d", messageId)
							}
						}(botToken, update.Message.Chat.Id, warningMsg.MessageId, update.Message.MessageThreadId)
					}

					// Prevent further processing of this message
					continue
				}

				// Only run AI suggestions for General topic (thread 0)
				if update.Message.MessageThreadId != 0 {
					log.Printf("Skipping AI for non-General topic: thread %d", update.Message.MessageThreadId)
					continue
				}
				if recentlyMovedMessages[update.Message.MessageId] {
					log.Printf("Skipping AI for recently moved message: %d", update.Message.MessageId)
					delete(recentlyMovedMessages, update.Message.MessageId)
					continue
				}
				// Fallback: skip AI for first message in a new topic with empty text
				if update.Message.MessageThreadId != 0 && strings.TrimSpace(update.Message.Text) == "" && update.Message.MessageId < 10 {
					log.Printf("Skipping AI for first message in new topic (empty text): %d", update.Message.MessageId)
					continue
				}
				log.Printf("Received message: %s (ChatType: %s, ThreadId: %d)", update.Message.Text, update.Message.Chat.Type, update.Message.MessageThreadId)

				// Check if the message mentions the bot (handle both possible usernames)
				messageText := strings.ToLower(update.Message.Text)
				if update.Message.Text != "" && (strings.Contains(messageText, "@savemessagbot") || strings.Contains(messageText, "@savemessagebot")) {
					// Show topic creation menu when bot is mentioned
					keyboard := &gotgbot.InlineKeyboardMarkup{
						InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
							{{Text: "📝 Create New Topic", CallbackData: "create_topic_menu"}},
							{{Text: "📁 Show All Topics", CallbackData: "show_all_topics_menu"}},
						},
					}
					bot.SendMessage(update.Message.Chat.Id, "🤖 **Bot Menu**\n\nWhat would you like to do?", &gotgbot.SendMessageOpts{
						ParseMode:   "Markdown",
						ReplyMarkup: *keyboard,
					})
					continue
				}

				// Check if user is waiting to provide a topic name
				if ctx, waiting := waitingForTopicName[update.Message.From.Id]; waiting {
					topicName := strings.TrimSpace(update.Message.Text)
					if topicName == "" {
						bot.SendMessage(update.Message.Chat.Id, "❌ Topic name cannot be empty. Please try again.", &gotgbot.SendMessageOpts{})
						continue
					}
					// Check if topic already exists
					topics, err := GetForumTopics(botToken, ctx.ChatId)
					if err == nil {
						exists := false
						for _, topic := range topics {
							if strings.EqualFold(topic.Name, topicName) {
								exists = true
								break
							}
						}
						if exists {
							bot.SendMessage(ctx.ChatId, "❌ A topic with this name already exists. Please choose a different name.", &gotgbot.SendMessageOpts{})
							delete(waitingForTopicName, update.Message.From.Id)
							delete(originalMessageStore, update.Message.From.Id)
							continue
						}
					}
					// Create the topic in the correct chat/thread
					newTopic, err := CreateForumTopic(botToken, ctx.ChatId, topicName)
					if err != nil {
						log.Printf("Error creating topic: %v", err)
						bot.SendMessage(ctx.ChatId, "❌ Failed to create topic. Please try again.", &gotgbot.SendMessageOpts{})
						delete(waitingForTopicName, update.Message.From.Id)
						delete(originalMessageStore, update.Message.From.Id)
						continue
					}
					// After creating the topic and before copying the original user message, send the topic name as the first message in the new topic
					if newTopic != nil {
						_, err := bot.SendMessage(ctx.ChatId, newTopic.Name, &gotgbot.SendMessageOpts{
							MessageThreadId: int64(newTopic.MessageThreadId),
						})
						if err != nil {
							log.Printf("Error sending topic name as first message: %v", err)
						}
					}
					// Copy the original user message to the new topic
					if origMsg, ok := originalMessageStore[update.Message.From.Id]; ok && newTopic != nil {
						// Copy the message and get the new message ID
						_, err := CopyMessageToTopicWithResult(botToken, ctx.ChatId, origMsg.Chat.Id, int(origMsg.MessageId), newTopic.MessageThreadId)
						if err != nil {
							log.Printf("Error copying message to new topic: %v", err)
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
							bot.SendMessage(ctx.ChatId, confirmMsg, &gotgbot.SendMessageOpts{
								MessageThreadId: 0,
							})
							// Delete the original message from General after a short delay
							go func(botToken string, chatId int64, messageId int) {
								time.Sleep(1 * time.Second)
								_ = DeleteMessage(botToken, chatId, messageId)
							}(botToken, origMsg.Chat.Id, int(origMsg.MessageId))
						}
					}
					// Clean up state
					delete(waitingForTopicName, update.Message.From.Id)
					delete(originalMessageStore, update.Message.From.Id)
					// Prevent further processing of this message
					continue
				}

				// Handle commands and regular messages
				switch update.Message.Text {
				case "/start":
					welcome := "Save Message is your personal assistant inside Telegram.\n\nIt helps you organize your saved messages using Topics and smart suggestions — without using any commands.\nYou can categorize, edit, and retrieve your notes easily with inline buttons.\n\n🛡️ 100% private: all your content stays inside Telegram.\n\nJust write — we'll handle the rest.\n\nFor more info, send /help."
					bot.SendMessage(update.Message.Chat.Id, welcome, nil)
				case "/help":
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

					bot.SendMessage(update.Message.Chat.Id, helpText, &gotgbot.SendMessageOpts{
						ParseMode: "Markdown",
					})
				case "/topics":
					// List all topics for this chat
					topics, err := GetForumTopics(botToken, update.Message.Chat.Id)
					if err != nil {
						log.Printf("Error getting topics: %v", err)
						bot.SendMessage(update.Message.Chat.Id, "❌ Failed to get topics.", &gotgbot.SendMessageOpts{})
						continue
					}

					if len(topics) == 0 {
						bot.SendMessage(update.Message.Chat.Id, "📁 No topics found yet. Send a message to create your first topic!", &gotgbot.SendMessageOpts{})
					} else {
						topicList := "📁 **Your Topics:**\n"
						for _, topic := range topics {
							topicList += "• " + topic.Name + "\n"
						}
						bot.SendMessage(update.Message.Chat.Id, topicList, &gotgbot.SendMessageOpts{
							ParseMode: "Markdown",
						})
					}
				case "/addtopic":
					// Show topic creation menu
					keyboard := &gotgbot.InlineKeyboardMarkup{
						InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
							{{Text: "📝 Create New Topic", CallbackData: "create_topic_menu"}},
						},
					}
					bot.SendMessage(update.Message.Chat.Id, "Choose an option:", &gotgbot.SendMessageOpts{
						ReplyMarkup: *keyboard,
					})
				default:
					// Handle regular messages (not commands)
					// Process topic suggestions for regular messages
					if update.Message.Chat.Type == "supergroup" {
						// Check if this is a forum chat
						chat, err := bot.GetChat(update.Message.Chat.Id, &gotgbot.GetChatOpts{})
						if err != nil {
							log.Printf("Error getting chat info: %v", err)
							continue
						}

						if chat.IsForum {
							log.Printf("Processing topic message: %s", update.Message.Text)
							// Process the message asynchronously
							go func(msg *gotgbot.Message) {
								// Send waiting message first
								waitingMsg, err := bot.SendMessage(msg.Chat.Id, "🤔 Thinking...", &gotgbot.SendMessageOpts{
									MessageThreadId: msg.MessageThreadId,
								})
								if err != nil {
									log.Printf("Error sending waiting message: %v", err)
									return
								}

								ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
								defer cancel()

								// Get existing topics from database
								topics, err := GetForumTopics(botToken, msg.Chat.Id)
								existingFolders := []string{}
								if err == nil {
									for _, topic := range topics {
										existingFolders = append(existingFolders, topic.Name)
									}
								}

								suggestions, err := openaiClient.SuggestFolders(ctx, msg.Text, existingFolders)
								if err != nil {
									log.Printf("OpenAI error: %v", err)
									// Update waiting message with error and retry button
									retryKeyboard := &gotgbot.InlineKeyboardMarkup{
										InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
											{{Text: "🔄 Try Again", CallbackData: "retry_" + strconv.FormatInt(msg.MessageId, 10)}},
										},
									}
									_, _, err = bot.EditMessageText("Sorry, I couldn't suggest folders right now.", &gotgbot.EditMessageTextOpts{
										ChatId:      msg.Chat.Id,
										MessageId:   waitingMsg.MessageId,
										ReplyMarkup: *retryKeyboard,
									})
									if err != nil {
										log.Printf("Error updating waiting message: %v", err)
									}
									return
								}
								log.Printf("OpenAI suggestions: %v", suggestions)

								// Build inline keyboard
								var rows [][]gotgbot.InlineKeyboardButton

								// Separate existing and new topics
								var existingTopics []string
								var newTopics []string

								log.Printf("Available topics: %v", topics)
								log.Printf("AI suggestions: %v", suggestions)

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
										log.Printf("Skipping General topic")
										continue
									}

									if isExisting {
										log.Printf("Found existing topic: %s (original: %s)", folder, existingTopicName)
										existingTopics = append(existingTopics, existingTopicName) // Use exact name
									} else {
										log.Printf("New topic suggested: %s", folder)
										newTopics = append(newTopics, folder)
									}
								}

								log.Printf("Existing topics to show: %v", existingTopics)
								log.Printf("New topics to show: %v", newTopics)

								// Add existing topics with folder icon
								for _, folder := range existingTopics {
									callbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
									messageStore[callbackData] = msg
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
									messageStore[callbackData] = msg
									rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "➕ " + cleanFolder, CallbackData: callbackData}})
								}

								// Add create new folder option
								createCallbackData := "create_new_folder_" + strconv.FormatInt(msg.MessageId, 10)
								messageStore[createCallbackData] = msg
								createBtn := gotgbot.InlineKeyboardButton{Text: "📝 Create Custom Topic", CallbackData: createCallbackData}
								rows = append(rows, []gotgbot.InlineKeyboardButton{createBtn})

								// Add show all topics button if there are existing topics
								topics, err = GetForumTopics(botToken, msg.Chat.Id)
								showAllCallbackData := ""
								if err == nil && len(topics) > 0 {
									showAllCallbackData = "show_all_topics_" + strconv.FormatInt(msg.MessageId, 10)
									messageStore[showAllCallbackData] = msg
									showAllBtn := gotgbot.InlineKeyboardButton{Text: "📁 Show All Topics", CallbackData: showAllCallbackData}
									rows = append(rows, []gotgbot.InlineKeyboardButton{showAllBtn})
								}

								// Add retry button
								retryCallbackData := "retry_" + strconv.FormatInt(msg.MessageId, 10)
								messageStore[retryCallbackData] = msg
								retryBtn := gotgbot.InlineKeyboardButton{Text: "🔄 Try Again", CallbackData: retryCallbackData}
								rows = append(rows, []gotgbot.InlineKeyboardButton{retryBtn})

								keyboard := &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}

								// Always try to update the existing message first
								callbackData := "suggestions_" + strconv.FormatInt(msg.MessageId, 10)
								if keyboardMsgId, exists := keyboardMessageStore[callbackData]; exists {
									_, _, err = bot.EditMessageText("Choose a folder:", &gotgbot.EditMessageTextOpts{
										ChatId:      msg.Chat.Id,
										MessageId:   int64(keyboardMsgId),
										ReplyMarkup: *keyboard,
									})
									if err != nil {
										log.Printf("Error updating message with suggestions: %v", err)
										// If update fails, try to find the message by searching through all stored keyboard messages
										for storedCallback, storedMsgId := range keyboardMessageStore {
											if strings.Contains(storedCallback, strconv.FormatInt(msg.MessageId, 10)) {
												_, _, updateErr := bot.EditMessageText("Choose a folder:", &gotgbot.EditMessageTextOpts{
													ChatId:      msg.Chat.Id,
													MessageId:   int64(storedMsgId),
													ReplyMarkup: *keyboard,
												})
												if updateErr == nil {
													// Store the keyboard message ID for all suggestion buttons
													for _, folder := range existingTopics {
														folderCallbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
														keyboardMessageStore[folderCallbackData] = storedMsgId
													}
													for _, folder := range newTopics {
														cleanFolder := strings.TrimSpace(folder)
														if len(cleanFolder) > 0 && len(cleanFolder) <= 50 && !strings.Contains(cleanFolder, "\n") {
															folderCallbackData := cleanFolder + "_" + strconv.FormatInt(msg.MessageId, 10)
															keyboardMessageStore[folderCallbackData] = storedMsgId
														}
													}
													keyboardMessageStore[createCallbackData] = storedMsgId
													if len(topics) > 0 {
														keyboardMessageStore[showAllCallbackData] = storedMsgId
													}
													keyboardMessageStore[retryCallbackData] = storedMsgId
													break
												}
											}
										}
									} else {
										// Store the keyboard message ID for all suggestion buttons
										for _, folder := range existingTopics {
											folderCallbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
											keyboardMessageStore[folderCallbackData] = keyboardMsgId
										}
										for _, folder := range newTopics {
											cleanFolder := strings.TrimSpace(folder)
											if len(cleanFolder) > 0 && len(cleanFolder) <= 50 && !strings.Contains(cleanFolder, "\n") {
												folderCallbackData := cleanFolder + "_" + strconv.FormatInt(msg.MessageId, 10)
												keyboardMessageStore[folderCallbackData] = keyboardMsgId
											}
										}
										keyboardMessageStore[createCallbackData] = keyboardMsgId
										if len(topics) > 0 {
											keyboardMessageStore[showAllCallbackData] = keyboardMsgId
										}
										keyboardMessageStore[retryCallbackData] = keyboardMsgId
									}
								} else {
									// Send new message with suggestions
									newMsg, err := bot.SendMessage(msg.Chat.Id, "Choose a folder:", &gotgbot.SendMessageOpts{
										MessageThreadId: msg.MessageThreadId,
										ReplyMarkup:     *keyboard,
									})
									if err != nil {
										log.Printf("Error sending message with suggestions: %v", err)
									} else {
										// Store the keyboard message ID for all suggestion buttons
										for _, folder := range existingTopics {
											folderCallbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
											keyboardMessageStore[folderCallbackData] = int(newMsg.MessageId)
										}
										for _, folder := range newTopics {
											cleanFolder := strings.TrimSpace(folder)
											if len(cleanFolder) > 0 && len(cleanFolder) <= 50 && !strings.Contains(cleanFolder, "\n") {
												folderCallbackData := cleanFolder + "_" + strconv.FormatInt(msg.MessageId, 10)
												keyboardMessageStore[folderCallbackData] = int(newMsg.MessageId)
											}
										}
										keyboardMessageStore[createCallbackData] = int(newMsg.MessageId)
										if len(topics) > 0 {
											keyboardMessageStore[showAllCallbackData] = int(newMsg.MessageId)
										}
										keyboardMessageStore[retryCallbackData] = int(newMsg.MessageId)
									}
								}
							}(update.Message)
						} else {
							log.Printf("Skipping AI for non-General topic: thread %d", update.Message.MessageThreadId)
						}
					}
				}
			}
		}
	}
}
