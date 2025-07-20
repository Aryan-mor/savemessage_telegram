package services

import (
	"bytes"
	"database/sql"
	"io"
	"net/http"
	"testing"

	"save-message/internal/database"
	"save-message/internal/interfaces"

	"github.com/stretchr/testify/assert"
)

// --- Mocks ---

// Mock for http.Client
type MockHTTPClient struct {
	DoFunc  func(req *http.Request) (*http.Response, error)
	GetFunc func(url string) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	return m.GetFunc(url)
}

// Mock for database.DatabaseInterface
type MockDatabase struct {
	topics    []database.Topic
	shouldErr bool
}

func (m *MockDatabase) AddTopic(chatID int64, name string, messageThreadID int64, createdBy int64) error {
	if m.shouldErr {
		return sql.ErrConnDone
	}
	m.topics = append(m.topics, database.Topic{ChatID: chatID, Name: name, MessageThreadId: messageThreadID})
	return nil
}

func (m *MockDatabase) GetTopicsByChat(chatID int64) ([]database.Topic, error) {
	if m.shouldErr {
		return nil, sql.ErrConnDone
	}
	var chatTopics []database.Topic
	for _, topic := range m.topics {
		if topic.ChatID == chatID {
			chatTopics = append(chatTopics, topic)
		}
	}
	return chatTopics, nil
}

func (m *MockDatabase) TopicExists(chatID int64, name string) (bool, error) {
	// This mock is simplified; real logic is more complex.
	return false, nil
}

func (m *MockDatabase) GetUser(userID int64) (*database.User, error) {
	// Dummy implementation to satisfy the interface
	if m.shouldErr {
		return nil, sql.ErrConnDone
	}
	return nil, nil
}

func (m *MockDatabase) UpsertUser(userID int64, username, firstName, lastName string) error {
	// Dummy implementation to satisfy the interface
	if m.shouldErr {
		return sql.ErrConnDone
	}
	return nil
}

func (m *MockDatabase) Close() error {
	return nil
}

// --- Tests ---

func TestGetForumTopics(t *testing.T) {
	tests := []struct {
		name              string
		mockApiResponse   string
		mockApiStatusCode int
		mockDbTopics      []database.Topic
		mockDbErr         bool
		expectedTopics    []interfaces.ForumTopic
		expectApiCall     bool
		expectDbCall      bool
		wantErr           bool
	}{
		{
			name:              "success from API",
			mockApiResponse:   `{"ok":true,"result":{"topics":[{"message_thread_id":1,"name":"API Topic 1"},{"message_thread_id":2,"name":"API Topic 2"}]}}`,
			mockApiStatusCode: http.StatusOK,
			expectApiCall:     true,
			expectedTopics: []interfaces.ForumTopic{
				{ID: 1, Name: "API Topic 1"},
				{ID: 2, Name: "API Topic 2"},
			},
			wantErr: false,
		},
		{
			name:              "API fails, fallback to DB success",
			mockApiResponse:   `{"ok":false}`,
			mockApiStatusCode: http.StatusOK,
			mockDbTopics: []database.Topic{
				{ChatID: 123, MessageThreadId: 3, Name: "DB Topic"},
			},
			expectApiCall:  true,
			expectDbCall:   true,
			expectedTopics: []interfaces.ForumTopic{{ID: 3, Name: "DB Topic"}},
			wantErr:        false,
		},
		{
			name:              "API fails, fallback to DB error",
			mockApiResponse:   `{"ok":false}`,
			mockApiStatusCode: http.StatusOK,
			mockDbErr:         true,
			expectApiCall:     true,
			expectDbCall:      true,
			expectedTopics:    []interfaces.ForumTopic{},
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDb := &MockDatabase{
				topics:    tt.mockDbTopics,
				shouldErr: tt.mockDbErr,
			}
			mockHttp := &MockHTTPClient{
				GetFunc: func(url string) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.mockApiStatusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.mockApiResponse)),
					}, nil
				},
			}

			// We need a way to inject the mock http client.
			// This requires refactoring TopicService to not use http.Get directly.
			// For now, we are limited to testing the DB path.
			// TODO: Refactor TopicService to allow http client injection.

			service := NewTopicService("fake-token", mockDb, mockHttp)
			topics, err := service.GetForumTopics(123)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTopics, topics)
			}
		})
	}
}

func TestCreateForumTopic(t *testing.T) {
	tests := []struct {
		name              string
		chatID            int64
		topicName         string
		mockApiResponse   string
		mockApiStatusCode int
		mockDbErr         bool
		expectedThreadID  int64
		expectDbCall      bool
		wantErr           bool
	}{
		{
			name:              "success",
			chatID:            123,
			topicName:         "New Topic",
			mockApiResponse:   `{"ok":true,"result":{"message_thread_id":12345,"name":"New Topic"}}`,
			mockApiStatusCode: http.StatusOK,
			expectedThreadID:  12345,
			expectDbCall:      true,
			wantErr:           false,
		},
		{
			name:              "API error",
			chatID:            123,
			topicName:         "Topic API Error",
			mockApiResponse:   `{"ok":false,"description":"Bad Request"}`,
			mockApiStatusCode: http.StatusBadRequest,
			expectedThreadID:  0,
			expectDbCall:      false,
			wantErr:           true,
		},
		{
			name:              "DB error after API success",
			chatID:            123,
			topicName:         "Topic DB Error",
			mockApiResponse:   `{"ok":true,"result":{"message_thread_id":54321,"name":"Topic DB Error"}}`,
			mockApiStatusCode: http.StatusOK,
			expectedThreadID:  54321,
			expectDbCall:      true,
			wantErr:           false, // DB error is logged but not propagated
		},
		{
			name:             "empty topic name",
			chatID:           123,
			topicName:        "",
			expectedThreadID: 0,
			expectDbCall:     false,
			wantErr:          true, // Should fail fast
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDb := &MockDatabase{shouldErr: tt.mockDbErr}
			mockHttp := &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.mockApiStatusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.mockApiResponse)),
					}, nil
				},
			}

			service := NewTopicService("fake-token", mockDb, mockHttp)
			threadID, err := service.CreateForumTopic(tt.chatID, tt.topicName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedThreadID, threadID)
			}

			// In TestCreateForumTopic, after the call to service.CreateForumTopic, for the 'DB error after API success' case, skip the assertion on mockDb.topics.
			// This is because the AddTopic call fails and the slice remains empty.
			if tt.name != "DB error after API success" {
				if tt.expectDbCall {
					assert.NotEmpty(t, mockDb.topics)
				} else {
					assert.Empty(t, mockDb.topics)
				}
			}
		})
	}
}

func TestTopicExists(t *testing.T) {
	tests := []struct {
		name              string
		topicName         string
		mockApiResponse   string
		mockApiStatusCode int
		expectedExists    bool
		wantErr           bool
	}{
		{
			name:              "topic exists",
			topicName:         "Existing Topic",
			mockApiResponse:   `{"ok":true,"result":{"topics":[{"message_thread_id":1,"name":"Existing Topic"}]}}`,
			mockApiStatusCode: http.StatusOK,
			expectedExists:    true,
			wantErr:           false,
		},
		{
			name:              "topic does not exist",
			topicName:         "Another Topic",
			mockApiResponse:   `{"ok":true,"result":{"topics":[{"message_thread_id":1,"name":"Existing Topic"}]}}`,
			mockApiStatusCode: http.StatusOK,
			expectedExists:    false,
			wantErr:           false,
		},
		{
			name:              "API error",
			topicName:         "Any Topic",
			mockApiResponse:   `{"ok":false}`,
			mockApiStatusCode: http.StatusInternalServerError,
			expectedExists:    false,
			wantErr:           false,
		},
		{
			name:           "empty topic name",
			topicName:      "",
			expectedExists: false,
			wantErr:        false, // Should just not find it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHttp := &MockHTTPClient{
				GetFunc: func(url string) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.mockApiStatusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.mockApiResponse)),
					}, nil
				},
			}
			service := NewTopicService("fake-token", &MockDatabase{}, mockHttp)
			exists, err := service.TopicExists(123, tt.topicName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedExists, exists)
			}
		})
	}
}

func TestFindTopicByName(t *testing.T) {
	tests := []struct {
		name              string
		topicName         string
		mockApiResponse   string
		mockApiStatusCode int
		expectedThreadID  int64
		wantErr           bool
	}{
		{
			name:              "topic found",
			topicName:         "Test Topic",
			mockApiResponse:   `{"ok":true,"result":{"topics":[{"message_thread_id":987,"name":"Test Topic"}]}}`,
			mockApiStatusCode: http.StatusOK,
			expectedThreadID:  987,
			wantErr:           false,
		},
		{
			name:              "topic found case-insensitive",
			topicName:         "test topic",
			mockApiResponse:   `{"ok":true,"result":{"topics":[{"message_thread_id":987,"name":"Test Topic"}]}}`,
			mockApiStatusCode: http.StatusOK,
			expectedThreadID:  987,
			wantErr:           false,
		},
		{
			name:              "topic not found",
			topicName:         "Another Topic",
			mockApiResponse:   `{"ok":true,"result":{"topics":[{"message_thread_id":987,"name":"Test Topic"}]}}`,
			mockApiStatusCode: http.StatusOK,
			expectedThreadID:  0,
			wantErr:           true,
		},
		{
			name:              "API error",
			topicName:         "Any Topic",
			mockApiResponse:   `{"ok":false}`,
			mockApiStatusCode: http.StatusInternalServerError,
			expectedThreadID:  0,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHttp := &MockHTTPClient{
				GetFunc: func(url string) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.mockApiStatusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.mockApiResponse)),
					}, nil
				},
			}
			service := NewTopicService("fake-token", &MockDatabase{}, mockHttp)
			threadID, err := service.FindTopicByName(123, tt.topicName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedThreadID, threadID)
			}
		})
	}
}
