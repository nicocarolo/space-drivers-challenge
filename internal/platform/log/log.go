package log

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {

	// Error logs a message at ErrorLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Error(msg string, fields ...Field)

	// Info logs a message at InfoLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Info(msg string, fields ...Field)
}

type logger struct {
	*zap.Logger
}

var DefaultLogger Logger

func Error(ctx context.Context, msg string, fields ...Field) {
	getLogger(ctx).Error(msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...Field) {
	getLogger(ctx).Info(msg, fields...)
}

type logCtxKey struct{}

func getLogger(ctx context.Context) Logger {
	l, ok := ctx.Value(logCtxKey{}).(Logger)
	if ok {
		return l
	}

	if DefaultLogger == nil {
		l, err := getZapConfig().Build()
		if err == nil {
			DefaultLogger = &logger{
				Logger: l,
			}
		} else {
			DefaultLogger = &logger{
				Logger: zap.NewNop(),
			}
		}
	}
	return DefaultLogger
}

func getZapConfig() zap.Config {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zap.Config{
		Encoding:          "json",
		EncoderConfig:     encoderConfig,
		Level:             zap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths:       []string{"stdout"},
		DisableStacktrace: true,
		DisableCaller:     true,
	}
}
