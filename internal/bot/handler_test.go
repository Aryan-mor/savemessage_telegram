package bot

import (
	"context"
	"os"
	"testing"

	"save-message/internal/config"
)

func TestStart_MainFlow(t *testing.T) {
	if os.Getenv("TELEGRAM_BOT_TOKEN") == "" {
		t.Skip("Skipping TestStart_MainFlow: TELEGRAM_BOT_TOKEN not set")
	}
	ctx := context.Background()
	cfg := &config.Env{TelegramBotToken: "dummy", OpenAIAPIKey: "dummy"}
	if err := Start(ctx, cfg); err != nil {
		t.Errorf("Start returned error: %v", err)
	}
}
