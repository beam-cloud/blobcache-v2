package blobcache

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logger struct {
	zapLogger *zap.Logger
	debug     bool
}

var (
	Logger *logger
	once   sync.Once
)

func InitLogger(debugMode bool) {
	once.Do(func() {
		var cfg zap.Config
		if debugMode {
			cfg = zap.NewDevelopmentConfig()
		} else {
			cfg = zap.NewProductionConfig()
		}

		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

		var err error
		zapLogger, err := cfg.Build()
		if err != nil {
			panic(err)
		}

		Logger = &logger{
			zapLogger: zapLogger,
			debug:     debugMode,
		}

	})
}

func GetLogger() *logger {
	if Logger == nil {
		panic("Logger is not initialized. Call InitLogger first.")
	}
	return Logger
}

func (l *logger) Debug(msg string, fields ...zap.Field) {
	if l.debug {
		l.zapLogger.Debug(msg, fields...)
	}
}

func (l *logger) Debugf(template string, args ...interface{}) {
	if l.debug {
		l.zapLogger.Debug(fmt.Sprintf(template, args...))
	}
}

func (l *logger) Info(msg string, fields ...zap.Field) {
	l.zapLogger.Info(msg, fields...)
}

func (l *logger) Infof(template string, args ...interface{}) {
	l.zapLogger.Info(fmt.Sprintf(template, args...))
}

func (l *logger) Warn(msg string, fields ...zap.Field) {
	l.zapLogger.Warn(msg, fields...)
}

func (l *logger) Warnf(template string, args ...interface{}) {
	l.zapLogger.Warn(fmt.Sprintf(template, args...))
}

func (l *logger) Error(msg string, fields ...zap.Field) {
	l.zapLogger.Error(msg, fields...)
}

func (l *logger) Errorf(template string, args ...interface{}) {
	l.zapLogger.Error(fmt.Sprintf(template, args...))
}

func (l *logger) Fatal(msg string, fields ...zap.Field) {
	l.zapLogger.Fatal(msg, fields...)
}

func (l *logger) Fatalf(template string, args ...interface{}) {
	l.zapLogger.Fatal(fmt.Sprintf(template, args...))
}

func (l *logger) Sync() {
	_ = l.zapLogger.Sync()
}
