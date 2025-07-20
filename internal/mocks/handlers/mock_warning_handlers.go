package handlers

import (
	"save-message/internal/interfaces"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type MockWarningHandlers struct{}

var _ interfaces.WarningHandlersInterface = (*MockWarningHandlers)(nil)

func (m *MockWarningHandlers) HandleNonGeneralTopicMessage(u *gotgbot.Update) error { return nil }
func (m *MockWarningHandlers) IsWarningCallback(cb string) bool                     { return false }
func (m *MockWarningHandlers) HandleWarningOkCallback(u *gotgbot.Update) error      { return nil }
