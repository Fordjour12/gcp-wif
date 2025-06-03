// Package logging provides structured logging functionality
// for the GCP Workload Identity Federation CLI tool.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging functionality
type Logger struct {
	level   LogLevel
	verbose bool
	writer  io.Writer
	slogger *slog.Logger
}

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	Level     LogLevel
	Verbose   bool
	Writer    io.Writer
	FilePath  string
	AddSource bool
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:     LevelInfo,
		Verbose:   false,
		Writer:    os.Stderr,
		AddSource: false,
	}
}

// NewLogger creates a new logger instance
func NewLogger(config *LoggerConfig) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	writer := config.Writer
	if writer == nil {
		writer = os.Stderr
	}

	// If file path is provided, create or append to file
	if config.FilePath != "" {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(config.FilePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writer = file
	}

	// Create structured logger
	opts := &slog.HandlerOptions{
		Level:     slogLevelFromLogLevel(config.Level),
		AddSource: config.AddSource,
	}

	var handler slog.Handler
	if config.Verbose {
		handler = slog.NewTextHandler(writer, opts)
	} else {
		handler = slog.NewJSONHandler(writer, opts)
	}

	slogger := slog.New(handler)

	return &Logger{
		level:   config.Level,
		verbose: config.Verbose,
		writer:  writer,
		slogger: slogger,
	}, nil
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	if l.level <= LevelDebug {
		l.slogger.Debug(msg, args...)
		if l.verbose {
			l.printColored(LevelDebug, msg, args...)
		}
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	if l.level <= LevelInfo {
		l.slogger.Info(msg, args...)
		if l.verbose {
			l.printColored(LevelInfo, msg, args...)
		}
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	if l.level <= LevelWarn {
		l.slogger.Warn(msg, args...)
		if l.verbose {
			l.printColored(LevelWarn, msg, args...)
		}
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	if l.level <= LevelError {
		l.slogger.Error(msg, args...)
		if l.verbose {
			l.printColored(LevelError, msg, args...)
		}
	}
}

// WithField adds a field to subsequent log entries
func (l *Logger) WithField(key string, value any) *Logger {
	newLogger := *l
	newLogger.slogger = l.slogger.With(key, value)
	return &newLogger
}

// WithFields adds multiple fields to subsequent log entries
func (l *Logger) WithFields(fields map[string]any) *Logger {
	newLogger := *l
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	newLogger.slogger = l.slogger.With(args...)
	return &newLogger
}

// printColored prints colored output for verbose mode
func (l *Logger) printColored(level LogLevel, msg string, args ...any) {
	var style lipgloss.Style

	switch level {
	case LevelDebug:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	case LevelInfo:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	case LevelWarn:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB000"))
	case LevelError:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))
	}

	timestamp := time.Now().Format("15:04:05")
	levelStr := fmt.Sprintf("[%s]", level.String())

	output := fmt.Sprintf("%s %s %s", timestamp, levelStr, msg)

	// Add structured fields if any
	if len(args) > 0 {
		fields := make([]string, 0, len(args)/2)
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				fields = append(fields, fmt.Sprintf("%s=%v", args[i], args[i+1]))
			}
		}
		if len(fields) > 0 {
			output += fmt.Sprintf(" %v", fields)
		}
	}

	fmt.Fprintln(os.Stderr, style.Render(output))
}

// slogLevelFromLogLevel converts our LogLevel to slog.Level
func slogLevelFromLogLevel(level LogLevel) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(config *LoggerConfig) error {
	logger, err := NewLogger(config)
	if err != nil {
		return fmt.Errorf("failed to initialize global logger: %w", err)
	}
	globalLogger = logger
	return nil
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// Initialize with default config if not set
		globalLogger, _ = NewLogger(DefaultConfig())
	}
	return globalLogger
}

// Convenience functions for global logger
func Debug(msg string, args ...any) {
	GetGlobalLogger().Debug(msg, args...)
}

func Info(msg string, args ...any) {
	GetGlobalLogger().Info(msg, args...)
}

func Warn(msg string, args ...any) {
	GetGlobalLogger().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	GetGlobalLogger().Error(msg, args...)
}

// WithField adds a field to the global logger
func WithField(key string, value any) *Logger {
	return GetGlobalLogger().WithField(key, value)
}

// WithFields adds multiple fields to the global logger
func WithFields(fields map[string]any) *Logger {
	return GetGlobalLogger().WithFields(fields)
}
