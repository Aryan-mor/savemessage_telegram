package config

import (
	"fmt"
	"os"

	"save-message/internal/logutils"

	"github.com/joho/godotenv"
)

// Env holds environment configuration
type Env struct {
	TelegramBotToken string
	OpenAIAPIKey     string
}

// LoadEnv loads environment variables from .env and returns Env struct
func LoadEnv() (*Env, error) {
	logutils.Info("LoadEnv: entry")
	_ = godotenv.Load() // Ignore error if .env is missing, allow env vars
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		logutils.Error("LoadEnv: TELEGRAM_BOT_TOKEN is not set", nil)
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is not set")
	}
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		logutils.Error("LoadEnv: OPENAI_API_KEY is not set", nil)
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}
	logutils.Success("LoadEnv: exit")
	return &Env{
		TelegramBotToken: token,
		OpenAIAPIKey:     openaiKey,
	}, nil
}
