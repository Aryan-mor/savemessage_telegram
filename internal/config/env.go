package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Env holds environment configuration
type Env struct {
	TelegramBotToken string
	OpenAIAPIKey     string
}

// LoadEnv loads environment variables from .env and returns Env struct
func LoadEnv() (*Env, error) {
	_ = godotenv.Load() // Ignore error if .env is missing, allow env vars
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is not set")
	}
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}
	return &Env{
		TelegramBotToken: token,
		OpenAIAPIKey:     openaiKey,
	}, nil
}
