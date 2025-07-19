package main

import (
	"log"
	"strings"
	"time"

	"save-message/internal/setup"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func main() {
	// Load configuration
	config, err := setup.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize bot instance
	botInstance, err := setup.InitializeBot(config)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}
	defer botInstance.Cleanup()

	// Start polling for updates
	var offset int64 = 0
	for {
		updates, err := botInstance.Bot.GetUpdates(&gotgbot.GetUpdatesOpts{
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
			err := botInstance.Dispatcher.HandleUpdate(&update)
			if err != nil {
				log.Printf("Error handling update: %v", err)
			}
		}
	}
}
