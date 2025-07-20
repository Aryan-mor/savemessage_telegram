package handlers

import (
	"strings"

	"save-message/internal/config"
	"save-message/internal/interfaces"
	"save-message/internal/logutils"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// CommandHandlers handles all command-related interactions
type CommandHandlers struct {
	MessageService interfaces.MessageServiceInterface
	TopicService   interfaces.TopicServiceInterface

	// Mockable funcs for testing
	HandleStartCommandFunc    func(update *gotgbot.Update) error
	HandleHelpCommandFunc     func(update *gotgbot.Update) error
	HandleTopicsCommandFunc   func(update *gotgbot.Update) error
	HandleAddTopicCommandFunc func(update *gotgbot.Update) error
	HandleBotMentionFunc      func(update *gotgbot.Update) error
}

// NewCommandHandlers creates a new command handlers instance
func NewCommandHandlers(messageService interfaces.MessageServiceInterface, topicService interfaces.TopicServiceInterface) *CommandHandlers {
	return &CommandHandlers{
		MessageService: messageService,
		TopicService:   topicService,
	}
}

// HandleStartCommand handles the /start command
func (ch *CommandHandlers) HandleStartCommand(update *gotgbot.Update) error {
	logutils.Info("HandleStartCommand", "chatID", update.Message.Chat.Id)

	keyboard := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: "Help", CallbackData: "show_help"}},
		},
	}

	_, err := ch.MessageService.SendMessage(update.Message.Chat.Id, config.WelcomeMessage, &gotgbot.SendMessageOpts{
		ReplyMarkup: *keyboard,
	})
	if err != nil {
		logutils.Error("HandleStartCommand: SendMessageError", err, "chatID", update.Message.Chat.Id)
		return err
	}
	logutils.Success("HandleStartCommand", "chatID", update.Message.Chat.Id)
	return nil
}

// HandleHelpCommand handles the /help command
func (ch *CommandHandlers) HandleHelpCommand(update *gotgbot.Update) error {
	logutils.Info("HandleHelpCommand", "chatID", update.Message.Chat.Id)

	_, err := ch.MessageService.SendMessage(update.Message.Chat.Id, config.HelpMessage, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})
	if err != nil {
		logutils.Error("HandleHelpCommand: SendMessageError", err, "chatID", update.Message.Chat.Id)
		return err
	}
	logutils.Success("HandleHelpCommand", "chatID", update.Message.Chat.Id)
	return nil
}

// HandleTopicsCommand handles the /topics command
func (ch *CommandHandlers) HandleTopicsCommand(update *gotgbot.Update) error {
	logutils.Info("HandleTopicsCommand", "chatID", update.Message.Chat.Id)

	topics, err := ch.TopicService.GetForumTopics(update.Message.Chat.Id)
	if err != nil {
		logutils.Error("HandleTopicsCommand: GetForumTopicsError", err, "chatID", update.Message.Chat.Id)
		_, sendErr := ch.MessageService.SendMessage(update.Message.Chat.Id, config.ErrorMessageFailed, &gotgbot.SendMessageOpts{})
		if sendErr != nil {
			logutils.Error("HandleTopicsCommand: SendErrorMessageError", sendErr, "chatID", update.Message.Chat.Id)
		}
		return err
	}

	if len(topics) == 0 {
		_, err = ch.MessageService.SendMessage(update.Message.Chat.Id, config.ErrorMessageNoTopics, &gotgbot.SendMessageOpts{})
		if err != nil {
			logutils.Error("HandleTopicsCommand: SendErrorMessageNoTopicsError", err, "chatID", update.Message.Chat.Id)
			return err
		}
	} else {
		topicList := config.TopicsListHeader
		for _, topic := range topics {
			topicList += "• " + topic.Name + "\n"
		}
		_, err = ch.MessageService.SendMessage(update.Message.Chat.Id, topicList, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
		if err != nil {
			logutils.Error("HandleTopicsCommand: SendTopicsListError", err, "chatID", update.Message.Chat.Id)
			return err
		}
	}

	logutils.Success("HandleTopicsCommand", "chatID", update.Message.Chat.Id)
	return nil
}

// HandleAddTopicCommand handles the /addtopic command
func (ch *CommandHandlers) HandleAddTopicCommand(update *gotgbot.Update) error {
	logutils.Info("HandleAddTopicCommand", "chatID", update.Message.Chat.Id)

	keyboard := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextCreateNewTopic, CallbackData: config.CallbackDataCreateTopicMenu}},
		},
	}

	_, err := ch.MessageService.SendMessage(update.Message.Chat.Id, config.ChooseOptionMessage, &gotgbot.SendMessageOpts{
		ReplyMarkup: *keyboard,
	})
	if err != nil {
		logutils.Error("HandleAddTopicCommand: SendMessageError", err, "chatID", update.Message.Chat.Id)
		return err
	}

	logutils.Success("HandleAddTopicCommand", "chatID", update.Message.Chat.Id)
	return nil
}

// HandleBotMention handles when the bot is mentioned
func (ch *CommandHandlers) HandleBotMention(update *gotgbot.Update) error {
	logutils.Info("HandleBotMention", "chatID", update.Message.Chat.Id)

	keyboard := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextCreateNewTopic, CallbackData: config.CallbackDataCreateTopicMenu}},
			{{Text: config.ButtonTextShowAllTopics, CallbackData: config.CallbackDataShowAllTopicsMenu}},
		},
	}

	_, err := ch.MessageService.SendMessage(update.Message.Chat.Id, config.BotMenuMessage, &gotgbot.SendMessageOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: *keyboard,
	})
	if err != nil {
		logutils.Error("HandleBotMention: SendMessageError", err, "chatID", update.Message.Chat.Id)
		return err
	}

	logutils.Success("HandleBotMention", "chatID", update.Message.Chat.Id)
	return nil
}

// IsBotMention checks if the message mentions the bot
func (ch *CommandHandlers) IsBotMention(messageText string) bool {
	lowerText := strings.ToLower(messageText)
	return strings.Contains(lowerText, config.BotUsername1) || strings.Contains(lowerText, config.BotUsername2)
}

func (ch *CommandHandlers) HandleNonGeneralTopicMessage(update *gotgbot.Update) error {
	// Not implemented for command handlers
	return nil
}

func (ch *CommandHandlers) HandleGeneralTopicMessage(update *gotgbot.Update) error {
	// Not implemented for command handlers
	return nil
}
