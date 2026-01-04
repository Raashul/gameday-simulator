package utils

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

// Logger provides structured logging using slog
type Logger struct {
	slog    *slog.Logger
	logFile *os.File
}

// NewLogger creates a new logger with dual output (console + file)
func NewLogger(level LogLevel) *Logger {
	// Create log file with date/timestamp structure
	logFile, err := createLogFile()
	if err != nil {
		slog.Warn("Failed to create log file, logging to console only", "error", err)
		return &Logger{
			slog: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: toSlogLevel(level),
			})),
		}
	}

	// Use MultiWriter to write to both console and file
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Create JSON handler with the specified level
	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: toSlogLevel(level),
	})

	return &Logger{
		slog:    slog.New(handler),
		logFile: logFile,
	}
}

// toSlogLevel converts our LogLevel to slog.Level
func toSlogLevel(level LogLevel) slog.Level {
	switch level {
	case DEBUG:
		return slog.LevelDebug
	case INFO:
		return slog.LevelInfo
	case WARN:
		return slog.LevelWarn
	case ERROR:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// createLogFile creates a log file with structure: logs/YYYY-MM-DD/HH-MM-SS.log
func createLogFile() (*os.File, error) {
	now := time.Now()

	// Create date-based directory: logs/2024-01-15
	dateDir := filepath.Join("logs", now.Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create timestamp-based filename: 14-30-45.log
	logFileName := now.Format("15-04-05") + ".log"
	logFilePath := filepath.Join(dateDir, logFileName)

	// Open log file for writing
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return file, nil
}

// Close closes the log file (call this on shutdown)
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.slog.Debug(message, mapToAttrs(fields)...)
}

// Info logs an info message
func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.slog.Info(message, mapToAttrs(fields)...)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.slog.Warn(message, mapToAttrs(fields)...)
}

// Error logs an error message
func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.slog.Error(message, mapToAttrs(fields)...)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.slog.Info(fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.slog.Error(fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.slog.Warn(fmt.Sprintf(format, args...))
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.slog.Debug(fmt.Sprintf(format, args...))
}

// mapToAttrs converts a map to slog attributes
func mapToAttrs(fields map[string]interface{}) []interface{} {
	if fields == nil {
		return nil
	}

	attrs := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	return attrs
}
