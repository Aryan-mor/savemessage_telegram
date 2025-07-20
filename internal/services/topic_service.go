package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"save-message/internal/database"
	"save-message/internal/interfaces"
	"save-message/internal/logutils"
)

// TopicService handles all topic-related operations
type TopicService struct {
	botToken string
	db       database.DatabaseInterface
	client   interfaces.HTTPClient
}

// NewTopicService creates a new topic service
func NewTopicService(botToken string, db database.DatabaseInterface, client interfaces.HTTPClient) *TopicService {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &TopicService{
		botToken: botToken,
		db:       db,
		client:   client,
	}
}

var _ interfaces.TopicServiceInterface = (*TopicService)(nil)

// GetForumTopics fetches all topics in a forum
func (ts *TopicService) GetForumTopics(chatID int64) ([]interfaces.ForumTopic, error) {
	logutils.Info("GetForumTopics", "chatID", chatID)

	// First, check if this is a forum chat
	chatURL := fmt.Sprintf("https://api.telegram.org/bot%s/getChat?chat_id=%d", ts.botToken, chatID)
	chatResp, err := ts.client.Get(chatURL)
	if err == nil {
		defer chatResp.Body.Close()
		chatBody, _ := io.ReadAll(chatResp.Body)
		logutils.Debug("GetForumTopics", "getChat_response", string(chatBody))

		var chatResult struct {
			Ok     bool `json:"ok"`
			Result struct {
				Type    string `json:"type"`
				IsForum bool   `json:"is_forum"`
			} `json:"result"`
		}

		if err := json.Unmarshal(chatBody, &chatResult); err == nil && chatResult.Ok {
			logutils.Debug("GetForumTopics", "chat_type", chatResult.Result.Type, "is_forum", chatResult.Result.IsForum)
		}
	} else {
		logutils.Warn("GetForumTopics: GetChatFailed", "error", err.Error(), "chatID", chatID)
	}

	// Try different methods to get forum topics
	methods := []string{
		"getForumTopics",
		"getForumTopicByID",
	}

	for _, method := range methods {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/%s?chat_id=%d", ts.botToken, method, chatID)
		resp, err := ts.client.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		logutils.Debug("GetForumTopics", "method", method, "response", string(body))

		var result struct {
			Ok     bool `json:"ok"`
			Result struct {
				Topics []interfaces.ForumTopic `json:"topics"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}

		if result.Ok {
			logutils.Success("GetForumTopics", "method", method, "topics_count", len(result.Result.Topics))
			// Update database with found topics
			for _, topic := range result.Result.Topics {
				err := ts.db.AddTopic(chatID, topic.Name, topic.ID, 0) // 0 for system-created topics
				if err != nil {
					logutils.Error("GetForumTopics", err, "chatID", chatID, "topic_name", topic.Name)
				}
			}
			return result.Result.Topics, nil
		}
	}

	// If all methods fail, use database
	logutils.Warn("GetForumTopics", "message", "All API methods failed, falling back to database", "chatID", chatID)
	dbTopics, err := ts.db.GetTopicsByChat(chatID)
	if err != nil {
		logutils.Error("GetForumTopics", err, "chatID", chatID)
		return []interfaces.ForumTopic{}, nil
	}

	var topics []interfaces.ForumTopic
	for _, dbTopic := range dbTopics {
		topics = append(topics, interfaces.ForumTopic{
			ID:   dbTopic.MessageThreadId,
			Name: dbTopic.Name,
		})
	}
	logutils.Success("GetForumTopics", "database_topics_count", len(topics), "chatID", chatID)
	return topics, nil
}

// CreateForumTopic creates a new topic in a forum
func (ts *TopicService) CreateForumTopic(chatID int64, name string) (int64, error) {
	logutils.Info("CreateForumTopic", "chatID", chatID, "name", name)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/createForumTopic", ts.botToken)

	requestBody := map[string]interface{}{
		"chat_id": chatID,
		"name":    name,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		logutils.Error("CreateForumTopic: CreateRequest", err, "chatID", chatID, "name", name)
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := ts.client
	resp, err := client.Do(req)
	if err != nil {
		logutils.Error("CreateForumTopic", err, "chatID", chatID, "name", name)
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Ok     bool                  `json:"ok"`
		Result interfaces.ForumTopic `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logutils.Error("CreateForumTopic", err, "response_body", string(body))
		return 0, err
	}

	if !result.Ok {
		logutils.Error("CreateForumTopic", fmt.Errorf("failed to create topic: %s", string(body)), "chatID", chatID, "name", name)
		return 0, fmt.Errorf("failed to create topic: %s", string(body))
	}

	// Add topic to database
	err = ts.db.AddTopic(chatID, name, result.Result.ID, 0) // 0 for system-created topics
	if err != nil {
		logutils.Error("CreateForumTopic", err, "chatID", chatID, "name", name, "threadID", result.Result.ID)
	} else {
		logutils.Success("CreateForumTopic", "chatID", chatID, "name", name, "threadID", result.Result.ID)
	}

	return result.Result.ID, nil
}

// TopicExists checks if a topic exists in the database
func (ts *TopicService) TopicExists(chatID int64, topicName string) (bool, error) {
	logutils.Info("TopicExists", "chatID", chatID, "topicName", topicName)

	topics, err := ts.GetForumTopics(chatID)
	if err != nil {
		logutils.Error("TopicExists", err, "chatID", chatID, "topicName", topicName)
		return false, err
	}

	for _, topic := range topics {
		if strings.EqualFold(topic.Name, topicName) {
			logutils.Success("TopicExists", "topicName", topicName, "exists", true)
			return true, nil
		}
	}

	logutils.Success("TopicExists", "topicName", topicName, "exists", false)
	return false, nil
}

// FindTopicByName finds a topic by name (case-insensitive)
func (ts *TopicService) FindTopicByName(chatID int64, topicName string) (int64, error) {
	logutils.Info("FindTopicByName", "chatID", chatID, "topicName", topicName)

	topics, err := ts.GetForumTopics(chatID)
	if err != nil {
		logutils.Error("FindTopicByName", err, "chatID", chatID, "topicName", topicName)
		return 0, err
	}

	for _, topic := range topics {
		if strings.EqualFold(topic.Name, topicName) {
			logutils.Success("FindTopicByName", "topicName", topicName, "threadID", topic.ID)
			return topic.ID, nil
		}
	}

	logutils.Warn("FindTopicByName", "message", "Topic not found", "topicName", topicName)
	return 0, fmt.Errorf("topic not found: %s", topicName)
}
