package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadEnv(t *testing.T) {
	t.Run("successful_load_with_both_environment_variables", func(t *testing.T) {
		t.Setenv("TELEGRAM_BOT_TOKEN", "dummy")
		t.Setenv("OPENAI_API_KEY", "dummy")
		_, err := LoadEnv()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("missing_TELEGRAM_BOT_TOKEN", func(t *testing.T) {
		t.Setenv("TELEGRAM_BOT_TOKEN", "")
		t.Setenv("OPENAI_API_KEY", "dummy")
		_, err := LoadEnv()
		if err == nil {
			t.Fatalf("expected error for missing TELEGRAM_BOT_TOKEN, got nil")
		}
	})

	t.Run("missing_OPENAI_API_KEY", func(t *testing.T) {
		t.Setenv("TELEGRAM_BOT_TOKEN", "dummy")
		t.Setenv("OPENAI_API_KEY", "")
		_, err := LoadEnv()
		if err == nil {
			t.Fatalf("expected error for missing OPENAI_API_KEY, got nil")
		}
	})

	t.Run("both_environment_variables_missing", func(t *testing.T) {
		t.Setenv("TELEGRAM_BOT_TOKEN", "")
		t.Setenv("OPENAI_API_KEY", "")
		_, err := LoadEnv()
		if err == nil {
			t.Fatalf("expected error for both missing, got nil")
		}
	})

	t.Run("empty_TELEGRAM_BOT_TOKEN", func(t *testing.T) {
		t.Setenv("TELEGRAM_BOT_TOKEN", "")
		t.Setenv("OPENAI_API_KEY", "dummy")
		_, err := LoadEnv()
		if err == nil {
			t.Fatalf("expected error for empty TELEGRAM_BOT_TOKEN, got nil")
		}
	})

	t.Run("empty_OPENAI_API_KEY", func(t *testing.T) {
		t.Setenv("TELEGRAM_BOT_TOKEN", "dummy")
		t.Setenv("OPENAI_API_KEY", "")
		_, err := LoadEnv()
		if err == nil {
			t.Fatalf("expected error for empty OPENAI_API_KEY, got nil")
		}
	})
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
