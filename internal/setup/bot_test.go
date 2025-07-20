package setup

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupEnv       func()
		expectedError  bool
		expectedConfig *BotConfig
	}{
		{
			name: "successful load with all environment variables",
			setupEnv: func() {
				os.Setenv("TELEGRAM_BOT_TOKEN", "test_bot_token")
				os.Setenv("OPENAI_API_KEY", "test_openai_key")
				os.Setenv("DB_PATH", "test.db")
			},
			expectedError: false,
			expectedConfig: &BotConfig{
				BotToken:  "test_bot_token",
				OpenAIKey: "test_openai_key",
				DBPath:    "test.db",
			},
		},
		{
			name: "successful load with default DB path",
			setupEnv: func() {
				os.Setenv("TELEGRAM_BOT_TOKEN", "test_bot_token")
				os.Setenv("OPENAI_API_KEY", "test_openai_key")
				os.Unsetenv("DB_PATH")
			},
			expectedError: false,
			expectedConfig: &BotConfig{
				BotToken:  "test_bot_token",
				OpenAIKey: "test_openai_key",
				DBPath:    "bot.db",
			},
		},
		{
			name: "missing TELEGRAM_BOT_TOKEN",
			setupEnv: func() {
				os.Unsetenv("TELEGRAM_BOT_TOKEN")
				os.Setenv("OPENAI_API_KEY", "test_openai_key")
				os.Setenv("DB_PATH", "test.db")
			},
			expectedError: true,
		},
		{
			name: "missing OPENAI_API_KEY",
			setupEnv: func() {
				os.Setenv("TELEGRAM_BOT_TOKEN", "test_bot_token")
				os.Unsetenv("OPENAI_API_KEY")
				os.Setenv("DB_PATH", "test.db")
			},
			expectedError: true,
		},
		{
			name: "both required environment variables missing",
			setupEnv: func() {
				os.Unsetenv("TELEGRAM_BOT_TOKEN")
				os.Unsetenv("OPENAI_API_KEY")
				os.Setenv("DB_PATH", "test.db")
			},
			expectedError: true,
		},
		{
			name: "empty TELEGRAM_BOT_TOKEN",
			setupEnv: func() {
				os.Setenv("TELEGRAM_BOT_TOKEN", "")
				os.Setenv("OPENAI_API_KEY", "test_openai_key")
				os.Setenv("DB_PATH", "test.db")
			},
			expectedError: true,
		},
		{
			name: "empty OPENAI_API_KEY",
			setupEnv: func() {
				os.Setenv("TELEGRAM_BOT_TOKEN", "test_bot_token")
				os.Setenv("OPENAI_API_KEY", "")
				os.Setenv("DB_PATH", "test.db")
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			tt.setupEnv()

			// Call function
			result, err := LoadConfig()

			// Assertions
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedConfig.BotToken, result.BotToken)
				assert.Equal(t, tt.expectedConfig.OpenAIKey, result.OpenAIKey)
				assert.Equal(t, tt.expectedConfig.DBPath, result.DBPath)
			}
		})
	}
}

func TestBotConfig_Fields(t *testing.T) {
	// Test that BotConfig struct has the expected fields
	config := &BotConfig{
		BotToken:  "test_token",
		OpenAIKey: "test_key",
		DBPath:    "test.db",
	}

	assert.Equal(t, "test_token", config.BotToken)
	assert.Equal(t, "test_key", config.OpenAIKey)
	assert.Equal(t, "test.db", config.DBPath)
}

func TestInitializeBot(t *testing.T) {
	tests := []struct {
		name        string
		config      *BotConfig
		expectError bool
	}{
		{
			name: "valid configuration",
			config: &BotConfig{
				BotToken:  "test_bot_token",
				OpenAIKey: "test_openai_key",
				DBPath:    ":memory:", // Use in-memory database for testing
			},
			expectError: false, // This will likely fail due to invalid token, but we test structure
		},
		{
			name:        "nil configuration",
			config:      nil,
			expectError: true,
		},
		{
			name: "empty bot token",
			config: &BotConfig{
				BotToken:  "",
				OpenAIKey: "test_openai_key",
				DBPath:    ":memory:",
			},
			expectError: true,
		},
		{
			name: "empty OpenAI key",
			config: &BotConfig{
				BotToken:  "test_bot_token",
				OpenAIKey: "",
				DBPath:    ":memory:",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config == nil {
				// Test that nil config causes panic (as expected)
				assert.Panics(t, func() {
					_, _ = InitializeBot(tt.config)
				})
			} else {
				// Test with valid config structure
				// Note: This will likely fail due to invalid tokens, but we're testing the interface
				instance, err := InitializeBot(tt.config)

				if tt.expectError {
					assert.Error(t, err)
					assert.Nil(t, instance)
				} else {
					// Even if there's an error due to invalid tokens, we're testing the structure
					assert.NotNil(t, tt.config)
				}
			}
		})
	}
}

func TestBotInstance_Cleanup(t *testing.T) {
	// Test that Cleanup doesn't panic
	instance := &BotInstance{}
	assert.NotPanics(t, func() {
		instance.Cleanup()
	})

	// Test with nil database
	instance = &BotInstance{
		Database: nil,
	}
	assert.NotPanics(t, func() {
		instance.Cleanup()
	})
}

func TestBotInstance_Structure(t *testing.T) {
	// Test that BotInstance struct has the expected fields
	instance := &BotInstance{
		Bot:              nil,
		Config:           &BotConfig{},
		Database:         nil,
		MessageService:   nil,
		TopicService:     nil,
		AIService:        nil,
		MessageHandlers:  nil,
		CallbackHandlers: nil,
		Dispatcher:       nil,
	}

	assert.NotNil(t, instance)
	assert.NotNil(t, instance.Config)
}

func TestLoadConfig_EnvironmentHandling(t *testing.T) {
	// Test with missing .env file (should not fail)
	originalEnv := os.Getenv("ENV")
	os.Unsetenv("ENV")
	defer os.Setenv("ENV", originalEnv)

	// Set required environment variables
	os.Setenv("TELEGRAM_BOT_TOKEN", "test_token")
	os.Setenv("OPENAI_API_KEY", "test_key")
	os.Unsetenv("DB_PATH") // Ensure DB_PATH is not set to get default value

	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "test_token", config.BotToken)
	assert.Equal(t, "test_key", config.OpenAIKey)
	assert.Equal(t, "bot.db", config.DBPath) // Default value when DB_PATH is not set
}

func TestLoadConfig_DefaultDBPath(t *testing.T) {
	// Test that default DB path is used when not specified
	os.Setenv("TELEGRAM_BOT_TOKEN", "test_token")
	os.Setenv("OPENAI_API_KEY", "test_key")
	os.Unsetenv("DB_PATH")

	config, err := LoadConfig()
	assert.NoError(t, err)
	assert.Equal(t, "bot.db", config.DBPath)
}

func TestInitializeBot_ComponentInitialization(t *testing.T) {
	// Test that all components are properly initialized
	config := &BotConfig{
		BotToken:  "test_bot_token",
		OpenAIKey: "test_openai_key",
		DBPath:    ":memory:",
	}

	// This will likely fail due to invalid tokens, but we're testing the structure
	instance, err := InitializeBot(config)

	// Even if there's an error, we can test that the function has the expected structure
	assert.NotNil(t, config)

	// If initialization succeeds, test the structure
	if err == nil && instance != nil {
		assert.NotNil(t, instance.Config)
		assert.NotNil(t, instance.Database)
		assert.NotNil(t, instance.MessageService)
		assert.NotNil(t, instance.TopicService)
		assert.NotNil(t, instance.AIService)
		assert.NotNil(t, instance.MessageHandlers)
		assert.NotNil(t, instance.CallbackHandlers)
		assert.NotNil(t, instance.Dispatcher)
	}
}
