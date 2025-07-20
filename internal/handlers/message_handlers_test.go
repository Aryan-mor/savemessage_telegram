package handlers

import (
	"save-message/internal/interfaces"
	"testing"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/stretchr/testify/assert"
)

type mockCommandHandlers struct {
	interfaces.MessageHandlersInterface
	StartCalled    *bool
	HelpCalled     *bool
	TopicsCalled   *bool
	AddTopicCalled *bool
	MentionCalled  *bool
}

func (m *mockCommandHandlers) HandleStartCommand(u *gotgbot.Update) error {
	*m.StartCalled = true
	return nil
}
func (m *mockCommandHandlers) HandleHelpCommand(u *gotgbot.Update) error {
	*m.HelpCalled = true
	return nil
}
func (m *mockCommandHandlers) HandleTopicsCommand(u *gotgbot.Update) error {
	*m.TopicsCalled = true
	return nil
}
func (m *mockCommandHandlers) HandleAddTopicCommand(u *gotgbot.Update) error {
	*m.AddTopicCalled = true
	return nil
}
func (m *mockCommandHandlers) HandleBotMention(u *gotgbot.Update) error {
	*m.MentionCalled = true
	return nil
}

type mockWarningHandlers struct {
	interfaces.WarningHandlersInterface
	Called *bool
}

func (m *mockWarningHandlers) HandleNonGeneralTopicMessage(u *gotgbot.Update) error {
	*m.Called = true
	return nil
}

type mockAIHandlers struct {
	interfaces.AIHandlersInterface
	Called *bool
}

func (m *mockAIHandlers) HandleGeneralTopicMessage(u *gotgbot.Update) error {
	*m.Called = true
	return nil
}

type mockTopicHandlers struct {
	interfaces.TopicHandlersInterface
	Called *bool
}

func (m *mockTopicHandlers) HandleTopicNameEntry(u *gotgbot.Update) error {
	*m.Called = true
	return nil
}
func (m *mockTopicHandlers) IsWaitingForTopicName(userID int64) bool { return false }

func TestMessageHandlersDelegation(t *testing.T) {
	update := &gotgbot.Update{Message: &gotgbot.Message{From: &gotgbot.User{Id: 1}}}

	startCalled := false
	helpCalled := false
	topicsCalled := false
	addTopicCalled := false
	mentionCalled := false
	warnCalled := false
	aiCalled := false
	topicCalled := false

	cmd := &mockCommandHandlers{
		StartCalled:    &startCalled,
		HelpCalled:     &helpCalled,
		TopicsCalled:   &topicsCalled,
		AddTopicCalled: &addTopicCalled,
		MentionCalled:  &mentionCalled,
	}
	warn := &mockWarningHandlers{Called: &warnCalled}
	ai := &mockAIHandlers{Called: &aiCalled}
	topic := &mockTopicHandlers{Called: &topicCalled}

	mh := NewMessageHandlers(cmd, ai, topic, warn, nil, "testbot")

	t.Run("delegates HandleStartCommand", func(t *testing.T) {
		startCalled = false
		mh.HandleStartCommand(update)
		assert.True(t, startCalled)
	})
	t.Run("delegates HandleHelpCommand", func(t *testing.T) {
		helpCalled = false
		mh.HandleHelpCommand(update)
		assert.True(t, helpCalled)
	})
	t.Run("delegates HandleTopicsCommand", func(t *testing.T) {
		topicsCalled = false
		mh.HandleTopicsCommand(update)
		assert.True(t, topicsCalled)
	})
	t.Run("delegates HandleAddTopicCommand", func(t *testing.T) {
		addTopicCalled = false
		mh.HandleAddTopicCommand(update)
		assert.True(t, addTopicCalled)
	})
	t.Run("delegates HandleBotMention", func(t *testing.T) {
		mentionCalled = false
		mh.HandleBotMention(update)
		assert.True(t, mentionCalled)
	})
	t.Run("delegates HandleNonGeneralTopicMessage", func(t *testing.T) {
		warnCalled = false
		mh.HandleNonGeneralTopicMessage(update)
		assert.True(t, warnCalled)
	})
	t.Run("delegates HandleGeneralTopicMessage", func(t *testing.T) {
		aiCalled = false
		mh.HandleGeneralTopicMessage(update)
		assert.True(t, aiCalled)
	})
	t.Run("delegates HandleTopicNameEntry", func(t *testing.T) {
		topicCalled = false
		mh.HandleTopicNameEntry(update)
		assert.True(t, topicCalled)
	})
}

func TestIsCommand(t *testing.T) {
	mh := &MessageHandlers{}
	assert.True(t, mh.isCommand(&gotgbot.Update{Message: &gotgbot.Message{Text: "/start"}}))
	assert.False(t, mh.isCommand(&gotgbot.Update{Message: &gotgbot.Message{Text: "not a command"}}))
}

func TestIsBotMention(t *testing.T) {
	mh := &MessageHandlers{BotUsername: "savemessagebot"}

	// Test case 1: Bot is mentioned
	update1 := &gotgbot.Update{
		Message: &gotgbot.Message{
			Text: "hello @savemessagebot how are you?",
			Entities: []gotgbot.MessageEntity{
				{Type: "mention", Offset: 6, Length: 15},
			},
		},
	}
	assert.True(t, mh.IsBotMention(update1))

	// Test case 2: No mention
	update2 := &gotgbot.Update{
		Message: &gotgbot.Message{
			Text: "hello world",
		},
	}
	assert.False(t, mh.IsBotMention(update2))

	// Test case 3: Another bot is mentioned
	update3 := &gotgbot.Update{
		Message: &gotgbot.Message{
			Text: "hello @anotherbot",
			Entities: []gotgbot.MessageEntity{
				{Type: "mention", Offset: 6, Length: 11},
			},
		},
	}
	assert.False(t, mh.IsBotMention(update3))
}
