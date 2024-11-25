package logger

import (
	"context"

	"github.com/KyberNetwork/kutils/klog"
)

// Fields Type to pass when we want to call WithFields for structured logging
type Fields = klog.Fields

// LoggerBackend represents the int enum for backend of logger.
type LoggerBackend = klog.LoggerBackend

const (
	// LoggerBackendZap logging using Uber's zap backend
	LoggerBackendZap LoggerBackend = klog.LoggerBackendZap
	// LoggerBackendLogrus logging using logrus backend
	LoggerBackendLogrus = klog.LoggerBackendLogrus
)

// Logger is our contract for the logger
type Logger = klog.Logger

// Configuration stores the config for the logger
// For some loggers there can only be one level across writers, for such the level of Console is picked by default
type Configuration = klog.Configuration

// DefaultLogger creates default logger, which uses zap sugarLogger and outputs to console
func DefaultLogger() Logger {
	logger, _ := NewLogger(Configuration{
		EnableConsole: true,
		ConsoleLevel:  "warn",
	}, LoggerBackendZap)
	return logger
}

// InitLogger returns an instance of logger
func InitLogger(config Configuration, backend LoggerBackend) (Logger, error) {
	return klog.InitLogger(config, backend)
}

func NewLogger(config Configuration, backend LoggerBackend) (Logger, error) {
	return klog.NewLogger(config, backend)
}

func Debug(ctx context.Context, msg string) {
	klog.Debugf(ctx, msg)
}

func Debugf(ctx context.Context, format string, args ...interface{}) {
	klog.Debugf(ctx, format, args...)
}

func Info(ctx context.Context, msg string) {
	klog.Infof(ctx, msg)
}

func Infof(ctx context.Context, format string, args ...interface{}) {
	klog.Infof(ctx, format, args...)
}

func Infoln(ctx context.Context, msg string) {
	klog.Infoln(ctx, msg)
}

func Warn(ctx context.Context, msg string) {
	klog.Warnf(ctx, msg)
}

func Warnf(ctx context.Context, format string, args ...interface{}) {
	klog.Warnf(ctx, format, args...)
}

func Error(ctx context.Context, msg string) {
	klog.Errorf(ctx, msg)
}

func Errorf(ctx context.Context, format string, args ...interface{}) {
	klog.Errorf(ctx, format, args...)
}

func Fatal(ctx context.Context, msg string) {
	klog.Fatalf(ctx, msg)
}

func Fatalf(ctx context.Context, format string, args ...interface{}) {
	klog.Fatalf(ctx, format, args...)
}

func WithFields(ctx context.Context, keyValues Fields) Logger {
	return klog.WithFields(ctx, keyValues)
}

func WithFieldsNonContext(keyValues Fields) Logger {
	return klog.WithFields(context.Background(), keyValues)
}

func GetDelegate() interface{} {
	return klog.GetDelegate(context.Background())
}

func SetLogLevel(level string) error {
	return klog.SetLogLevel(context.Background(), level)
}
