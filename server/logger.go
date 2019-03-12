package server

import (
	"go.uber.org/zap"
)

type Logger struct {
	logger *zap.Logger
	sugar *zap.SugaredLogger
}

func NewLogger(config *Config) (*Logger) {

	var logger *zap.Logger

	if config.DevelopmentEnabled {
		logger, _ = zap.NewDevelopment(zap.AddCallerSkip(1))
	}else{
		logger, _ = zap.NewProduction(zap.AddCallerSkip(1))
	}

	sugar := logger.Sugar()

	return &Logger{
		logger: logger,
		sugar: sugar,
	}

}

func (l Logger) Sync() {
	l.logger.Sync()
}

func (l Logger) Debug(args ...interface{}) {
	l.sugar.Debug(args...)
}

func (l Logger) Info(args ...interface{}) {
	l.sugar.Info(args...)
}

func (l Logger) Warn(args ...interface{}) {
	l.sugar.Warn(args...)
}

func (l Logger) Error(args ...interface{}) {
	l.sugar.Error(args...)
}

func (l Logger) DPanic(args ...interface{}) {
	l.sugar.DPanic(args...)
}

func (l Logger) Panic(args ...interface{}) {
	l.sugar.Panic(args...)
}

func (l Logger) Fatal(args ...interface{}) {
	l.sugar.Fatal(args...)
}

func (l Logger) Debugf(template string, args ...interface{}) {
	l.sugar.Debugf(template, args...)
}

func (l Logger) Infof(template string, args ...interface{}) {
	l.sugar.Infof(template, args...)
}

func (l Logger) Warnf(template string, args ...interface{}) {
	l.sugar.Warnf(template, args...)
}

func (l Logger) Errorf(template string, args ...interface{}) {
	l.sugar.Errorf(template, args...)
}

func (l Logger) DPanicf(template string, args ...interface{}) {
	l.sugar.DPanicf(template, args...)
}

func (l Logger) Panicf(template string, args ...interface{}) {
	l.Panicf(template, args...)
}

func (l Logger) Fatalf(template string, args ...interface{}) {
	l.sugar.Fatalf(template, args...)
}

func (l Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}

func (l Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

func (l Logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}

func (l Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

func (l Logger) DPanicw(msg string, keysAndValues ...interface{}) {
	l.sugar.DPanicw(msg, keysAndValues...)
}

func (l Logger) Panicw(msg string, keysAndValues ...interface{}) {
	l.sugar.Panicw(msg, keysAndValues...)
}

func (l Logger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.sugar.Fatalw(msg, keysAndValues...)
}