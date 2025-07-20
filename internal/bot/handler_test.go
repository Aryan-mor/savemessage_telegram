package bot

import (
	"context"
	"testing"

	"save-message/internal/config"
)

func TestStart_MainFlow(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Env{TelegramBotToken: "dummy", OpenAIAPIKey: "dummy"}
	if err := Start(ctx, cfg); err != nil {
		t.Errorf("Start returned error: %v", err)
	}
}
