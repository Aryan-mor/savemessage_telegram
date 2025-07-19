package main

import (
	"fmt"
	"log"
	"os"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	bot, err := gotgbot.NewBot(botToken, nil)
	if err != nil {
		log.Fatal(err)
	}
	var offset int64 = 0
	for {
		updates, err := bot.GetUpdates(&gotgbot.GetUpdatesOpts{
			Offset:  offset,
			Timeout: 10,
		})
		if err != nil {
			log.Println("GetUpdates error:", err)
			continue
		}
		for _, update := range updates {
			offset = update.UpdateId + 1
			fmt.Printf("RAW UPDATE: %+v\n", update)
		}
	}
}
