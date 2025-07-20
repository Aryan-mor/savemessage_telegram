package interfaces

type TopicServiceInterface interface {
	GetForumTopics(chatID int64) ([]ForumTopic, error)
	CreateForumTopic(chatID int64, name string) (int64, error)
	TopicExists(chatID int64, name string) (bool, error)
	FindTopicByName(chatID int64, name string) (int64, error)
}

type ForumTopic struct {
	Name string
	ID   int64
}
