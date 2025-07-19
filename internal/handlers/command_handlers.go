package handlers

import (
	"log"
	"strings"

	"save-message/internal/config"
	"save-message/internal/services"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

// CommandHandlers handles all command-related interactions
type CommandHandlers struct {
	MessageService *services.MessageService
	TopicService   *services.TopicService
}

// NewCommandHandlers creates a new command handlers instance
func NewCommandHandlers(messageService *services.MessageService, topicService *services.TopicService) *CommandHandlers {
	return &CommandHandlers{
		MessageService: messageService,
		TopicService:   topicService,
	}
}

// HandleStartCommand handles the /start command
func (ch *CommandHandlers) HandleStartCommand(update *gotgbot.Update) error {
	log.Printf("[CommandHandlers] Handling /start command: ChatID=%d", update.Message.Chat.Id)

	_, err := ch.MessageService.SendMessage(update.Message.Chat.Id, config.WelcomeMessage, nil)
	if err != nil {
		log.Printf("[CommandHandlers] Error sending start message: %v", err)
		return err
	}

	log.Printf("[CommandHandlers] Successfully sent start message: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// HandleHelpCommand handles the /help command
func (ch *CommandHandlers) HandleHelpCommand(update *gotgbot.Update) error {
	log.Printf("[CommandHandlers] Handling /help command: ChatID=%d", update.Message.Chat.Id)

	_, err := ch.MessageService.SendMessage(update.Message.Chat.Id, config.HelpMessage, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})
	if err != nil {
		log.Printf("[CommandHandlers] Error sending help message: %v", err)
		return err
	}

	log.Printf("[CommandHandlers] Successfully sent help message: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// HandleTopicsCommand handles the /topics command
func (ch *CommandHandlers) HandleTopicsCommand(update *gotgbot.Update) error {
	log.Printf("[CommandHandlers] Handling /topics command: ChatID=%d", update.Message.Chat.Id)

	topics, err := ch.TopicService.GetForumTopics(update.Message.Chat.Id)
	if err != nil {
		log.Printf("[CommandHandlers] Error getting topics: %v", err)
		_, sendErr := ch.MessageService.SendMessage(update.Message.Chat.Id, config.ErrorMessageFailed, &gotgbot.SendMessageOpts{})
		if sendErr != nil {
			log.Printf("[CommandHandlers] Error sending error message: %v", sendErr)
		}
		return err
	}

	if len(topics) == 0 {
		_, err = ch.MessageService.SendMessage(update.Message.Chat.Id, config.ErrorMessageNoTopics, &gotgbot.SendMessageOpts{})
		if err != nil {
			log.Printf("[CommandHandlers] Error sending no topics message: %v", err)
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
			log.Printf("[CommandHandlers] Error sending topics list: %v", err)
			return err
		}
	}

	log.Printf("[CommandHandlers] Successfully handled /topics command: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// HandleAddTopicCommand handles the /addtopic command
func (ch *CommandHandlers) HandleAddTopicCommand(update *gotgbot.Update) error {
	log.Printf("[CommandHandlers] Handling /addtopic command: ChatID=%d", update.Message.Chat.Id)

	keyboard := &gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: config.ButtonTextCreateNewTopic, CallbackData: config.CallbackDataCreateTopicMenu}},
		},
	}

	_, err := ch.MessageService.SendMessage(update.Message.Chat.Id, config.ChooseOptionMessage, &gotgbot.SendMessageOpts{
		ReplyMarkup: *keyboard,
	})
	if err != nil {
		log.Printf("[CommandHandlers] Error sending add topic menu: %v", err)
		return err
	}

	log.Printf("[CommandHandlers] Successfully handled /addtopic command: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// HandleBotMention handles when the bot is mentioned
func (ch *CommandHandlers) HandleBotMention(update *gotgbot.Update) error {
	log.Printf("[CommandHandlers] Handling bot mention: ChatID=%d", update.Message.Chat.Id)

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
		log.Printf("[CommandHandlers] Error sending bot menu: %v", err)
		return err
	}

	log.Printf("[CommandHandlers] Successfully handled bot mention: ChatID=%d", update.Message.Chat.Id)
	return nil
}

// IsBotMention checks if the message mentions the bot
func (ch *CommandHandlers) IsBotMention(messageText string) bool {
	lowerText := strings.ToLower(messageText)
	return strings.Contains(lowerText, config.BotUsername1) || strings.Contains(lowerText, config.BotUsername2)
}
