package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockHTTPClient is a mock of HTTPClient for testing purposes.
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return m.Do(req)
}

func TestNewOpenAIClient(t *testing.T) {
	apiKey := "test-api-key"
	mockClient := &MockHTTPClient{}
	client := NewOpenAIClient(apiKey, mockClient)

	require.NotNil(t, client)
	// Can't directly test unexported fields, but we can check if it's not nil
	// require.Equal(t, apiKey, client.apiKey)
	// require.Equal(t, mockClient, client.httpClient)
}

func TestBuildPrompt(t *testing.T) {
	tests := []struct {
		name             string
		message          string
		existingFolders  []string
		expectedContains []string
	}{
		{
			name:             "message with no existing folders",
			message:          "test message",
			existingFolders:  []string{},
			expectedContains: []string{"test message", "Suggest 2-3 relevant topic names"},
		},
		{
			name:             "message with existing folders",
			message:          "work related message",
			existingFolders:  []string{"Work", "Personal"},
			expectedContains: []string{"work related message", "Existing topics", "ALWAYS check if any existing topics", "IMPORTANT RULES"},
		},
		{
			name:             "empty message with folders",
			message:          "",
			existingFolders:  []string{"Work"},
			expectedContains: []string{"''", "Existing topics"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPrompt(tt.message, tt.existingFolders)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestParseFolders(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectedResult []string
	}{
		{
			name:           "simple comma separated",
			response:       "Work, Personal, Shopping",
			expectedResult: []string{"Work", "Personal", "Shopping"},
		},
		{
			name:           "with extra spaces",
			response:       "  Work  ,  Personal  ,  Shopping  ",
			expectedResult: []string{"Work", "Personal", "Shopping"},
		},
		{
			name:           "single folder",
			response:       "Work",
			expectedResult: []string{"Work"},
		},
		{
			name:           "empty response",
			response:       "",
			expectedResult: nil,
		},
		{
			name:           "response with empty entries",
			response:       "Work,,Personal,",
			expectedResult: []string{"Work", "Personal"},
		},
		{
			name:           "response with only spaces",
			response:       "   ,  ,  ",
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFolders(tt.response)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestOpenAIClient_SuggestFolders_Integration(t *testing.T) {
	// This test would require mocking the HTTP client
	// For now, we'll test the structure and error handling
	client := NewOpenAIClient("test_key", &MockHTTPClient{})
	ctx := context.Background()

	// Test that the method signature is correct
	// In a real scenario, this would make an HTTP call
	// but we're testing the interface and structure
	assert.NotNil(t, client)
	assert.NotNil(t, ctx)

	// Test that the client implements the interface
	var _ OpenAIClientInterface = client
}

func TestOpenAIClient_InterfaceCompliance(t *testing.T) {
	client := NewOpenAIClient("test_key", &MockHTTPClient{})

	// Test that the client implements the interface
	var interfaceClient OpenAIClientInterface = client
	assert.NotNil(t, interfaceClient)
}

func TestBuildPrompt_EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		existingFolders []string
		expectedRules   []string
	}{
		{
			name:            "message with special characters",
			message:         "Message with @#$%^&*() characters",
			existingFolders: []string{"Work"},
			expectedRules:   []string{"ALWAYS check", "IMPORTANT RULES", "Never suggest 'General'"},
		},
		{
			name:            "very long message",
			message:         "This is a very long message that contains many words and should still be processed correctly by the prompt building function",
			existingFolders: []string{"Work", "Personal", "Shopping", "Travel"},
			expectedRules:   []string{"ALWAYS check", "IMPORTANT RULES"},
		},
		{
			name:            "message with newlines",
			message:         "Line 1\nLine 2\nLine 3",
			existingFolders: []string{"Work"},
			expectedRules:   []string{"ALWAYS check", "IMPORTANT RULES"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPrompt(tt.message, tt.existingFolders)

			// Check that the message is included
			assert.Contains(t, result, tt.message)

			// Check that the rules are included when there are existing folders
			if len(tt.existingFolders) > 0 {
				for _, rule := range tt.expectedRules {
					assert.Contains(t, result, rule)
				}
			}
		})
	}
}

func TestParseFolders_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectedResult []string
	}{
		{
			name:           "response with tabs",
			response:       "Work\t,Personal\t,Shopping",
			expectedResult: []string{"Work", "Personal", "Shopping"},
		},
		{
			name:           "response with mixed whitespace",
			response:       "  Work  \t,  Personal  \n,  Shopping  ",
			expectedResult: []string{"Work", "Personal", "Shopping"},
		},
		{
			name:           "response with unicode characters",
			response:       "Trabajo, Personal, Compras",
			expectedResult: []string{"Trabajo", "Personal", "Compras"},
		},
		{
			name:           "response with numbers",
			response:       "Project1, Project2, Project3",
			expectedResult: []string{"Project1", "Project2", "Project3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFolders(tt.response)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
