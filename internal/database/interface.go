package database

// DatabaseInterface defines the interface for database operations
type DatabaseInterface interface {
	UpsertUser(userID int64, username, firstName, lastName string) error
	GetUser(userID int64) (*User, error)
	AddTopic(chatID int64, name string, messageThreadId int64, createdBy int64) error
	GetTopicsByChat(chatID int64) ([]Topic, error)
	TopicExists(chatID int64, name string) (bool, error)
	Close() error
}
