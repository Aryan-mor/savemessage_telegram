package router

import (
	"context"

	"save-message/internal/logutils"

	gotgbot "github.com/PaulSonOfLars/gotgbot/v2"
)

// HandleMessage processes incoming messages and replies appropriately
func HandleMessage(ctx context.Context, bot *gotgbot.Bot, msg *gotgbot.Message) error {
	logutils.Info("router.HandleMessage: entry", "chatID", func() int64 {
		if msg != nil {
			return msg.Chat.Id
		} else {
			return 0
		}
	}(), "messageID", func() int64 {
		if msg != nil {
			return msg.MessageId
		} else {
			return 0
		}
	}())
	// TODO: Re-implement message handling using gotgbot types and methods.
	logutils.Success("router.HandleMessage: exit", "chatID", func() int64 {
		if msg != nil {
			return msg.Chat.Id
		} else {
			return 0
		}
	}(), "messageID", func() int64 {
		if msg != nil {
			return msg.MessageId
		} else {
			return 0
		}
	}())
	return nil
}
