package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type logrusLogEntry struct {
	entry *logrus.Entry
}

type logrusLogger struct {
	logger *logrus.Logger
}

func getFormatter(isJSON bool) logrus.Formatter {
	if isJSON {
		return &logrus.JSONFormatter{}
	}
	return &logrus.TextFormatter{
		FullTimestamp:          true,
		DisableLevelTruncation: true,
		ForceColors:            true,
	}
}

func newLogrusLogger(config Configuration) (*logrusLogger, error) {
	logLevel := config.ConsoleLevel
	if logLevel == "" {
		logLevel = config.FileLevel
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return nil, err
	}

	stdOutHandler := os.Stdout
	fileHandler := &lumberjack.Logger{
		Filename: config.FileLocation,
		MaxSize:  100,
		Compress: true,
		MaxAge:   28,
	}
	lLogger := &logrus.Logger{
		Out:       stdOutHandler,
		Formatter: getFormatter(config.EnableJSONFormat),
		Hooks:     make(logrus.LevelHooks),
		Level:     level,
	}

	if config.EnableConsole && config.EnableFile {
		lLogger.SetOutput(io.MultiWriter(stdOutHandler, fileHandler))
	} else if config.EnableFile {
		lLogger.SetOutput(fileHandler)
		lLogger.SetFormatter(getFormatter(config.FileJSONFormat))
	}

	return &logrusLogger{
		logger: lLogger,
	}, nil
}

func (l *logrusLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *logrusLogger) Debug(msg string) {
	l.logger.Debug(msg)
}

func (l *logrusLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *logrusLogger) Info(msg string) {
	l.logger.Info(msg)
}

func (l *logrusLogger) Infoln(msg string) {
	l.logger.Infoln(msg)
}

func (l *logrusLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *logrusLogger) Warn(msg string) {
	l.logger.Warn(msg)
}

func (l *logrusLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *logrusLogger) Error(msg string) {
	l.logger.Error(msg)
}

func (l *logrusLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *logrusLogger) Fatal(msg string) {
	l.logger.Fatal(msg)
}

func (l *logrusLogger) Panicf(format string, args ...interface{}) {
	l.logger.Panicf(format, args...)
}

func (l *logrusLogger) Panic(msg string) {
	l.logger.Panic(msg)
}

func (l *logrusLogger) WithFields(fields Fields) Logger {
	return &logrusLogEntry{
		entry: l.logger.WithFields(convertToLogrusFields(fields)),
	}
}

func (l *logrusLogger) GetDelegate() interface{} {
	return l.logger
}

func (l *logrusLogger) SetLogLevel(logLevel string) error {
	logrusLogLevel, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	l.logger.SetLevel(logrusLogLevel)
	return nil
}

func (l *logrusLogEntry) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

func (l *logrusLogEntry) Debug(msg string) {
	l.entry.Debug(msg)
}

func (l *logrusLogEntry) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

func (l *logrusLogEntry) Info(msg string) {
	l.entry.Info(msg)
}

func (l *logrusLogEntry) Infoln(msg string) {
	l.entry.Infoln(msg)
}

func (l *logrusLogEntry) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

func (l *logrusLogEntry) Warn(msg string) {
	l.entry.Warn(msg)
}

func (l *logrusLogEntry) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

func (l *logrusLogEntry) Error(msg string) {
	l.entry.Error(msg)
}

func (l *logrusLogEntry) Fatalf(format string, args ...interface{}) {
	l.entry.Fatalf(format, args...)
}

func (l *logrusLogEntry) Fatal(msg string) {
	l.entry.Fatal(msg)
}

func (l *logrusLogEntry) WithFields(fields Fields) Logger {
	return &logrusLogEntry{
		entry: l.entry.WithFields(convertToLogrusFields(fields)),
	}
}

func (l *logrusLogEntry) SetLogLevel(logLevel string) error {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	l.entry.Logger.SetLevel(level)
	return nil
}

func (l *logrusLogEntry) GetDelegate() interface{} {
	return l.entry
}

func convertToLogrusFields(fields Fields) logrus.Fields {
	logrusFields := logrus.Fields{}
	for index, val := range fields {
		logrusFields[index] = val
	}
	return logrusFields
}
