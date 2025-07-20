package handlers

import (
	"strconv"
	"testing"

	"save-message/internal/config"
	"save-message/internal/interfaces"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewKeyboardBuilder(t *testing.T) {
	kb := NewKeyboardBuilder()
	assert.NotNil(t, kb)
	assert.IsType(t, &KeyboardBuilder{}, kb)
}

func TestKeyboardBuilder_BuildSuggestionKeyboard(t *testing.T) {
	kb := NewKeyboardBuilder()

	tests := []struct {
		name        string
		msg         *gotgbot.Message
		suggestions []string
		topics      []interfaces.ForumTopic
		expectRows  int
		expectError bool
	}{
		{
			name:        "successful_suggestion_keyboard",
			msg:         &gotgbot.Message{MessageId: 123},
			suggestions: []string{"Programming", "Development"},
			topics: []interfaces.ForumTopic{
				{Name: "Work", ID: 1},
				{Name: "General", ID: 2},
			},
			expectRows:  5, // 2 suggestions + create + show all + retry
			expectError: false,
		},
		{
			name:        "empty_suggestions",
			msg:         &gotgbot.Message{MessageId: 456},
			suggestions: []string{},
			topics: []interfaces.ForumTopic{
				{Name: "Work", ID: 1},
			},
			expectRows:  3, // create + show all + retry
			expectError: false,
		},
		{
			name:        "nil_suggestions",
			msg:         &gotgbot.Message{MessageId: 789},
			suggestions: nil,
			topics: []interfaces.ForumTopic{
				{Name: "Work", ID: 1},
			},
			expectRows:  3, // create + show all + retry
			expectError: false,
		},
		{
			name:        "no_existing_topics",
			msg:         &gotgbot.Message{MessageId: 999},
			suggestions: []string{"Programming"},
			topics:      []interfaces.ForumTopic{},
			expectRows:  3, // 1 suggestion + create + retry (no show all)
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyboard, err := kb.BuildSuggestionKeyboard(tt.msg, tt.suggestions, tt.topics)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, keyboard)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, keyboard)
			assert.Len(t, keyboard.InlineKeyboard, tt.expectRows)

			// Check for required buttons
			hasCreateButton := false
			hasRetryButton := false
			hasShowAllButton := false

			for _, row := range keyboard.InlineKeyboard {
				for _, button := range row {
					if button.CallbackData == config.CallbackPrefixCreateNewFolder+strconv.FormatInt(tt.msg.MessageId, 10) {
						hasCreateButton = true
					}
					if button.CallbackData == config.CallbackPrefixRetry+strconv.FormatInt(tt.msg.MessageId, 10) {
						hasRetryButton = true
					}
					if button.CallbackData == config.CallbackPrefixShowAllTopics+strconv.FormatInt(tt.msg.MessageId, 10) {
						hasShowAllButton = true
					}
				}
			}

			assert.True(t, hasCreateButton, "Should have create button")
			assert.True(t, hasRetryButton, "Should have retry button")

			// Show all button should only be present if there are existing topics
			if len(tt.topics) > 0 {
				assert.True(t, hasShowAllButton, "Should have show all button when topics exist")
			} else {
				assert.False(t, hasShowAllButton, "Should not have show all button when no topics exist")
			}
		})
	}
}

func TestKeyboardBuilder_BuildAllTopicsKeyboard(t *testing.T) {
	kb := NewKeyboardBuilder()

	tests := []struct {
		name        string
		originalMsg *gotgbot.Message
		topics      []interfaces.ForumTopic
		expectRows  int
		expectError bool
	}{
		{
			name:        "successful_all_topics",
			originalMsg: &gotgbot.Message{MessageId: 123},
			topics: []interfaces.ForumTopic{
				{Name: "Work", ID: 1},
				{Name: "General", ID: 2},
			},
			expectRows:  3, // 2 topics + back
			expectError: false,
		},
		{
			name:        "empty_topics",
			originalMsg: &gotgbot.Message{MessageId: 456},
			topics:      []interfaces.ForumTopic{},
			expectRows:  1, // back only
			expectError: false,
		},
		{
			name:        "nil_topics",
			originalMsg: &gotgbot.Message{MessageId: 789},
			topics:      nil,
			expectRows:  1, // back only
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyboard, err := kb.BuildAllTopicsKeyboard(tt.originalMsg, tt.topics)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, keyboard)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, keyboard)
			assert.Len(t, keyboard.InlineKeyboard, tt.expectRows)

			// Check for required buttons
			hasBackButton := false

			for _, row := range keyboard.InlineKeyboard {
				for _, button := range row {
					if button.CallbackData == config.CallbackPrefixBackToSuggestions+strconv.FormatInt(tt.originalMsg.MessageId, 10) {
						hasBackButton = true
					}
				}
			}

			assert.True(t, hasBackButton, "Should have back button")

			// Check topic buttons
			for _, topic := range tt.topics {
				found := false
				for _, row := range keyboard.InlineKeyboard {
					for _, button := range row {
						if button.CallbackData == topic.Name+"_"+strconv.FormatInt(tt.originalMsg.MessageId, 10) {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				assert.True(t, found, "Should have button for topic: %s", topic.Name)
			}
		})
	}
}

func TestKeyboardBuilder_BuildBotMenuKeyboard(t *testing.T) {
	kb := NewKeyboardBuilder()

	keyboard := kb.BuildBotMenuKeyboard()

	assert.NotNil(t, keyboard)
	assert.Len(t, keyboard.InlineKeyboard, 2)

	// Check for required buttons
	hasCreateButton := false
	hasShowAllButton := false

	for _, row := range keyboard.InlineKeyboard {
		for _, button := range row {
			if button.CallbackData == config.CallbackDataCreateTopicMenu {
				hasCreateButton = true
			}
			if button.CallbackData == config.CallbackDataShowAllTopicsMenu {
				hasShowAllButton = true
			}
		}
	}

	assert.True(t, hasCreateButton, "Should have create button")
	assert.True(t, hasShowAllButton, "Should have show all button")
}

func TestKeyboardBuilder_BuildAddTopicKeyboard(t *testing.T) {
	kb := NewKeyboardBuilder()

	keyboard := kb.BuildAddTopicKeyboard()

	assert.NotNil(t, keyboard)
	assert.Len(t, keyboard.InlineKeyboard, 1)

	// Check for required button
	hasCreateButton := false

	for _, row := range keyboard.InlineKeyboard {
		for _, button := range row {
			if button.CallbackData == config.CallbackDataCreateTopicMenu {
				hasCreateButton = true
			}
		}
	}

	assert.True(t, hasCreateButton, "Should have create button")
}

func TestKeyboardBuilder_BuildWarningKeyboard(t *testing.T) {
	kb := NewKeyboardBuilder()

	testCallbackData := "test_callback_data"
	keyboard := kb.BuildWarningKeyboard(testCallbackData)

	assert.NotNil(t, keyboard)
	assert.Len(t, keyboard.InlineKeyboard, 1)

	// Check for required button
	hasOkButton := false

	for _, row := range keyboard.InlineKeyboard {
		for _, button := range row {
			if button.CallbackData == testCallbackData {
				hasOkButton = true
			}
		}
	}

	assert.True(t, hasOkButton, "Should have ok button")
}
