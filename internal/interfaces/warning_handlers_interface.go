package interfaces

import "github.com/PaulSonOfLars/gotgbot/v2"

// WarningHandlersInterface defines the interface for warning-related handlers.
type WarningHandlersInterface interface {
	HandleNonGeneralTopicMessage(update *gotgbot.Update) error
	IsWarningCallback(callbackData string) bool
	HandleWarningOkCallback(update *gotgbot.Update) error
}
