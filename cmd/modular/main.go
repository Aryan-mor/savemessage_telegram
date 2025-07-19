package main

import (
	"log"
	"os"
	"strings"
	"time"

	"save-message/internal/database"
	"save-message/internal/handlers"
	"save-message/internal/router"
	"save-message/internal/services"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	_ = godotenv.Load()
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	openaiKey := os.Getenv("OPENAI_API_KEY")

	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set in .env")
	}
	if openaiKey == "" {
		log.Fatal("OPENAI_API_KEY is not set in .env")
	}

	// Initialize database
	db, err := database.NewDatabase("bot.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize services
	messageService := services.NewMessageService(botToken, db)
	topicService := services.NewTopicService(botToken, db)
	aiService := services.NewAIService(openaiKey)

	// Initialize handlers
	messageHandlers := handlers.NewMessageHandlers(messageService, topicService, aiService)
	callbackHandlers := handlers.NewCallbackHandlers(messageService, topicService, aiService)

	// Initialize dispatcher
	dispatcher := router.NewDispatcher(messageHandlers, callbackHandlers)

	// Initialize bot
	bot, err := gotgbot.NewBot(botToken, nil)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	log.Printf("Authorized on account %s", bot.User.Username)

	// Start polling for updates
	var offset int64 = 0
	for {
		updates, err := bot.GetUpdates(&gotgbot.GetUpdatesOpts{
			Offset:  offset,
			Timeout: 10,
		})
		if err != nil {
			if !strings.Contains(err.Error(), "context deadline exceeded") {
				log.Printf("GetUpdates error: %v", err)
			}
			time.Sleep(2 * time.Second)
			continue
		}

		for _, update := range updates {
			// Always increment offset for each update to prevent infinite loops
			if update.UpdateId >= offset {
				offset = update.UpdateId + 1
			}

			// Route update to appropriate handler
			err := dispatcher.HandleUpdate(&update)
			if err != nil {
				log.Printf("Error handling update: %v", err)
			}
		}
	}
}
