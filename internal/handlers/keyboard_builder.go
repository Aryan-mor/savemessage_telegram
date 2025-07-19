package handlers

import (
	"strconv"
	"strings"

	"save-message/internal/config"
	"save-message/internal/services"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// KeyboardBuilder handles building inline keyboards
type KeyboardBuilder struct{}

// NewKeyboardBuilder creates a new keyboard builder instance
func NewKeyboardBuilder() *KeyboardBuilder {
	return &KeyboardBuilder{}
}

// BuildSuggestionKeyboard builds keyboard for AI suggestions
func (kb *KeyboardBuilder) BuildSuggestionKeyboard(msg *gotgbot.Message, suggestions []string, topics []services.ForumTopic) (*gotgbot.InlineKeyboardMarkup, error) {
	var rows [][]gotgbot.InlineKeyboardButton

	// Separate existing and new topics
	var existingTopics []string
	var newTopics []string

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
			continue
		}

		if isExisting {
			existingTopics = append(existingTopics, existingTopicName) // Use exact name
		} else {
			newTopics = append(newTopics, folder)
		}
	}

	// Add existing topics with folder icon
	for _, folder := range existingTopics {
		callbackData := folder + "_" + strconv.FormatInt(msg.MessageId, 10)
		rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: config.IconFolder + " " + folder, CallbackData: callbackData}})
	}

	// Add new topics with plus icon
	for _, folder := range newTopics {
		cleanFolder := strings.TrimSpace(folder)
		// Skip suggestions that are too long or contain newlines
		if len(cleanFolder) == 0 || len(cleanFolder) > 50 || strings.Contains(cleanFolder, "\n") {
			continue
		}
		callbackData := cleanFolder + "_" + strconv.FormatInt(msg.MessageId, 10)
		rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: config.IconNewFolder + " " + cleanFolder, CallbackData: callbackData}})
	}

	// Add create new folder option
	createCallbackData := config.CallbackPrefixCreateNewFolder + strconv.FormatInt(msg.MessageId, 10)
	createBtn := gotgbot.InlineKeyboardButton{Text: config.ButtonTextCreateNewTopic, CallbackData: createCallbackData}
	rows = append(rows, []gotgbot.InlineKeyboardButton{createBtn})

	// Add show all topics button if there are existing topics
	showAllCallbackData := ""
	if len(topics) > 0 {
		showAllCallbackData = config.CallbackPrefixShowAllTopics + strconv.FormatInt(msg.MessageId, 10)
		showAllBtn := gotgbot.InlineKeyboardButton{Text: config.ButtonTextShowAllTopics, CallbackData: showAllCallbackData}
		rows = append(rows, []gotgbot.InlineKeyboardButton{showAllBtn})
	}

	// Add retry button
	retryCallbackData := config.CallbackPrefixRetry + strconv.FormatInt(msg.MessageId, 10)
	retryBtn := gotgbot.InlineKeyboardButton{Text: config.ButtonTextTryAgain, CallbackData: retryCallbackData}
	rows = append(rows, []gotgbot.InlineKeyboardButton{retryBtn})

	return &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}, nil
}

// BuildAllTopicsKeyboard builds keyboard for showing all topics
func (kb *KeyboardBuilder) BuildAllTopicsKeyboard(originalMsg *gotgbot.Message, topics []services.ForumTopic) (*gotgbot.InlineKeyboardMarkup, error) {
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

	return &gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}, nil
}

// BuildBotMenuKeyboard builds keyboard for bot menu
func (kb *KeyboardBuilder) BuildBotMenuKeyboard() *gotgbot.InlineKeyboardMarkup {
	return &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextCreateNewTopic, CallbackData: config.CallbackDataCreateTopicMenu}},
			{{Text: config.ButtonTextShowAllTopics, CallbackData: config.CallbackDataShowAllTopicsMenu}},
		},
	}
}

// BuildAddTopicKeyboard builds keyboard for add topic command
func (kb *KeyboardBuilder) BuildAddTopicKeyboard() *gotgbot.InlineKeyboardMarkup {
	return &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextCreateNewTopic, CallbackData: config.CallbackDataCreateTopicMenu}},
		},
	}
}

// BuildWarningKeyboard builds keyboard for warning messages
func (kb *KeyboardBuilder) BuildWarningKeyboard(callbackData string) *gotgbot.InlineKeyboardMarkup {
	return &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextOk, CallbackData: callbackData}},
		},
	}
}
