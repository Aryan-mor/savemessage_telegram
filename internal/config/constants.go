package config

import "time"

// Bot messages and text constants
const (
	// Welcome messages
	WelcomeMessage = "Save Message is your personal assistant inside Telegram.\n\nIt helps you organize your saved messages using Topics and smart suggestions — without using any commands.\nYou can categorize, edit, and retrieve your notes easily with inline buttons.\n\n🛡️ 100% private: all your content stays inside Telegram.\n\nJust write — we'll handle the rest."

	// Help message
	HelpMessage = `🤖 **Save Message Bot Help**

**How to use:**
• Simply send any message and the bot will suggest relevant folders
• Click on a suggested folder to save your message there
• Use "📁 Show All Topics" to browse all existing topics

**Important:** ⚠️ **Don't create topics manually in Save message group!** Let the bot create them automatically when you save messages. This ensures proper organization and prevents confusion.

**Tips:**
• The bot uses AI to suggest relevant folders
• Existing topics show with 📁 icon, new ones with ➕
• Messages are automatically cleaned from General topic after saving
• Success messages auto-delete after 1 minute`

	// Error messages
	ErrorMessageNotFound     = "❌ Error: Message not found. Please try again."
	ErrorMessageFailed       = "❌ Failed to get topics. Please try again."
	ErrorMessageNoTopics     = "📁 No topics found yet. Send a message to create your first topic!"
	ErrorMessageCreateFailed = "❌ Failed to create topic. Please try again."
	ErrorMessageUnknown      = "❓ Unknown action. Please try again."

	// Success messages
	SuccessMessageRetry = "🔄 Retrying... Please send your message again."
	SuccessMessageSaved = "✅ Message saved to topic: "

	// Warning messages
	WarningNonGeneralTopic = "⚠️ **Please send messages only in the General topic!**\n\nThis message will be removed automatically in 1 minute."

	// UI elements
	ButtonTextCreateNewTopic    = "📝 Create New Topic"
	ButtonTextShowAllTopics     = "📁 Show All Topics"
	ButtonTextBackToSuggestions = "⬅️ Back to Suggestions"
	ButtonTextTryAgain          = "🔄 Try Again"
	ButtonTextOk                = "Ok"

	// Menu messages
	BotMenuMessage             = "🤖 **Bot Menu**\n\nWhat would you like to do?"
	ChooseOptionMessage        = "Choose an option:"
	ChooseFolderMessage        = "Choose a folder:"
	ChooseFromAllTopicsMessage = "Choose from all existing topics:"

	// Topic creation messages
	TopicNamePrompt          = "📝 Please enter the name for your new topic:"
	TopicNameEmptyError      = "❌ Topic name cannot be empty. Please try again."
	TopicNameExistsError     = "❌ A topic with this name already exists. Please choose a different name."
	TopicCreationMenuMessage = "📝 **Create New Topic**\n\nPlease send the name of the topic you want to create:"

	// Topic list messages
	TopicsListHeader          = "📁 **Your Topics:**\n"
	NoTopicsDiscoveredMessage = "📁 No topics discovered yet. Create some topics and the bot will remember them!"

	// AI processing messages
	AIProcessingMessage = "🤔 Thinking..."
	AIFailedMessage     = "Sorry, I couldn't suggest folders right now."

	// Callback data prefixes
	CallbackPrefixCreateNewFolder           = "create_new_folder_"
	CallbackPrefixRetry                     = "retry_"
	CallbackPrefixShowAllTopics             = "show_all_topics_"
	CallbackPrefixBackToSuggestions         = "back_to_suggestions_"
	CallbackPrefixDetectMessageOnOtherTopic = "detectMessageOnOtherTopic_ok_"
	CallbackDataCreateTopicMenu             = "create_topic_menu"
	CallbackDataShowAllTopicsMenu           = "show_all_topics_menu"

	// Bot usernames (for mention detection)
	BotUsername1 = "@savemessagbot"
	BotUsername2 = "@savemessagebot"

	// Default values
	DefaultDatabasePath           = "bot.db"
	DefaultPollingTimeout         = 10
	DefaultRetryDelay             = 2 * time.Second
	DefaultWarningAutoDeleteDelay = 60 * time.Second
	DefaultMessageAutoDeleteDelay = 1 * time.Second

	// Icons
	IconFolder    = "📁"
	IconNewFolder = "➕"
	IconCreate    = "📝"
	IconRetry     = "🔄"
	IconBack      = "⬅️"
	IconBot       = "🤖"
	IconWarning   = "⚠️"
	IconError     = "❌"
	IconSuccess   = "✅"
	IconThinking  = "🤔"
)
