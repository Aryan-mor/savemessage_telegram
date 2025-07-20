package logutils

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

// Init initializes the global logger
func Init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"

	// Use console encoding for development
	if os.Getenv("ENV") != "production" {
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	zapLogger, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	logger = zapLogger.Sugar()
}

// Info logs an info level message with structured fields
func Info(funcName string, kv ...any) {
	if logger == nil {
		Init()
	}
	logger.Infow("▶️ "+funcName, kv...)
}

// Warn logs a warning level message with structured fields
func Warn(funcName string, kv ...any) {
	if logger == nil {
		Init()
	}
	logger.Warnw("⚠️ "+funcName, kv...)
}

// Error logs an error level message with structured fields
func Error(funcName string, err error, kv ...any) {
	if logger == nil {
		Init()
	}
	allKv := append([]any{"error", err.Error()}, kv...)
	logger.Errorw("❌ "+funcName, allKv...)
}

// Debug logs a debug level message with structured fields
func Debug(funcName string, kv ...any) {
	if logger == nil {
		Init()
	}
	logger.Debugw("🔍 "+funcName, kv...)
}

// Success logs a success message with structured fields
func Success(funcName string, kv ...any) {
	if logger == nil {
		Init()
	}
	logger.Infow("✅ "+funcName, kv...)
}

// Sync flushes any buffered log entries
func Sync() error {
	if logger == nil {
		return nil
	}
	return logger.Sync()
}
