package database

import (
	"os"
	"testing"
)

func TestNewDatabase(t *testing.T) {
	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
	}{
		{
			name:    "valid database path",
			dbPath:  ":memory:",
			wantErr: false,
		},
		{
			name:    "invalid database path",
			dbPath:  "/invalid/path/test.db",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDatabase(tt.dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDatabase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && db == nil {
				t.Error("NewDatabase() returned nil database when no error expected")
			}
			if db != nil {
				db.Close()
			}
		})
	}
}

func TestDatabase_UpsertUser(t *testing.T) {
	db, err := NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name      string
		userID    int64
		username  string
		firstName string
		lastName  string
		wantErr   bool
	}{
		{
			name:      "valid user data",
			userID:    123456,
			username:  "testuser",
			firstName: "Test",
			lastName:  "User",
			wantErr:   false,
		},
		{
			name:      "user with empty fields",
			userID:    789012,
			username:  "",
			firstName: "",
			lastName:  "",
			wantErr:   false,
		},
		{
			name:      "update existing user",
			userID:    123456,
			username:  "updateduser",
			firstName: "Updated",
			lastName:  "User",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.UpsertUser(tt.userID, tt.username, tt.firstName, tt.lastName)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpsertUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabase_GetUser(t *testing.T) {
	db, err := NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Insert test user
	testUserID := int64(123456)
	err = db.UpsertUser(testUserID, "testuser", "Test", "User")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	tests := []struct {
		name    string
		userID  int64
		wantErr bool
	}{
		{
			name:    "existing user",
			userID:  testUserID,
			wantErr: false,
		},
		{
			name:    "non-existing user",
			userID:  999999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := db.GetUser(tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && user == nil {
				t.Error("GetUser() returned nil user when no error expected")
			}
			if !tt.wantErr && user != nil {
				if user.ID != tt.userID {
					t.Errorf("GetUser() returned user with ID %d, want %d", user.ID, tt.userID)
				}
			}
		})
	}
}

func TestDatabase_AddTopic(t *testing.T) {
	db, err := NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name            string
		chatID          int64
		topicName       string
		messageThreadID int64
		createdBy       int64
		wantErr         bool
	}{
		{
			name:            "valid topic",
			chatID:          123456,
			topicName:       "Test Topic",
			messageThreadID: 1,
			createdBy:       123,
			wantErr:         false,
		},
		{
			name:            "topic with empty name",
			chatID:          123456,
			topicName:       "",
			messageThreadID: 2,
			createdBy:       123,
			wantErr:         false,
		},
		{
			name:            "duplicate topic (should not error due to INSERT OR IGNORE)",
			chatID:          123456,
			topicName:       "Test Topic",
			messageThreadID: 3,
			createdBy:       123,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.AddTopic(tt.chatID, tt.topicName, tt.messageThreadID, tt.createdBy)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddTopic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabase_GetTopicsByChat(t *testing.T) {
	db, err := NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Insert test topics
	testChatID := int64(123456)
	topics := []struct {
		name            string
		messageThreadID int64
		createdBy       int64
	}{
		{"Topic A", 1, 123},
		{"Topic B", 2, 123},
		{"Topic C", 3, 456},
	}

	for _, topic := range topics {
		err := db.AddTopic(testChatID, topic.name, topic.messageThreadID, topic.createdBy)
		if err != nil {
			t.Fatalf("Failed to insert test topic: %v", err)
		}
	}

	tests := []struct {
		name      string
		chatID    int64
		wantCount int
		wantErr   bool
	}{
		{
			name:      "chat with topics",
			chatID:    testChatID,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "chat without topics",
			chatID:    999999,
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topics, err := db.GetTopicsByChat(tt.chatID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTopicsByChat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(topics) != tt.wantCount {
				t.Errorf("GetTopicsByChat() returned %d topics, want %d", len(topics), tt.wantCount)
			}
		})
	}
}

func TestDatabase_TopicExists(t *testing.T) {
	db, err := NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Insert test topic
	testChatID := int64(123456)
	testTopicName := "Test Topic"
	err = db.AddTopic(testChatID, testTopicName, 1, 123)
	if err != nil {
		t.Fatalf("Failed to insert test topic: %v", err)
	}

	tests := []struct {
		name      string
		chatID    int64
		topicName string
		want      bool
		wantErr   bool
	}{
		{
			name:      "existing topic",
			chatID:    testChatID,
			topicName: testTopicName,
			want:      true,
			wantErr:   false,
		},
		{
			name:      "non-existing topic",
			chatID:    testChatID,
			topicName: "Non-existing Topic",
			want:      false,
			wantErr:   false,
		},
		{
			name:      "non-existing chat",
			chatID:    999999,
			topicName: testTopicName,
			want:      false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := db.TopicExists(tt.chatID, tt.topicName)
			if (err != nil) != tt.wantErr {
				t.Errorf("TopicExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if exists != tt.want {
				t.Errorf("TopicExists() = %v, want %v", exists, tt.want)
			}
		})
	}
}

func TestDatabase_Close(t *testing.T) {
	db, err := NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	err = db.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// Test helper function to create a temporary database file
func createTempDB(t *testing.T) (string, func()) {
	tmpfile, err := os.CreateTemp("", "testdb_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()

	cleanup := func() {
		os.Remove(tmpfile.Name())
	}

	return tmpfile.Name(), cleanup
}
