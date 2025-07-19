package setup

import (
	"fmt"
	"log"
	"os"

	"save-message/internal/database"
	"save-message/internal/handlers"
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
	log.Printf("[Setup] Loading configuration from environment")

	_ = godotenv.Load()

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is not set in .env")
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
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

	log.Printf("[Setup] Configuration loaded successfully")
	return config, nil
}

// InitializeBot creates and initializes all bot components
func InitializeBot(config *BotConfig) (*BotInstance, error) {
	log.Printf("[Setup] Initializing bot components")

	// Initialize database
	db, err := database.NewDatabase(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	// Initialize services
	messageService := services.NewMessageService(config.BotToken, db)
	topicService := services.NewTopicService(config.BotToken, db)
	aiService := services.NewAIService(config.OpenAIKey)

	// Initialize handlers
	messageHandlers := handlers.NewMessageHandlers(messageService, topicService, aiService)
	callbackHandlers := handlers.NewCallbackHandlers(messageService, topicService, aiService)

	// Initialize dispatcher
	dispatcher := router.NewDispatcher(messageHandlers, callbackHandlers)

	// Initialize bot
	bot, err := gotgbot.NewBot(config.BotToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %v", err)
	}

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

	log.Printf("[Setup] Bot initialized successfully: %s", bot.User.Username)
	return instance, nil
}

// Cleanup performs cleanup operations
func (bi *BotInstance) Cleanup() {
	log.Printf("[Setup] Cleaning up bot instance")
	if bi.Database != nil {
		bi.Database.Close()
	}
}
