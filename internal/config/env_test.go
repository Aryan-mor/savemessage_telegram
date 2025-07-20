package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadEnv(t *testing.T) {
	tests := []struct {
		name           string
		setupEnv       func()
		expectedError  bool
		expectedToken  string
		expectedAPIKey string
	}{
		{
			name: "successful load with both environment variables",
			setupEnv: func() {
				os.Setenv("TELEGRAM_BOT_TOKEN", "test_token_123")
				os.Setenv("OPENAI_API_KEY", "test_api_key_456")
			},
			expectedError:  false,
			expectedToken:  "test_token_123",
			expectedAPIKey: "test_api_key_456",
		},
		{
			name: "missing TELEGRAM_BOT_TOKEN",
			setupEnv: func() {
				os.Unsetenv("TELEGRAM_BOT_TOKEN")
				os.Setenv("OPENAI_API_KEY", "test_api_key_456")
			},
			expectedError: true,
		},
		{
			name: "missing OPENAI_API_KEY",
			setupEnv: func() {
				os.Setenv("TELEGRAM_BOT_TOKEN", "test_token_123")
				os.Unsetenv("OPENAI_API_KEY")
			},
			expectedError: true,
		},
		{
			name: "both environment variables missing",
			setupEnv: func() {
				os.Unsetenv("TELEGRAM_BOT_TOKEN")
				os.Unsetenv("OPENAI_API_KEY")
			},
			expectedError: true,
		},
		{
			name: "empty TELEGRAM_BOT_TOKEN",
			setupEnv: func() {
				os.Setenv("TELEGRAM_BOT_TOKEN", "")
				os.Setenv("OPENAI_API_KEY", "test_api_key_456")
			},
			expectedError: true,
		},
		{
			name: "empty OPENAI_API_KEY",
			setupEnv: func() {
				os.Setenv("TELEGRAM_BOT_TOKEN", "test_token_123")
				os.Setenv("OPENAI_API_KEY", "")
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			tt.setupEnv()

			// Call function
			result, err := LoadEnv()

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedToken, result.TelegramBotToken)
				assert.Equal(t, tt.expectedAPIKey, result.OpenAIAPIKey)
			}
		})
	}
}

func TestEnv_Fields(t *testing.T) {
	// Test that Env struct has the expected fields
	env := &Env{
		TelegramBotToken: "test_token",
		OpenAIAPIKey:     "test_key",
	}

	assert.Equal(t, "test_token", env.TelegramBotToken)
	assert.Equal(t, "test_key", env.OpenAIAPIKey)
}

func TestConstants(t *testing.T) {
	// Test that constants are properly defined
	assert.NotEmpty(t, WelcomeMessage)
	assert.NotEmpty(t, HelpMessage)
	assert.NotEmpty(t, ErrorMessageNotFound)
	assert.NotEmpty(t, SuccessMessageSaved)
	assert.NotEmpty(t, WarningNonGeneralTopic)
	assert.NotEmpty(t, ButtonTextCreateNewTopic)
	assert.NotEmpty(t, BotMenuMessage)
	assert.NotEmpty(t, TopicNamePrompt)
	assert.NotEmpty(t, TopicsListHeader)
	assert.NotEmpty(t, AIProcessingMessage)
	assert.NotEmpty(t, CallbackPrefixCreateNewFolder)
	assert.NotEmpty(t, BotUsername1)
	assert.NotEmpty(t, BotUsername2)
	assert.NotEmpty(t, DefaultDatabasePath)
	assert.Greater(t, DefaultPollingTimeout, 0)
	assert.Greater(t, DefaultRetryDelay, time.Duration(0))
	assert.Greater(t, DefaultWarningAutoDeleteDelay, time.Duration(0))
	assert.Greater(t, DefaultMessageAutoDeleteDelay, time.Duration(0))
	assert.NotEmpty(t, IconFolder)
	assert.NotEmpty(t, IconNewFolder)
	assert.NotEmpty(t, IconCreate)
	assert.NotEmpty(t, IconRetry)
	assert.NotEmpty(t, IconBack)
	assert.NotEmpty(t, IconBot)
	assert.NotEmpty(t, IconWarning)
	assert.NotEmpty(t, IconError)
	assert.NotEmpty(t, IconSuccess)
	assert.NotEmpty(t, IconThinking)
}
