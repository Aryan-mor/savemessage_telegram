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
	builder := NewKeyboardBuilder()
	msg := &gotgbot.Message{MessageId: 123, Chat: gotgbot.Chat{Id: 456}}
	t.Run("successful_suggestion_keyboard", func(t *testing.T) {
		suggestions := []string{"Programming", "Development"}
		topics := []interfaces.ForumTopic{}
		keyboard, err := builder.BuildSuggestionKeyboard(msg, suggestions, topics)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var foundCreate, foundProgramming, foundDevelopment bool
		for _, row := range keyboard.InlineKeyboard {
			for _, btn := range row {
				if btn.Text == "📝 Create New Topic" {
					foundCreate = true
				}
				if btn.Text == "➕ Programming" {
					foundProgramming = true
				}
				if btn.Text == "➕ Development" {
					foundDevelopment = true
				}
				if btn.Text == "🔄 Try Again" {
					t.Errorf("Retry button should NOT be present in the keyboard, but was found")
				}
			}
		}
		if !foundCreate {
			t.Errorf("Create New Topic button not found")
		}
		if !foundProgramming {
			t.Errorf("Programming suggestion not found")
		}
		if !foundDevelopment {
			t.Errorf("Development suggestion not found")
		}
	})

	t.Run("empty_suggestions", func(t *testing.T) {
		suggestions := []string{}
		topics := []interfaces.ForumTopic{}
		keyboard, err := builder.BuildSuggestionKeyboard(msg, suggestions, topics)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var foundCreate bool
		for _, row := range keyboard.InlineKeyboard {
			for _, btn := range row {
				if btn.Text == "📝 Create New Topic" {
					foundCreate = true
				}
				if btn.Text == "🔄 Try Again" {
					t.Errorf("Retry button should NOT be present in the keyboard, but was found")
				}
			}
		}
		if !foundCreate {
			t.Errorf("Create New Topic button not found")
		}
	})

	t.Run("nil_suggestions", func(t *testing.T) {
		var suggestions []string
		topics := []interfaces.ForumTopic{}
		keyboard, err := builder.BuildSuggestionKeyboard(msg, suggestions, topics)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var foundCreate bool
		for _, row := range keyboard.InlineKeyboard {
			for _, btn := range row {
				if btn.Text == "📝 Create New Topic" {
					foundCreate = true
				}
				if btn.Text == "🔄 Try Again" {
					t.Errorf("Retry button should NOT be present in the keyboard, but was found")
				}
			}
		}
		if !foundCreate {
			t.Errorf("Create New Topic button not found")
		}
	})

	t.Run("no_existing_topics", func(t *testing.T) {
		suggestions := []string{"Programming"}
		topics := []interfaces.ForumTopic{}
		keyboard, err := builder.BuildSuggestionKeyboard(msg, suggestions, topics)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var foundProgramming, foundCreate bool
		for _, row := range keyboard.InlineKeyboard {
			for _, btn := range row {
				if btn.Text == "➕ Programming" {
					foundProgramming = true
				}
				if btn.Text == "📝 Create New Topic" {
					foundCreate = true
				}
				if btn.Text == "🔄 Try Again" {
					t.Errorf("Retry button should NOT be present in the keyboard, but was found")
				}
			}
		}
		if !foundProgramming {
			t.Errorf("Programming suggestion not found")
		}
		if !foundCreate {
			t.Errorf("Create New Topic button not found")
		}
	})
}

// Regression test: ensures that the '🔄 Try Again' button is NOT present in the keyboard returned by BuildSuggestionKeyboard.
// This prevents accidental reintroduction of the retry button in the topic suggestion UI.
func TestBuildSuggestionKeyboard_DoesNotIncludeRetryButton(t *testing.T) {
	builder := NewKeyboardBuilder()
	msg := &gotgbot.Message{MessageId: 123, Chat: gotgbot.Chat{Id: 456}}
	suggestions := []string{"Topic1", "Topic2"}
	topics := []interfaces.ForumTopic{}
	keyboard, err := builder.BuildSuggestionKeyboard(msg, suggestions, topics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, row := range keyboard.InlineKeyboard {
		for _, btn := range row {
			if btn.Text == "🔄 Try Again" {
				t.Errorf("Retry button should NOT be present in the keyboard, but was found")
			}
		}
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
