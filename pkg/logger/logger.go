package logger

import (
	"context"
	"errors"
	"sync"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

// A global variable so that log functions can be directly accessed
var log = DefaultLogger()

// Fields Type to pass when we want to call WithFields for structured logging
type Fields map[string]interface{}

// LoggerBackend represents the int enum for backend of logger.
// nolint:revive
type LoggerBackend int

const (
	// Debug has verbose message
	debugLvl = "debug"
	// Info is default log level
	infoLvl = "info"
	// Warn is for logging messages about possible issues
	warnLvl = "warn"
	// Error is for logging errors
	errorLvl = "error"
	// Fatal is for logging fatal messages. The system shutdowns after logging the message.
	fatalLvl = "fatal"
)

const (
	// LoggerBackendZap logging using Uber's zap backend
	LoggerBackendZap LoggerBackend = iota
	// LoggerBackendLogrus logging using logrus backend
	LoggerBackendLogrus
)

var (
	errInvalidLoggerInstance = errors.New("invalid logger instance")

	once sync.Once
)

// Logger is our contract for the logger
type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})

	Info(msg string)
	Infof(format string, args ...interface{})
	Infoln(msg string)

	Warn(msg string)
	Warnf(format string, args ...interface{})

	Error(msg string)
	Errorf(format string, args ...interface{})

	Fatal(msg string)
	Fatalf(format string, args ...interface{})

	WithFields(keyValues Fields) Logger

	// extract instance logger.
	GetDelegate() interface{}
	SetLogLevel(level string) error
}

// Configuration stores the config for the logger
// For some loggers there can only be one level across writers, for such the level of Console is picked by default
type Configuration struct {
	EnableConsole    bool   `mapstructure:"enableConsole"`
	EnableJSONFormat bool   `mapstructure:"enableJSONFormat"`
	ConsoleLevel     string `mapstructure:"consoleLevel"`
	EnableFile       bool
	FileJSONFormat   bool
	FileLevel        string
	FileLocation     string
}

// DefaultLogger creates default logger, which uses zap sugarLogger and outputs to console
func DefaultLogger() Logger {
	cfg := Configuration{
		EnableConsole:    true,
		EnableJSONFormat: false,
		ConsoleLevel:     "warn",
		EnableFile:       false,
		FileJSONFormat:   false,
	}
	logger, _ := newZapLogger(cfg)
	return logger
}

// InitLogger returns an instance of logger
func InitLogger(config Configuration, backend LoggerBackend) (Logger, error) {
	var err error
	once.Do(func() {
		log, err = NewLogger(config, backend)
	})
	return log, err
}

func NewLogger(config Configuration, backend LoggerBackend) (Logger, error) {
	switch backend {
	case LoggerBackendZap:
		return newZapLogger(config)

	case LoggerBackendLogrus:
		return newLogrusLogger(config)

	default:
		return nil, errInvalidLoggerInstance
	}
}

func Debug(ctx context.Context, msg string) {
	fromCtx(ctx).Debugf(msg)
}

func Debugf(ctx context.Context, format string, args ...interface{}) {
	fromCtx(ctx).Debugf(format, args...)
}

func Info(ctx context.Context, msg string) {
	fromCtx(ctx).Infof(msg)
}

func Infof(ctx context.Context, format string, args ...interface{}) {
	fromCtx(ctx).Infof(format, args...)
}

func Infoln(ctx context.Context, msg string) {
	fromCtx(ctx).Infoln(msg)
}

func Warn(ctx context.Context, msg string) {
	fromCtx(ctx).Warnf(msg)
}

func Warnf(ctx context.Context, format string, args ...interface{}) {
	fromCtx(ctx).Warnf(format, args...)
}

func Error(ctx context.Context, msg string) {
	fromCtx(ctx).Errorf(msg)
}

func Errorf(ctx context.Context, format string, args ...interface{}) {
	fromCtx(ctx).Errorf(format, args...)
}

func Fatal(ctx context.Context, msg string) {
	fromCtx(ctx).Fatalf(msg)
}

func Fatalf(ctx context.Context, format string, args ...interface{}) {
	fromCtx(ctx).Fatalf(format, args...)
}

func WithFields(ctx context.Context, keyValues Fields) Logger {
	return fromCtx(ctx).WithFields(keyValues)
}

func WithFieldsNonContext(keyValues Fields) Logger {
	return log.WithFields(keyValues)
}

func GetDelegate() interface{} {
	return log.GetDelegate()
}

func SetLogLevel(level string) error {
	return log.SetLogLevel(level)
}

func fromCtx(ctx context.Context) Logger {
	l := ctx.Value(constant.CtxLoggerKey)
	if ctxLogger, ok := l.(Logger); ok && ctxLogger != nil {
		return ctxLogger
	}
	return log
}
