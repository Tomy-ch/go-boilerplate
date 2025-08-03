package logging

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewProductionLogger は、本番環境用のZapロガーを生成します。
func NewProductionLogger() *zap.Logger {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Development:      false,
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:       "ts",
			LevelKey:      "level",
			NameKey:       "logger",
			CallerKey:     "caller",
			MessageKey:    "msg",
			StacktraceKey: "stacktrace",
			EncodeTime:    zapcore.ISO8601TimeEncoder,
			EncodeLevel:   zapcore.LowercaseLevelEncoder,
			EncodeCaller:  zapcore.ShortCallerEncoder,
		},
	}

	logger, err := cfg.Build(
		zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		panic(fmt.Sprintf(
			"failed to create production logger instance: %v", err,
		))
	}
	return logger
}

// NewDevelopmentLogger は、開発環境用のZapロガーを生成します。
func NewDevelopmentLogger() *zap.Logger {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      true,
		Encoding:         "console",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:       "Time",
			LevelKey:      "Level",
			NameKey:       "Name",
			CallerKey:     "Call",
			MessageKey:    "Msg",
			StacktraceKey: "Stack",
			EncodeTime:    zapcore.ISO8601TimeEncoder,
			EncodeLevel:   zapcore.CapitalColorLevelEncoder,
			EncodeCaller:  zapcore.ShortCallerEncoder,
		},
	}

	logger, err := cfg.Build(
		zap.AddCaller(), zap.AddStacktrace(zapcore.WarnLevel),
	)
	if err != nil {
		panic(fmt.Sprintf(
			"failed to create development logger instance: %v", err,
		))
	}
	return logger
}
