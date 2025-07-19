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
var messageStore = make(map[string]*gotgbot.Message)        // callbackData -> original message
var keyboardMessageStore = make(map[string]int)             // callbackData -> keyboard message ID
var waitingForTopicName = make(map[int64]string)            // user_id -> callbackData waiting for topic name
var originalMessageStore = make(map[int64]*gotgbot.Message) // user_id -> original message for topic creation

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

					// Store the callback data to wait for topic name
					waitingForTopicName[update.CallbackQuery.From.Id] = callbackData
					// Store the original message separately for topic creation
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
							// If no keyboard message ID found, try to find any message with the same message ID
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
						}
					}
				} else if callbackData == "create_topic_menu" {
					// Show topic creation input prompt
					bot.SendMessage(originalMsg.Chat.Id, "📝 **Create New Topic**\n\nPlease send the name of the topic you want to create:", &gotgbot.SendMessageOpts{
						ParseMode: "Markdown",
					})
					// Set flag to wait for topic name
					waitingForTopicName[originalMsg.From.Id] = "create_topic"
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
											callbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
											messageStore[callbackData] = msg
											rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "➕ " + folder, CallbackData: callbackData}})
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

						// Find the topic in the database
						topics, err := GetForumTopics(botToken, originalMsg.Chat.Id)
						if err != nil {
							log.Printf("Error getting topics: %v", err)
							bot.SendMessage(originalMsg.Chat.Id, "❌ Failed to get topics.", &gotgbot.SendMessageOpts{
								MessageThreadId: originalMsg.MessageThreadId,
							})
							continue
						}

						var targetTopic *ForumTopic
						for _, topic := range topics {
							if strings.EqualFold(topic.Name, topicName) {
								targetTopic = &topic
								break
							}
						}

						if targetTopic == nil {
							bot.SendMessage(originalMsg.Chat.Id, "❌ Topic not found. Please try again.", &gotgbot.SendMessageOpts{
								MessageThreadId: originalMsg.MessageThreadId,
							})
							continue
						}

						// Copy message to the selected topic
						err = CopyMessageToTopic(botToken, originalMsg.Chat.Id, originalMsg.Chat.Id, int(originalMsg.MessageId), targetTopic.MessageThreadId)
						if err != nil {
							log.Printf("Error copying message: %v", err)
							bot.SendMessage(originalMsg.Chat.Id, "❌ Failed to move message to topic.", &gotgbot.SendMessageOpts{
								MessageThreadId: originalMsg.MessageThreadId,
							})
							continue
						}

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
				if waitingType, waiting := waitingForTopicName[update.Message.From.Id]; waiting {
					if waitingType == "create_topic" {
						// User is creating a new topic
						topicName := strings.TrimSpace(update.Message.Text)
						if topicName == "" {
							bot.SendMessage(update.Message.Chat.Id, "❌ Topic name cannot be empty. Please try again.", &gotgbot.SendMessageOpts{})
							continue
						}

						// Check if topic already exists
						topics, err := GetForumTopics(botToken, update.Message.Chat.Id)
						if err == nil {
							for _, topic := range topics {
								if strings.EqualFold(topic.Name, topicName) {
									bot.SendMessage(update.Message.Chat.Id, "❌ A topic with this name already exists. Please choose a different name.", &gotgbot.SendMessageOpts{})
									delete(waitingForTopicName, update.Message.From.Id)
									continue
								}
							}
						}

						// Create the topic
						_, err = bot.CreateForumTopic(update.Message.Chat.Id, topicName, &gotgbot.CreateForumTopicOpts{})
						if err != nil {
							log.Printf("Error creating topic: %v", err)
							bot.SendMessage(update.Message.Chat.Id, "❌ Failed to create topic. Please try again.", &gotgbot.SendMessageOpts{})
							delete(waitingForTopicName, update.Message.From.Id)
							continue
						}

						// Success message
						successMsg := fmt.Sprintf("✅ **Topic Created Successfully!**\n\n📁 **Topic Name:** %s\n\nYou can now send messages and the bot will suggest this topic when relevant.", topicName)
						bot.SendMessage(update.Message.Chat.Id, successMsg, &gotgbot.SendMessageOpts{
							ParseMode: "Markdown",
						})

						// Auto-delete success message after 1 minute
						go func(msgId int64, chatId int64) {
							time.Sleep(1 * time.Minute)
							DeleteMessage(botToken, chatId, int(msgId))
						}(int64(update.Message.MessageId), update.Message.Chat.Id)

						delete(waitingForTopicName, update.Message.From.Id)
						continue
					}
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
									callbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
									messageStore[callbackData] = msg
									rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "➕ " + folder, CallbackData: callbackData}})
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
							}(update.Message)
							continue
						}
					}
				}
			}
		}
	}
}
