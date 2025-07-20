package logutils

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	// Test that Init doesn't panic
	assert.NotPanics(t, func() {
		Init()
	})

	// Test that logger is initialized
	assert.NotNil(t, logger)
}

func TestInfo(t *testing.T) {
	// Test Info function
	assert.NotPanics(t, func() {
		Info("TestFunction", "key1", "value1", "key2", "value2")
	})
}

func TestWarn(t *testing.T) {
	// Test Warn function
	assert.NotPanics(t, func() {
		Warn("TestFunction", "key1", "value1", "key2", "value2")
	})
}

func TestError(t *testing.T) {
	// Test Error function
	testErr := errors.New("test error")
	assert.NotPanics(t, func() {
		Error("TestFunction", testErr, "key1", "value1", "key2", "value2")
	})
}

func TestDebug(t *testing.T) {
	// Test Debug function
	assert.NotPanics(t, func() {
		Debug("TestFunction", "key1", "value1", "key2", "value2")
	})
}

func TestSuccess(t *testing.T) {
	// Test Success function
	assert.NotPanics(t, func() {
		Success("TestFunction", "key1", "value1", "key2", "value2")
	})
}

func TestSync(t *testing.T) {
	// Test Sync function
	// Sync may fail in some environments but shouldn't panic
	assert.NotPanics(t, func() {
		Sync()
	})
}

func TestLoggerWithNilLogger(t *testing.T) {
	// Temporarily set logger to nil to test auto-initialization
	originalLogger := logger
	logger = nil
	defer func() {
		logger = originalLogger
	}()

	// Test that functions auto-initialize logger
	assert.NotPanics(t, func() {
		Info("AutoInitTest")
		Warn("AutoInitTest")
		Error("AutoInitTest", errors.New("test error"))
		Debug("AutoInitTest")
		Success("AutoInitTest")
	})
}

func TestLoggerWithEnvironment(t *testing.T) {
	// Test with production environment
	originalEnv := os.Getenv("ENV")
	os.Setenv("ENV", "production")
	defer os.Setenv("ENV", originalEnv)

	// Reset logger and test production config
	logger = nil
	assert.NotPanics(t, func() {
		Init()
		Info("ProductionTest")
	})
}

func TestLoggerWithDevelopmentEnvironment(t *testing.T) {
	// Test with development environment
	originalEnv := os.Getenv("ENV")
	os.Setenv("ENV", "development")
	defer os.Setenv("ENV", originalEnv)

	// Reset logger and test development config
	logger = nil
	assert.NotPanics(t, func() {
		Init()
		Info("DevelopmentTest")
	})
}

func TestLoggerWithNoEnvironment(t *testing.T) {
	// Test with no environment variable set
	originalEnv := os.Getenv("ENV")
	os.Unsetenv("ENV")
	defer os.Setenv("ENV", originalEnv)

	// Reset logger and test default config
	logger = nil
	assert.NotPanics(t, func() {
		Init()
		Info("DefaultTest")
	})
}

func TestLoggerFunctionsWithVariousInputs(t *testing.T) {
	tests := []struct {
		name     string
		function func()
	}{
		{
			name: "Info with no fields",
			function: func() {
				Info("TestFunction")
			},
		},
		{
			name: "Warn with no fields",
			function: func() {
				Warn("TestFunction")
			},
		},
		{
			name: "Error with no additional fields",
			function: func() {
				Error("TestFunction", errors.New("test error"))
			},
		},
		{
			name: "Debug with no fields",
			function: func() {
				Debug("TestFunction")
			},
		},
		{
			name: "Success with no fields",
			function: func() {
				Success("TestFunction")
			},
		},
		{
			name: "Info with single field",
			function: func() {
				Info("TestFunction", "key", "value")
			},
		},
		{
			name: "Info with multiple fields",
			function: func() {
				Info("TestFunction", "key1", "value1", "key2", "value2", "key3", "value3")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.function)
		})
	}
}
