package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type zapLogger struct {
	sugaredLogger *zap.SugaredLogger
	atomicLevel   *zap.AtomicLevel
}

func getEncoder(isJSON bool) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	if isJSON {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getZapLevel(level string) zapcore.Level {
	switch level {
	case infoLvl:
		return zapcore.InfoLevel
	case warnLvl:
		return zapcore.WarnLevel
	case debugLvl:
		return zapcore.DebugLevel
	case errorLvl:
		return zapcore.ErrorLevel
	case fatalLvl:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func newZapLogger(config Configuration) (*zapLogger, error) {
	cores := []zapcore.Core{}
	atom := zap.NewAtomicLevel()
	if config.EnableConsole {
		level := getZapLevel(config.ConsoleLevel)
		atom.SetLevel(level)
		writer := zapcore.Lock(os.Stdout)
		core := zapcore.NewCore(getEncoder(config.EnableJSONFormat), writer, atom)
		cores = append(cores, core)
	}

	if config.EnableFile {
		level := getZapLevel(config.FileLevel)
		atom.SetLevel(level)
		writer := zapcore.AddSync(&lumberjack.Logger{
			Filename: config.FileLocation,
			MaxSize:  100,
			Compress: true,
			MaxAge:   28,
		})
		core := zapcore.NewCore(getEncoder(config.FileJSONFormat), writer, atom)
		cores = append(cores, core)
	}

	combinedCore := zapcore.NewTee(cores...)

	// AddCallerSkip skips 2 number of callers, this is important else the file that gets
	// logged will always be the wrapped file. In our case zap.go
	logger := zap.New(combinedCore,
		zap.AddCallerSkip(2),
		zap.AddCaller(),
	).Sugar()
	return &zapLogger{
		sugaredLogger: logger,
		atomicLevel:   &atom,
	}, nil
}

func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.sugaredLogger.Debugf(format, args...)
}

func (l *zapLogger) Debug(msg string) {
	l.sugaredLogger.Debug(msg)
}

func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.sugaredLogger.Infof(format, args...)
}

func (l *zapLogger) Info(msg string) {
	l.sugaredLogger.Info(msg)
}

func (l *zapLogger) Infoln(msg string) {
	l.sugaredLogger.Infoln(msg)
}

func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.sugaredLogger.Warnf(format, args...)
}

func (l *zapLogger) Warn(msg string) {
	l.sugaredLogger.Warn(msg)
}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.sugaredLogger.Errorf(format, args...)
}

func (l *zapLogger) Error(msg string) {
	l.sugaredLogger.Error(msg)
}

func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.sugaredLogger.Fatalf(format, args...)
}

func (l *zapLogger) Fatal(msg string) {
	l.sugaredLogger.Fatal(msg)
}

func (l *zapLogger) WithFields(fields Fields) Logger {
	var fds = make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		fds = append(fds, k)
		fds = append(fds, v)
	}
	newLogger := l.sugaredLogger.With(fds...)
	return &zapLogger{newLogger, l.atomicLevel}
}

func (l *zapLogger) GetDelegate() interface{} {
	return l.sugaredLogger
}

func (l *zapLogger) SetLogLevel(level string) error {
	l.atomicLevel.SetLevel(getZapLevel(level))
	return nil
}

func GetDesugaredZapLoggerDelegate(instance Logger) (*zap.Logger, error) {
	switch v := instance.GetDelegate().(type) {
	case *zap.SugaredLogger:
		return instance.GetDelegate().(*zap.SugaredLogger).Desugar(), nil
	default:
		return nil, fmt.Errorf("expected zap.SugaredLogger but got: %v", v)
	}
}
