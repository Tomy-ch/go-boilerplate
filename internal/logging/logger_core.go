// Package logging は、アプリケーションのロギング機能を提供します。
package logging

import (
	"fmt"

	"go-boilerplate/internal/config"

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

	return buildLogger(cfg, zapcore.ErrorLevel)
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

	return buildLogger(cfg, zapcore.WarnLevel)
}

// buildLogger は、zap.Config から Logger を構築する共通処理です。
// Build 失敗時は zap が nil の *zap.Logger を返すため、それを包んだ
// Logger（中身 nil）を返すと初回ログ出力で nil 参照 panic になる。
// よってエラー時は Logger を返さず、nil とエラーのみを返す。
//
// stacktrace は JSON 出力時のみ行配列に変換する（wrapStacktraceCore）。
// console エンコーダは Entry.Stack を独自に整形（改行+インデント）するため、
// wrap すると console 出力が一行 JSON ダンプ化して可読性が落ちるので適用しない。
// StacktraceKey 未設定の Config（テスト用最小構成など）でも wrap をスキップする。
func buildLogger(cfg zap.Config, stacktraceLevel zapcore.Level) (Logger, error) {
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(stacktraceLevel),
	}
	if stacktraceKey := cfg.EncoderConfig.StacktraceKey; cfg.Encoding == "json" && stacktraceKey != "" {
		opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return wrapStacktraceCore(core, stacktraceKey)
		}))
	}
	zl, err := cfg.Build(opts...)
	if err != nil {
		return nil, err
	}

	return &logger{log: zl}, nil
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
