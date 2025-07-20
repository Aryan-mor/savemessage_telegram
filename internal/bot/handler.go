package bot

import (
	"context"
	"save-message/internal/config"
	"save-message/internal/logutils"

	gotgbot "github.com/PaulSonOfLars/gotgbot/v2"
)

// Start initializes the bot and begins polling for updates
func Start(ctx context.Context, cfg *config.Env) error {
	logutils.Info("Start: entry")
	bot, err := gotgbot.NewBot(cfg.TelegramBotToken, nil)
	if err != nil {
		logutils.Error("Start: failed to create bot", err)
		return err
	}
	logutils.Success("Start: bot authorized", "username", bot.User.Username)

	// TODO: Re-implement update polling and topic logic using gotgbot's dispatcher/ext package.
	// For now, just log that migration is in progress.
	logutils.Info("Start: migration in progress - gotgbot is installed. Please re-implement update polling and topic logic here.")
	logutils.Success("Start: exit")
	return nil
}
