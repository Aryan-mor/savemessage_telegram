package setup

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"save-message/internal/database"
	"save-message/internal/handlers"
	"save-message/internal/logutils"
	"save-message/internal/router"
	"save-message/internal/services"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/joho/godotenv"
)

// BotConfig holds all configuration for the bot
type BotConfig struct {
	BotToken  string
	OpenAIKey string
	DBPath    string
}

// BotInstance holds all initialized components
type BotInstance struct {
	Bot              *gotgbot.Bot
	Config           *BotConfig
	Database         *database.Database
	MessageService   *services.MessageService
	TopicService     *services.TopicService
	AIService        *services.AIService
	MessageHandlers  *handlers.MessageHandlers
	CallbackHandlers *handlers.CallbackHandlers
	Dispatcher       *router.Dispatcher
}

// LoadConfig loads configuration from environment
func LoadConfig() (*BotConfig, error) {
	logutils.Info("LoadConfig: entry")

	_ = godotenv.Load()

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		logutils.Error("LoadConfig: TELEGRAM_BOT_TOKEN is not set in .env", nil)
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is not set in .env")
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		logutils.Error("LoadConfig: OPENAI_API_KEY is not set in .env", nil)
		return nil, fmt.Errorf("OPENAI_API_KEY is not set in .env")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "bot.db" // Default database path
	}

	config := &BotConfig{
		BotToken:  botToken,
		OpenAIKey: openaiKey,
		DBPath:    dbPath,
	}

	logutils.Success("LoadConfig: exit")
	return config, nil
}

// InitializeBot creates and initializes all bot components
func InitializeBot(config *BotConfig) (*BotInstance, error) {
	logutils.Info("InitializeBot: entry")

	bot, err := gotgbot.NewBot(config.BotToken, nil)
	if err != nil {
		logutils.Error("InitializeBot: failed to create bot", err)
		return nil, fmt.Errorf("failed to create bot: %v", err)
	}

	db, err := database.NewDatabase(config.DBPath)
	if err != nil {
		logutils.Error("InitializeBot: failed to initialize database", err)
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}

	// Initialize services with the correct signatures
	messageService := services.NewMessageService(config.BotToken, db)
	topicService := services.NewTopicService(config.BotToken, db, httpClient)
	aiService := services.NewAIService(config.OpenAIKey, httpClient)

	// Initialize handlers in the correct order
	commandHandlers := handlers.NewCommandHandlers(messageService, topicService)
	warningHandlers := handlers.NewWarningHandlers(messageService)
	aiHandlers := handlers.NewAIHandlers(messageService, topicService, aiService)
	topicHandlers := handlers.NewTopicHandlers(messageService, topicService)

	// This was the key: Inject the concrete handlers
	callbackHandlers := handlers.NewCallbackHandlers(
		messageService,
		topicHandlers,
		aiHandlers,
		warningHandlers,
	)

	messageHandlers := handlers.NewMessageHandlers(
		commandHandlers,
		aiHandlers,
		topicHandlers,
		warningHandlers,
		messageService,
		bot.User.Username,
	)

	// Initialize the dispatcher, passing handlers and services (as interfaces).
	dispatcher := router.NewDispatcher(
		messageHandlers,
		callbackHandlers,
		messageService,
	)

	// Set the bot's user ID for self-detection in join events
	dispatcher.BotUserID = bot.User.Id

	instance := &BotInstance{
		Bot:              bot,
		Config:           config,
		Database:         db,
		MessageService:   messageService,
		TopicService:     topicService,
		AIService:        aiService,
		MessageHandlers:  messageHandlers,
		CallbackHandlers: callbackHandlers,
		Dispatcher:       dispatcher,
	}

	logutils.Success("InitializeBot: exit", "bot_username", bot.User.Username)
	return instance, nil
}

// Cleanup performs cleanup operations
func (bi *BotInstance) Cleanup() {
	logutils.Info("Cleanup: entry")
	if bi.Database != nil {
		bi.Database.Close()
	}
	logutils.Success("Cleanup: exit")
}
