# Modular AI Telegram Bot Architecture

This document describes the new modular architecture for the Telegram bot, implementing clean architecture principles with proper separation of concerns.

## 🏗️ Architecture Overview

The bot has been refactored into a modular structure with clear boundaries:

```
save-message/
├── internal/
│   ├── services/          # Business logic layer
│   │   ├── message_service.go
│   │   ├── topic_service.go
│   │   └── ai_service.go
│   ├── handlers/          # Request handling layer
│   │   ├── message_handlers.go
│   │   └── callback_handlers.go
│   ├── router/            # Routing layer
│   │   └── dispatcher.go
│   ├── ai/               # AI integration
│   ├── database/         # Data persistence
│   └── config/           # Configuration
├── main.go               # Original monolithic implementation
└── cmd/modular/main.go  # New modular implementation
```

## ✅ 1. Functional Isolation

Each key behavior is implemented as a **dedicated function** with a single purpose:

### MessageService Functions:
- `DeleteMessage(chatID, messageID)` - Deletes messages from Telegram
- `CopyMessageToTopic(chatID, fromChatID, messageID, threadID)` - Copies messages to topics
- `SendMessage(chatID, text, opts)` - Sends messages with options
- `EditMessageText(chatID, messageID, text, opts)` - Edits existing messages
- `AnswerCallbackQuery(callbackID, opts)` - Answers callback queries

### TopicService Functions:
- `GetForumTopics(chatID)` - Retrieves all topics in a forum
- `CreateForumTopic(chatID, name)` - Creates new topics
- `TopicExists(chatID, topicName)` - Checks if topic exists
- `FindTopicByName(chatID, topicName)` - Finds topics by name

### AIService Functions:
- `SuggestFolders(ctx, messageText, existingFolders)` - AI-powered folder suggestions

## ✅ 2. Handler Separation by User Flow

### MessageHandlers:
- `HandleStartCommand()` - Handles /start command
- `HandleHelpCommand()` - Handles /help command
- `HandleTopicsCommand()` - Handles /topics command
- `HandleAddTopicCommand()` - Handles /addtopic command
- `HandleBotMention()` - Handles bot mentions
- `HandleNonGeneralTopicMessage()` - Handles messages in non-General topics
- `HandleGeneralTopicMessage()` - Handles messages in General topic

### CallbackHandlers:
- `HandleNewTopicCreationRequest()` - Handles new topic creation requests
- `HandleRetryCallback()` - Handles retry button clicks
- `HandleShowAllTopicsCallback()` - Shows all existing topics
- `HandleCreateTopicMenuCallback()` - Shows topic creation menu
- `HandleTopicSelectionCallback()` - Handles topic selection
- `HandleTopicNameEntry()` - Handles topic name input

## ✅ 3. Command/Event Routing

The `Dispatcher` routes incoming updates to appropriate handlers:

```go
func (d *Dispatcher) HandleUpdate(update *gotgbot.Update) error {
    if update.CallbackQuery != nil {
        return d.callbackHandlers.HandleCallbackQuery(update)
    }
    
    if update.Message != nil {
        return d.handleMessage(update)
    }
    
    return nil
}
```

The dispatcher includes helper methods for routing decisions:
- `IsEditRequest(update)` - Checks if message is an edit request
- `IsTopicSelection(update)` - Checks if callback is topic selection
- `IsNewTopicPrompt(update)` - Checks if waiting for topic name
- `IsMessageInGeneralTopic(update)` - Checks if message is in General topic

## ✅ 4. Architecture Goals Achieved

### Clean Architecture / Separation of Concerns:
- **Services Layer**: Pure business logic, no HTTP concerns
- **Handlers Layer**: Request/response handling, no business logic
- **Router Layer**: Pure routing logic, no business or HTTP concerns
- **Main Layer**: Only setup and orchestration

### Clear Package Boundaries:
- `handlers/` - Request handling and user interaction
- `services/` - Business logic and external API calls
- `ai/` - AI integration
- `utils/` - Shared utilities
- `router/` - Request routing and dispatching

### Central Dispatcher:
All messages, callbacks, and responses go through the `Dispatcher`, which routes to appropriate handlers.

### Small, Focused Services:
- `MessageService`: Handles all Telegram message operations
- `TopicService`: Handles all topic/forum operations
- `AIService`: Handles all AI-related operations

### Clean Main Function:
The new `mainNew()` function only handles setup and orchestration, with no business logic.

## 🧪 Additional Features

### Comprehensive Logging:
Every handler entry and exit is logged with context:
```go
log.Printf("[MessageHandlers] Handling /start command: ChatID=%d", update.Message.Chat.Id)
```

### Future-Proof Design:
- Easy to add new interaction types (search, tagging, etc.)
- Modular structure allows independent testing
- Clear interfaces make extension straightforward

### Privacy-First:
- No message or user data saved in storage
- All data stays within Telegram
- Clean separation prevents data leakage

### No Hard-Coded Text:
- All text messages are centralized
- Easy to implement i18n in the future
- Consistent messaging across handlers

## 🚀 Usage

To use the new modular architecture:

1. **Run the new implementation**:
   ```bash
   go run cmd/modular/main.go
   ```

2. **Or build and run**:
   ```bash
   go build -o bot_modular cmd/modular/main.go
   ./bot_modular
   ```

## 🔄 Migration Path

The original `main.go` is preserved for reference. The new modular implementation can be gradually adopted:

1. **Phase 1**: Run both implementations in parallel
2. **Phase 2**: Switch to modular implementation
3. **Phase 3**: Remove old implementation

## 🧪 Testing

Each service and handler can be tested independently:

```go
// Test MessageService
messageService := services.NewMessageService(botToken, db)
err := messageService.DeleteMessage(chatID, messageID)

// Test MessageHandlers
messageHandlers := handlers.NewMessageHandlers(messageService, topicService, aiService)
err := messageHandlers.HandleStartCommand(update)

// Test Dispatcher
dispatcher := router.NewDispatcher(messageHandlers, callbackHandlers)
err := dispatcher.HandleUpdate(update)
```

## 📈 Benefits

1. **Maintainability**: Clear separation makes code easier to understand and modify
2. **Testability**: Each component can be tested independently
3. **Scalability**: Easy to add new features without affecting existing code
4. **Debugging**: Clear logging and separation make issues easier to trace
5. **Reusability**: Services can be reused across different handlers
6. **Future-Proof**: Architecture supports easy extension and modification 