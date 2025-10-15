package sietch

import (
	"context"
	"fmt"
	"time"
)

// QueryLogger defines the interface for logging repository operations
type QueryLogger interface {
	// LogQuery logs a query execution with timing and error information
	LogQuery(ctx context.Context, operation string, query string, args []any, duration time.Duration, err error)

	// LogOperation logs a high-level repository operation
	LogOperation(ctx context.Context, operation string, entityType string, duration time.Duration, err error)
}

// LogLevel defines the severity of a log entry
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Level       LogLevel
	Timestamp   time.Time
	Operation   string
	Query       string
	Args        []any
	Duration    time.Duration
	Error       error
	ContextData map[string]any
}

// ConsoleLogger is a simple logger that writes to stdout
type ConsoleLogger struct {
	MinLevel LogLevel
}

// NewConsoleLogger creates a new console logger
func NewConsoleLogger(minLevel LogLevel) *ConsoleLogger {
	return &ConsoleLogger{
		MinLevel: minLevel,
	}
}

// LogQuery implements QueryLogger
func (l *ConsoleLogger) LogQuery(ctx context.Context, operation string, query string, args []any, duration time.Duration, err error) {
	level := LogLevelInfo
	if err != nil {
		level = LogLevelError
	}

	if l.shouldLog(level) {
		fmt.Printf("[%s] %s - Query: %s | Duration: %v | Args: %v | Error: %v\n",
			level, operation, query, duration, args, err)
	}
}

// LogOperation implements QueryLogger
func (l *ConsoleLogger) LogOperation(ctx context.Context, operation string, entityType string, duration time.Duration, err error) {
	level := LogLevelInfo
	if err != nil {
		level = LogLevelError
	}

	if l.shouldLog(level) {
		fmt.Printf("[%s] %s on %s | Duration: %v | Error: %v\n",
			level, operation, entityType, duration, err)
	}
}

func (l *ConsoleLogger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
	}
	return levels[level] >= levels[l.MinLevel]
}

// NoOpLogger is a logger that does nothing (useful for disabling logging)
type NoOpLogger struct{}

// NewNoOpLogger creates a no-op logger
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

// LogQuery implements QueryLogger
func (l *NoOpLogger) LogQuery(ctx context.Context, operation string, query string, args []any, duration time.Duration, err error) {
	// No-op
}

// LogOperation implements QueryLogger
func (l *NoOpLogger) LogOperation(ctx context.Context, operation string, entityType string, duration time.Duration, err error) {
	// No-op
}

// LoggableRepository is an optional interface for repositories that support logging
type LoggableRepository interface {
	// SetLogger sets the query logger for this repository
	SetLogger(logger QueryLogger)

	// GetLogger returns the current query logger
	GetLogger() QueryLogger
}

// measureDuration is a helper function to measure operation duration
func measureDuration(start time.Time) time.Duration {
	return time.Since(start)
}

// logOperation is a helper to log an operation with timing
func logOperation(logger QueryLogger, ctx context.Context, operation string, entityType string, start time.Time, err error) {
	if logger != nil {
		duration := measureDuration(start)
		logger.LogOperation(ctx, operation, entityType, duration, err)
	}
}

// logQuery is a helper to log a query with timing
func logQuery(logger QueryLogger, ctx context.Context, operation string, query string, args []any, start time.Time, err error) {
	if logger != nil {
		duration := measureDuration(start)
		logger.LogQuery(ctx, operation, query, args, duration, err)
	}
}
