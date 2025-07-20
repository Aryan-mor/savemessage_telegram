package handlers

import (
	"strconv"

	"save-message/internal/config"
	"save-message/internal/interfaces"
	"save-message/internal/logutils"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// KeyboardBuilder handles building inline keyboards
type KeyboardBuilder struct{}

// NewKeyboardBuilder creates a new keyboard builder instance
func NewKeyboardBuilder() *KeyboardBuilder {
	return &KeyboardBuilder{}
}

// BuildSuggestionKeyboard builds keyboard for AI suggestions
func (kb *KeyboardBuilder) BuildSuggestionKeyboard(msg *gotgbot.Message, suggestions []string, topics []interfaces.ForumTopic) (*gotgbot.InlineKeyboardMarkup, error) {
	logutils.Info("BuildSuggestionKeyboard: entry", "messageID", msg.MessageId)
	var rows [][]gotgbot.InlineKeyboardButton
	for _, suggestion := range suggestions {
		rows = append(rows, []gotgbot.InlineKeyboardButton{{
			Text:         "➕ " + suggestion,
			CallbackData: suggestion + "_" + strconv.FormatInt(int64(msg.MessageId), 10),
		}})
	}
	// Add create new topic button
	rows = append(rows, []gotgbot.InlineKeyboardButton{{
		Text:         "📝 Create New Topic",
		CallbackData: "create_new_folder_" + strconv.FormatInt(int64(msg.MessageId), 10),
	}})
	keyboard := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
	logutils.Success("BuildSuggestionKeyboard: exit", "messageID", msg.MessageId)
	return keyboard, nil
}

// BuildAllTopicsKeyboard builds keyboard for showing all topics
func (kb *KeyboardBuilder) BuildAllTopicsKeyboard(originalMsg *gotgbot.Message, topics []interfaces.ForumTopic) (*gotgbot.InlineKeyboardMarkup, error) {
	logutils.Info("BuildAllTopicsKeyboard: entry", "messageID", originalMsg.MessageId)
	var rows [][]gotgbot.InlineKeyboardButton

	// Add all existing topics as buttons
	for _, topic := range topics {
		callbackData := topic.Name + "_" + strconv.FormatInt(originalMsg.MessageId, 10)
		rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: config.IconFolder + " " + topic.Name, CallbackData: callbackData}})
	}

	// Add back button
	backCallbackData := config.CallbackPrefixBackToSuggestions + strconv.FormatInt(originalMsg.MessageId, 10)
	backBtn := gotgbot.InlineKeyboardButton{Text: config.ButtonTextBackToSuggestions, CallbackData: backCallbackData}
	rows = append(rows, []gotgbot.InlineKeyboardButton{backBtn})

	result, err := func() (*gotgbot.InlineKeyboardMarkup, error) {
		return &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}, nil
	}()
	if err == nil {
		logutils.Success("BuildAllTopicsKeyboard: exit", "messageID", originalMsg.MessageId)
	} else {
		logutils.Error("BuildAllTopicsKeyboard: exit", err, "messageID", originalMsg.MessageId)
	}
	return result, err
}

// BuildBotMenuKeyboard builds keyboard for bot menu
func (kb *KeyboardBuilder) BuildBotMenuKeyboard() *gotgbot.InlineKeyboardMarkup {
	logutils.Info("BuildBotMenuKeyboard: entry")
	result := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextCreateNewTopic, CallbackData: config.CallbackDataCreateTopicMenu}},
			{{Text: config.ButtonTextShowAllTopics, CallbackData: config.CallbackDataShowAllTopicsMenu}},
		},
	}
	logutils.Success("BuildBotMenuKeyboard: exit")
	return result
}

// BuildAddTopicKeyboard builds keyboard for add topic command
func (kb *KeyboardBuilder) BuildAddTopicKeyboard() *gotgbot.InlineKeyboardMarkup {
	logutils.Info("BuildAddTopicKeyboard: entry")
	result := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextCreateNewTopic, CallbackData: config.CallbackDataCreateTopicMenu}},
		},
	}
	logutils.Success("BuildAddTopicKeyboard: exit")
	return result
}

// BuildWarningKeyboard builds keyboard for warning messages
func (kb *KeyboardBuilder) BuildWarningKeyboard(callbackData string) *gotgbot.InlineKeyboardMarkup {
	logutils.Info("BuildWarningKeyboard: entry", "callbackData", callbackData)
	result := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextOk, CallbackData: callbackData}},
		},
	}
	logutils.Success("BuildWarningKeyboard: exit", "callbackData", callbackData)
	return result
}
