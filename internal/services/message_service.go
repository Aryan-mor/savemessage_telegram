package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"save-message/internal/database"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// MessageService handles all message-related operations
type MessageService struct {
	BotToken string
	db       *database.Database
}

// NewMessageService creates a new message service
func NewMessageService(botToken string, db *database.Database) *MessageService {
	return &MessageService{
		BotToken: botToken,
		db:       db,
	}
}

// DeleteMessage deletes a message from a chat
func (ms *MessageService) DeleteMessage(chatID int64, messageID int) error {
	log.Printf("[MessageService] Deleting message: ChatID=%d, MessageID=%d", chatID, messageID)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteMessage", ms.BotToken)

	requestBody := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		log.Printf("[MessageService] Error creating delete request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[MessageService] Error executing delete request: %v", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok bool `json:"ok"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[MessageService] Error parsing delete response: %v", err)
		return err
	}

	if !result.Ok {
		log.Printf("[MessageService] Failed to delete message: %s", string(body))
		return fmt.Errorf("failed to delete message: %s", string(body))
	}

	log.Printf("[MessageService] Successfully deleted message: ChatID=%d, MessageID=%d", chatID, messageID)
	return nil
}

// CopyMessageToTopic copies a message to a specific topic
func (ms *MessageService) CopyMessageToTopic(chatID int64, fromChatID int64, messageID int, messageThreadID int) error {
	log.Printf("[MessageService] Copying message to topic: ChatID=%d, FromChatID=%d, MessageID=%d, ThreadID=%d",
		chatID, fromChatID, messageID, messageThreadID)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/copyMessage", ms.BotToken)

	requestBody := map[string]interface{}{
		"chat_id":           chatID,
		"from_chat_id":      fromChatID,
		"message_id":        messageID,
		"message_thread_id": messageThreadID,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		log.Printf("[MessageService] Error creating copy request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[MessageService] Error executing copy request: %v", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok bool `json:"ok"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[MessageService] Error parsing copy response: %v", err)
		return err
	}

	if !result.Ok {
		log.Printf("[MessageService] Failed to copy message: %s", string(body))
		return fmt.Errorf("failed to copy message: %s", string(body))
	}

	log.Printf("[MessageService] Successfully copied message to topic: ChatID=%d, MessageID=%d, ThreadID=%d",
		chatID, messageID, messageThreadID)
	return nil
}

// CopyMessageToTopicWithResult copies a message to a topic and returns the new message
func (ms *MessageService) CopyMessageToTopicWithResult(chatID int64, fromChatID int64, messageID int, messageThreadID int) (*gotgbot.Message, error) {
	log.Printf("[MessageService] Copying message to topic with result: ChatID=%d, FromChatID=%d, MessageID=%d, ThreadID=%d",
		chatID, fromChatID, messageID, messageThreadID)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/copyMessage", ms.BotToken)

	requestBody := map[string]interface{}{
		"chat_id":           chatID,
		"from_chat_id":      fromChatID,
		"message_id":        messageID,
		"message_thread_id": messageThreadID,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		log.Printf("[MessageService] Error creating copy request: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[MessageService] Error executing copy request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok     bool            `json:"ok"`
		Result gotgbot.Message `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[MessageService] Error parsing copy response: %v", err)
		return nil, err
	}

	if !result.Ok {
		log.Printf("[MessageService] Failed to copy message: %s", string(body))
		return nil, fmt.Errorf("failed to copy message: %s", string(body))
	}

	log.Printf("[MessageService] Successfully copied message to topic with result: ChatID=%d, MessageID=%d, ThreadID=%d",
		chatID, result.Result.MessageId, messageThreadID)
	return &result.Result, nil
}

// SendMessage sends a message to a chat
func (ms *MessageService) SendMessage(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
	log.Printf("[MessageService] Sending message: ChatID=%d, Text=%s", chatID, text)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", ms.BotToken)

	requestBody := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	if opts != nil {
		if opts.ParseMode != "" {
			requestBody["parse_mode"] = opts.ParseMode
		}
		if opts.MessageThreadId != 0 {
			requestBody["message_thread_id"] = opts.MessageThreadId
		}
		if opts.ReplyMarkup != nil {
			requestBody["reply_markup"] = opts.ReplyMarkup
		}
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		log.Printf("[MessageService] Error creating send request: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[MessageService] Error executing send request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok     bool            `json:"ok"`
		Result gotgbot.Message `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[MessageService] Error parsing send response: %v", err)
		return nil, err
	}

	if !result.Ok {
		log.Printf("[MessageService] Failed to send message: %s", string(body))
		return nil, fmt.Errorf("failed to send message: %s", string(body))
	}

	log.Printf("[MessageService] Successfully sent message: ChatID=%d, MessageID=%d", chatID, result.Result.MessageId)
	return &result.Result, nil
}

// EditMessageText edits a message's text
func (ms *MessageService) EditMessageText(chatID int64, messageID int64, text string, opts *gotgbot.EditMessageTextOpts) (*gotgbot.Message, error) {
	log.Printf("[MessageService] Editing message text: ChatID=%d, MessageID=%d, Text=%s", chatID, messageID, text)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", ms.BotToken)

	requestBody := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}

	if opts != nil {
		if opts.ParseMode != "" {
			requestBody["parse_mode"] = opts.ParseMode
		}
		// ReplyMarkup is not a pointer, so we can't check for nil
		// We'll include it if it's not the zero value
		if opts.ReplyMarkup.InlineKeyboard != nil {
			requestBody["reply_markup"] = opts.ReplyMarkup
		}
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		log.Printf("[MessageService] Error creating edit request: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[MessageService] Error executing edit request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok     bool            `json:"ok"`
		Result gotgbot.Message `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[MessageService] Error parsing edit response: %v", err)
		return nil, err
	}

	if !result.Ok {
		log.Printf("[MessageService] Failed to edit message: %s", string(body))
		return nil, fmt.Errorf("failed to edit message: %s", string(body))
	}

	log.Printf("[MessageService] Successfully edited message: ChatID=%d, MessageID=%d", chatID, messageID)
	return &result.Result, nil
}

// AnswerCallbackQuery answers a callback query
func (ms *MessageService) AnswerCallbackQuery(callbackQueryID string, opts *gotgbot.AnswerCallbackQueryOpts) error {
	log.Printf("[MessageService] Answering callback query: ID=%s", callbackQueryID)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", ms.BotToken)

	requestBody := map[string]interface{}{
		"callback_query_id": callbackQueryID,
	}

	if opts != nil {
		if opts.Text != "" {
			requestBody["text"] = opts.Text
		}
		if opts.ShowAlert {
			requestBody["show_alert"] = opts.ShowAlert
		}
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		log.Printf("[MessageService] Error creating callback answer request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[MessageService] Error executing callback answer request: %v", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok bool `json:"ok"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[MessageService] Error parsing callback answer response: %v", err)
		return err
	}

	if !result.Ok {
		log.Printf("[MessageService] Failed to answer callback query: %s", string(body))
		return fmt.Errorf("failed to answer callback query: %s", string(body))
	}

	log.Printf("[MessageService] Successfully answered callback query: ID=%s", callbackQueryID)
	return nil
}
