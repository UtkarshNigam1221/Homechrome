// Package logger provides structured logging
package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type contextKey string

const (
	loggerKey    contextKey = "logger"
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
)

// Logger wraps zerolog.Logger
type Logger struct {
	zl zerolog.Logger
}

// New creates a new logger
func New(debug bool) *Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	var zl zerolog.Logger
	if debug {
		output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		zl = zerolog.New(output).With().Timestamp().Caller().Logger()
	} else {
		zl = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	return &Logger{zl: zl}
}

// WithContext returns a logger with context values
func (l *Logger) WithContext(ctx context.Context) *Logger {
	newLogger := l.zl.With().Logger()

	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		newLogger = newLogger.With().Str("request_id", requestID).Logger()
	}

	if userID, ok := ctx.Value(userIDKey).(string); ok {
		newLogger = newLogger.With().Str("user_id", userID).Logger()
	}

	return &Logger{zl: newLogger}
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{zl: l.zl.With().Interface(key, value).Logger()}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.zl.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{zl: ctx.Logger()}
}

// WithError adds an error to the logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{zl: l.zl.With().Err(err).Logger()}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	l.zl.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.zl.Debug().Msgf(format, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	l.zl.Info().Msg(msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.zl.Info().Msgf(format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	l.zl.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.zl.Warn().Msgf(format, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.zl.Error().Msg(msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.zl.Error().Msgf(format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string) {
	l.zl.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.zl.Fatal().Msgf(format, args...)
}

// ToContext adds the logger to the context
func ToContext(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext retrieves the logger from the context
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(loggerKey).(*Logger); ok {
		return l
	}
	return New(false)
}

// SetRequestID sets the request ID in the context
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID gets the request ID from the context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// SetUserID sets the user ID in the context
func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID gets the user ID from the context
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}

// NewNoop creates a noop logger for testing
func NewNoop() *Logger {
	return &Logger{zl: zerolog.Nop()}
}
