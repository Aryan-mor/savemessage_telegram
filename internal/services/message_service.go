package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"save-message/internal/database"
	"save-message/internal/interfaces"
	"save-message/internal/logutils"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// MessageService handles all message-related operations
type MessageService struct {
	BotToken string
	db       database.DatabaseInterface
}

// NewMessageService creates a new message service
func NewMessageService(botToken string, db database.DatabaseInterface) *MessageService {
	return &MessageService{
		BotToken: botToken,
		db:       db,
	}
}

var _ interfaces.MessageServiceInterface = (*MessageService)(nil)

// DeleteMessage deletes a message from a chat
func (ms *MessageService) DeleteMessage(chatID int64, messageID int) error {
	logutils.Info("DeleteMessage: entry", "chatID", chatID, "messageID", messageID)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteMessage", ms.BotToken)

	requestBody := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		logutils.Error("DeleteMessage: CreateRequest", err, "chatID", chatID)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logutils.Error("DeleteMessage: ExecuteRequest", err, "chatID", chatID)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	logutils.Info("DeleteMessage: API response", "chatID", chatID, "messageID", messageID, "body", string(body))

	var result struct {
		Ok bool `json:"ok"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logutils.Error("DeleteMessage: ParseResponse", err, "body", string(body))
		return err
	}

	if !result.Ok {
		err := fmt.Errorf("failed to delete message: %s", string(body))
		logutils.Warn("DeleteMessage: APIError", "error", err.Error())
		return err
	}

	logutils.Success("DeleteMessage: exit", "chatID", chatID, "messageID", messageID)
	return nil
}

// CopyMessageToTopic copies a message to a specific topic
func (ms *MessageService) CopyMessageToTopic(chatID int64, fromChatID int64, messageID int, messageThreadID int) error {
	logutils.Info("CopyMessageToTopic", "chatID", chatID, "fromChatID", fromChatID, "messageID", messageID, "messageThreadID", messageThreadID)

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
		logutils.Error("CopyMessageToTopic: CreateRequest", err, "chatID", chatID)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logutils.Error("CopyMessageToTopic: ExecuteRequest", err, "chatID", chatID)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok bool `json:"ok"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logutils.Error("CopyMessageToTopic: ParseResponse", err, "body", string(body))
		return err
	}

	if !result.Ok {
		err := fmt.Errorf("failed to copy message: %s", string(body))
		logutils.Warn("CopyMessageToTopic: APIError", "error", err.Error())
		return err
	}

	logutils.Success("CopyMessageToTopic", "chatID", chatID, "messageID", messageID, "messageThreadID", messageThreadID)
	return nil
}

// CopyMessageToTopicWithResult copies a message to a topic and returns the new message
func (ms *MessageService) CopyMessageToTopicWithResult(chatID int64, fromChatID int64, messageID int, messageThreadID int) (*gotgbot.Message, error) {
	logutils.Info("CopyMessageToTopicWithResult", "chatID", chatID, "fromChatID", fromChatID, "messageID", messageID, "messageThreadID", messageThreadID)

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
		logutils.Error("CopyMessageToTopicWithResult: CreateRequest", err, "chatID", chatID)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logutils.Error("CopyMessageToTopicWithResult: ExecuteRequest", err, "chatID", chatID)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok     bool            `json:"ok"`
		Result gotgbot.Message `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logutils.Error("CopyMessageToTopicWithResult: ParseResponse", err, "body", string(body))
		return nil, err
	}

	if !result.Ok {
		err := fmt.Errorf("failed to copy message: %s", string(body))
		logutils.Warn("CopyMessageToTopicWithResult: APIError", "error", err.Error())
		return nil, err
	}

	logutils.Success("CopyMessageToTopicWithResult", "chatID", chatID, "messageID", messageID, "messageThreadID", messageThreadID)
	return &result.Result, nil
}

// SendMessage sends a message to a chat
func (ms *MessageService) SendMessage(chatID int64, text string, opts *gotgbot.SendMessageOpts) (*gotgbot.Message, error) {
	logutils.Info("SendMessage", "chatID", chatID, "text", text)

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
		logutils.Error("SendMessage: CreateRequest", err, "chatID", chatID)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logutils.Error("SendMessage: ExecuteRequest", err, "chatID", chatID)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok     bool            `json:"ok"`
		Result gotgbot.Message `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logutils.Error("SendMessage: ParseResponse", err, "body", string(body))
		return nil, err
	}

	if !result.Ok {
		err := fmt.Errorf("failed to send message: %s", string(body))
		logutils.Warn("SendMessage: APIError", "error", err.Error())
		return nil, err
	}

	logutils.Success("SendMessage", "chatID", chatID, "messageID", result.Result.MessageId)
	return &result.Result, nil
}

// EditMessageText edits a message's text
func (ms *MessageService) EditMessageText(chatID int64, messageID int64, text string, opts *gotgbot.EditMessageTextOpts) (*gotgbot.Message, error) {
	logutils.Info("EditMessageText", "chatID", chatID, "messageID", messageID, "text", text)

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
		// Check if ReplyMarkup has any content
		if len(opts.ReplyMarkup.InlineKeyboard) > 0 {
			requestBody["reply_markup"] = opts.ReplyMarkup
		}
	}

	bodyBytes, _ := json.Marshal(requestBody)
	logutils.Info("EditMessageText: RequestBody", "body", string(bodyBytes))

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		logutils.Error("EditMessageText: CreateRequest", err, "chatID", chatID)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logutils.Error("EditMessageText: ExecuteRequest", err, "chatID", chatID)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	logutils.Info("EditMessageText: ResponseBody", "body", string(body))

	var result struct {
		Ok     bool            `json:"ok"`
		Result gotgbot.Message `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logutils.Error("EditMessageText: ParseResponse", err, "body", string(body))
		return nil, err
	}

	if !result.Ok {
		err := fmt.Errorf("failed to edit message: %s", string(body))
		logutils.Warn("EditMessageText: APIError", "error", err.Error())
		return nil, err
	}

	logutils.Success("EditMessageText", "chatID", chatID, "messageID", messageID)
	return &result.Result, nil
}

// AnswerCallbackQuery answers a callback query
func (ms *MessageService) AnswerCallbackQuery(callbackQueryID string, opts *gotgbot.AnswerCallbackQueryOpts) error {
	logutils.Info("AnswerCallbackQuery", "callbackQueryID", callbackQueryID)

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
		logutils.Error("AnswerCallbackQuery: CreateRequest", err, "callbackQueryID", callbackQueryID)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logutils.Error("AnswerCallbackQuery: ExecuteRequest", err, "callbackQueryID", callbackQueryID)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok bool `json:"ok"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logutils.Error("AnswerCallbackQuery: ParseResponse", err, "body", string(body))
		return err
	}

	if !result.Ok {
		err := fmt.Errorf("failed to answer callback query: %s", string(body))
		logutils.Warn("AnswerCallbackQuery: APIError", "error", err.Error())
		return err
	}

	logutils.Success("AnswerCallbackQuery", "callbackQueryID", callbackQueryID)
	return nil
}
