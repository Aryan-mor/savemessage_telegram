# Telegram Assistant Bot – Save Message

A privacy-first, smart Telegram bot to enhance the "Saved Messages" experience using Telegram Topics, inline buttons, and AI (future phases). This starter is written in Go with clean architecture and zero message storage.

## Features (Phase 1)
- Listens for messages in a specific forum topic ("General") in a one-person group
- Replies with a placeholder message (no message storage)
- Clean, scalable project structure
- Loads bot token securely from `.env`
- Graceful error handling and logging

## Folder Structure
```
.
├── main.go
├── go.mod
├── internal/
│   ├── bot/
│   │   └── handler.go
│   ├── config/
│   │   └── env.go
│   └── router/
│       └── router.go
└── .env.example
```

## Getting Started

1. **Clone the repo**
2. **Copy `.env.example` to `.env` and add your Telegram bot token:**
   ```
   cp .env.example .env
   # Edit .env and set TELEGRAM_BOT_TOKEN
   ```
3. **Install dependencies:**
   ```
   go mod tidy
   ```
4. **Run the bot:**
   ```
   go run main.go
   ```

## Requirements
- Go 1.21+
- Telegram bot token (create via [BotFather](https://t.me/BotFather))
- Add the bot to a group with topics enabled, make it admin
- Ensure a topic named "General" exists

## Notes
- No database, no file storage, no user data retention
- All replies are inline (no slash commands)
- Future phases will add AI-based categorization and smart search

---
MIT License 