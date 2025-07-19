package bot

import (
	"context"
	"log"

	"save-message/internal/config"

	gotgbot "github.com/PaulSonOfLars/gotgbot/v2"
)

// Start initializes the bot and begins polling for updates
func Start(ctx context.Context, cfg *config.Env) error {
	bot, err := gotgbot.NewBot(cfg.TelegramBotToken, nil)
	if err != nil {
		return err
	}
	log.Printf("Authorized on account %s", bot.User.Username)

	// TODO: Re-implement update polling and topic logic using gotgbot's dispatcher/ext package.
	// For now, just log that migration is in progress.
	log.Println("[MIGRATION] gotgbot is installed. Please re-implement update polling and topic logic here.")
	return nil
}
