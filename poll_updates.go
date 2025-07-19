package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set in .env")
	}

	offset := 0
	for {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=10&offset=%d", token, offset)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error polling: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Ok     bool              `json:"ok"`
			Result []json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("Error decoding: %v", err)
			continue
		}
		for _, update := range result.Result {
			fmt.Printf("Update: %s\n", string(update))
			// Extract update_id to advance offset
			var u struct {
				UpdateId int `json:"update_id"`
			}
			_ = json.Unmarshal(update, &u)
			if u.UpdateId >= offset {
				offset = u.UpdateId + 1
			}
		}
	}
}
