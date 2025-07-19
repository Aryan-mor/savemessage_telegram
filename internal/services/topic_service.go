package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"save-message/internal/database"
)

// TopicService handles all topic-related operations
type TopicService struct {
	botToken string
	db       *database.Database
}

// ForumTopic represents a Telegram forum topic
type ForumTopic struct {
	MessageThreadId int    `json:"message_thread_id"`
	Name            string `json:"name"`
}

// NewTopicService creates a new topic service
func NewTopicService(botToken string, db *database.Database) *TopicService {
	return &TopicService{
		botToken: botToken,
		db:       db,
	}
}

// GetForumTopics fetches all topics in a forum
func (ts *TopicService) GetForumTopics(chatID int64) ([]ForumTopic, error) {
	log.Printf("[TopicService] Getting forum topics: ChatID=%d", chatID)

	// First, check if this is a forum chat
	chatURL := fmt.Sprintf("https://api.telegram.org/bot%s/getChat?chat_id=%d", ts.botToken, chatID)
	chatResp, err := http.Get(chatURL)
	if err == nil {
		defer chatResp.Body.Close()
		chatBody, _ := io.ReadAll(chatResp.Body)
		log.Printf("[TopicService] getChat response: %s", string(chatBody))

		var chatResult struct {
			Ok     bool `json:"ok"`
			Result struct {
				Type    string `json:"type"`
				IsForum bool   `json:"is_forum"`
			} `json:"result"`
		}

		if err := json.Unmarshal(chatBody, &chatResult); err == nil && chatResult.Ok {
			log.Printf("[TopicService] Chat type: %s, Is forum: %v", chatResult.Result.Type, chatResult.Result.IsForum)
		}
	}

	// Try different methods to get forum topics
	methods := []string{
		"getForumTopics",
		"getForumTopicByID",
	}

	for _, method := range methods {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/%s?chat_id=%d", ts.botToken, method, chatID)
		resp, err := http.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		log.Printf("[TopicService] %s response: %s", method, string(body))

		var result struct {
			Ok     bool `json:"ok"`
			Result struct {
				Topics []ForumTopic `json:"topics"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}

		if result.Ok {
			log.Printf("[TopicService] Successfully got topics using %s: %v", method, result.Result.Topics)
			// Update database with found topics
			for _, topic := range result.Result.Topics {
				err := ts.db.AddTopic(chatID, topic.Name, int64(topic.MessageThreadId), 0) // 0 for system-created topics
				if err != nil {
					log.Printf("[TopicService] Error adding topic to database: %v", err)
				}
			}
			return result.Result.Topics, nil
		}
	}

	// If all methods fail, use database
	log.Printf("[TopicService] All forum topic methods failed, using database")
	dbTopics, err := ts.db.GetTopicsByChat(chatID)
	if err != nil {
		log.Printf("[TopicService] Error getting topics from database: %v", err)
		return []ForumTopic{}, nil
	}

	var topics []ForumTopic
	for _, dbTopic := range dbTopics {
		topics = append(topics, ForumTopic{
			MessageThreadId: int(dbTopic.MessageThreadId), // Convert int64 to int for Telegram API
			Name:            dbTopic.Name,
		})
	}
	log.Printf("[TopicService] Using database topics: %v", topics)
	return topics, nil
}

// CreateForumTopic creates a new topic in a forum
func (ts *TopicService) CreateForumTopic(chatID int64, name string) (*ForumTopic, error) {
	log.Printf("[TopicService] Creating forum topic: ChatID=%d, Name=%s", chatID, name)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/createForumTopic", ts.botToken)

	requestBody := map[string]interface{}{
		"chat_id": chatID,
		"name":    name,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		log.Printf("[TopicService] Error creating forum topic request: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[TopicService] Error executing forum topic creation request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok     bool       `json:"ok"`
		Result ForumTopic `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[TopicService] Error parsing forum topic creation response: %v", err)
		return nil, err
	}

	if !result.Ok {
		log.Printf("[TopicService] Failed to create forum topic: %s", string(body))
		return nil, fmt.Errorf("failed to create topic: %s", string(body))
	}

	// Add topic to database
	err = ts.db.AddTopic(chatID, name, int64(result.Result.MessageThreadId), 0) // 0 for system-created topics
	if err != nil {
		log.Printf("[TopicService] Error adding topic to database: %v", err)
	} else {
		log.Printf("[TopicService] Added topic '%s' to database for chat %d", name, chatID)
	}

	log.Printf("[TopicService] Successfully created forum topic: Name=%s, ThreadID=%d", name, result.Result.MessageThreadId)
	return &result.Result, nil
}

// TopicExists checks if a topic exists in the database
func (ts *TopicService) TopicExists(chatID int64, topicName string) (bool, error) {
	log.Printf("[TopicService] Checking if topic exists: ChatID=%d, Name=%s", chatID, topicName)

	topics, err := ts.GetForumTopics(chatID)
	if err != nil {
		log.Printf("[TopicService] Error getting topics for existence check: %v", err)
		return false, err
	}

	for _, topic := range topics {
		if strings.EqualFold(topic.Name, topicName) {
			log.Printf("[TopicService] Topic exists: %s", topicName)
			return true, nil
		}
	}

	log.Printf("[TopicService] Topic does not exist: %s", topicName)
	return false, nil
}

// FindTopicByName finds a topic by name (case-insensitive)
func (ts *TopicService) FindTopicByName(chatID int64, topicName string) (*ForumTopic, error) {
	log.Printf("[TopicService] Finding topic by name: ChatID=%d, Name=%s", chatID, topicName)

	topics, err := ts.GetForumTopics(chatID)
	if err != nil {
		log.Printf("[TopicService] Error getting topics for name search: %v", err)
		return nil, err
	}

	for _, topic := range topics {
		if strings.EqualFold(topic.Name, topicName) {
			log.Printf("[TopicService] Found topic: %s (ThreadID=%d)", topic.Name, topic.MessageThreadId)
			return &topic, nil
		}
	}

	log.Printf("[TopicService] Topic not found: %s", topicName)
	return nil, fmt.Errorf("topic not found: %s", topicName)
}
