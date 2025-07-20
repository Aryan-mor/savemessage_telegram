package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockOpenAIClient is a mock of the OpenAIClientInterface
type MockOpenAIClient struct {
	SuggestFoldersFunc func(ctx context.Context, messageText string, existingFolders []string) ([]string, error)
}

func (m *MockOpenAIClient) SuggestFolders(ctx context.Context, messageText string, existingFolders []string) ([]string, error) {
	return m.SuggestFoldersFunc(ctx, messageText, existingFolders)
}

func TestAIService_SuggestFolders(t *testing.T) {
	tests := []struct {
		name                string
		messageText         string
		existingFolders     []string
		mockSuggestions     []string
		mockErr             error
		expectedSuggestions []string
		wantErr             bool
	}{
		{
			name:                "success",
			messageText:         "test message",
			existingFolders:     []string{"Work"},
			mockSuggestions:     []string{"Personal", "Projects"},
			expectedSuggestions: []string{"Personal", "Projects"},
			wantErr:             false,
		},
		{
			name:            "OpenAI client error",
			messageText:     "another message",
			existingFolders: []string{},
			mockErr:         errors.New("API error"),
			wantErr:         true,
		},
		{
			name:                "empty message text",
			messageText:         "",
			existingFolders:     []string{"General"},
			mockSuggestions:     []string{},
			expectedSuggestions: []string{},
			wantErr:             false,
		},
		{
			name:                "no existing folders",
			messageText:         "message",
			existingFolders:     []string{},
			mockSuggestions:     []string{"Inbox"},
			expectedSuggestions: []string{"Inbox"},
			wantErr:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockOpenAIClient{
				SuggestFoldersFunc: func(ctx context.Context, messageText string, existingFolders []string) ([]string, error) {
					return tt.mockSuggestions, tt.mockErr
				},
			}

			// The service now needs an http client, but we are mocking the layer above it.
			// Pass nil for the http client as it won't be used by the mock.
			service := &AIService{openAIClient: mockClient}

			suggestions, err := service.SuggestFolders(context.Background(), tt.messageText, tt.existingFolders)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSuggestions, suggestions)
			}
		})
	}
}
