// Package logging は、アプリケーションのロギング機能を提供します。
package logging

import (
	"fmt"

	"boilerplate-go/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewProductionLogger は、本番環境用のZapロガーを生成します。
func NewProductionLogger() (Logger, error) {
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

	core, err := cfg.Build(
		zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return &logger{log: core}, err
}

// NewDevelopmentLogger は、開発環境用のZapロガーを生成します。
func NewDevelopmentLogger() (Logger, error) {
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
	core, err := cfg.Build(
		zap.AddCaller(), zap.AddStacktrace(zapcore.WarnLevel),
	)

	return &logger{log: core}, err
}

// New は、指定された設定に基づいて新しいZapロガーを生成します。
func New(appCfg *config.ApplicationConfig) (Logger, error) {
	if appCfg.IsProductionMode() {
		return NewProductionLogger()
	}
	if appCfg.IsDevelopmentMode() {
		return NewDevelopmentLogger()
	}
	return nil, fmt.Errorf("unknown app mode: %s", appCfg.Mode())
}
