package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

type User struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	CreatedAt time.Time
}

type Topic struct {
	ID              int
	ChatID          int64
	Name            string
	MessageThreadId int64
	CreatedBy       int64
	CreatedAt       time.Time
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &Database{db: db}, nil
}

func createTables(db *sql.DB) error {
	// Create users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			username TEXT,
			first_name TEXT,
			last_name TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create topics table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS topics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			message_thread_id INTEGER,
			created_by INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(chat_id, name)
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

// UpsertUser adds or updates a user
func (d *Database) UpsertUser(userID int64, username, firstName, lastName string) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO users (id, username, first_name, last_name, created_at)
		VALUES (?, ?, ?, ?, COALESCE((SELECT created_at FROM users WHERE id = ?), CURRENT_TIMESTAMP))
	`, userID, username, firstName, lastName, userID)
	return err
}

// GetUser retrieves a user by ID
func (d *Database) GetUser(userID int64) (*User, error) {
	var user User
	err := d.db.QueryRow(`
		SELECT id, username, first_name, last_name, created_at
		FROM users WHERE id = ?
	`, userID).Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// AddTopic adds a new topic
func (d *Database) AddTopic(chatID int64, name string, messageThreadId int64, createdBy int64) error {
	_, err := d.db.Exec(`
		INSERT OR IGNORE INTO topics (chat_id, name, message_thread_id, created_by)
		VALUES (?, ?, ?, ?)
	`, chatID, name, messageThreadId, createdBy)
	return err
}

// GetTopicsByChat retrieves all topics for a chat
func (d *Database) GetTopicsByChat(chatID int64) ([]Topic, error) {
	rows, err := d.db.Query(`
		SELECT id, chat_id, name, message_thread_id, created_by, created_at
		FROM topics WHERE chat_id = ?
		ORDER BY name
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []Topic
	for rows.Next() {
		var topic Topic
		err := rows.Scan(&topic.ID, &topic.ChatID, &topic.Name, &topic.MessageThreadId, &topic.CreatedBy, &topic.CreatedAt)
		if err != nil {
			return nil, err
		}
		topics = append(topics, topic)
	}
	return topics, nil
}

// TopicExists checks if a topic exists in a chat
func (d *Database) TopicExists(chatID int64, name string) (bool, error) {
	var exists int
	err := d.db.QueryRow(`
		SELECT 1 FROM topics WHERE chat_id = ? AND name = ?
	`, chatID, name).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}
