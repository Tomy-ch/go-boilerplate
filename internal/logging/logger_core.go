// Package logging は、アプリケーションのロギング機能を提供します。
package logging

import (
	"fmt"
	"os"

	"go-boilerplate/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// stacktraceKey は stacktrace の出力キー名。EncoderConfig.StacktraceKey と
// JSON array 化ラッパ（wrapStacktraceCore）の双方が参照する単一の出所。
const stacktraceKey = "stacktrace"

// encoderConfig は JSON / console 共通のエンコーダ設定を返します。
//
// キー名は JSON 取り込み適性を優先した名称に統一します。console エンコーダは
// 組み込みフィールド（time/level/name/caller/msg/stack）を位置出力しキー文字列を
// 印字しない（空文字なら省略するだけ）ため、console でもこの統一で齟齬は出ません。
//
// encodeLevel のみ出力方式依存（JSON=lowercase / console=color）で呼び出し側が差し込みます。
// EncodeDuration は意図的に未設定です。zap.Duration フィールドを一切生成しない
// （DurationMs は float64 へ変換済み）ため、未設定で副作用がありません。
func encoderConfig(encodeLevel zapcore.LevelEncoder) zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:       "ts",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: stacktraceKey,
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeLevel:   encodeLevel,
		EncodeCaller:  zapcore.ShortCallerEncoder,
		EncodeName:    zapcore.FullNameEncoder,
	}
}

// NewJSONLogger は JSON 構造化ロガーを生成します（機械可読・本番向け出力方式）。
// level は出力レベル、stacktraceLevel は stacktrace を付与し始めるレベルです。
func NewJSONLogger(level, stacktraceLevel zapcore.Level) Logger {
	enc := zapcore.NewJSONEncoder(encoderConfig(zapcore.LowercaseLevelEncoder))
	return buildLogger(enc, zapcore.Lock(os.Stdout), level, stacktraceLevel, true)
}

// NewConsoleLogger は人間可読な console ロガーを生成します（開発向け出力方式）。
// level は出力レベル、stacktraceLevel は stacktrace を付与し始めるレベルです。
func NewConsoleLogger(level, stacktraceLevel zapcore.Level) Logger {
	enc := zapcore.NewConsoleEncoder(encoderConfig(zapcore.CapitalColorLevelEncoder))
	return buildLogger(enc, zapcore.Lock(os.Stdout), level, stacktraceLevel, false)
}

// buildLogger は encoder と出力先から Logger を直接構築する共通処理です。
//
// jsonArrayStacktrace=true のとき、zap が単一文字列で付与する stacktrace を
// 行配列へ変換する core でラップします（JSON 出力時のみ。console は zap が
// 改行付きで独自整形するため不要）。
// ws は出力先（本番系は stdout、テストはバッファ）です。zap 自身の内部エラーは
// stderr へ出します（ErrorOutput はレベルルーティングではなく zap の内部エラー出力先）。
func buildLogger(
	enc zapcore.Encoder, ws zapcore.WriteSyncer, level, stacktraceLevel zapcore.Level, jsonArrayStacktrace bool,
) Logger {
	core := zapcore.NewCore(enc, ws, zap.NewAtomicLevelAt(level))
	if jsonArrayStacktrace {
		core = wrapStacktraceCore(core, stacktraceKey)
	}
	zl := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(stacktraceLevel),
		zap.ErrorOutput(zapcore.Lock(os.Stderr)),
	)
	return &logger{log: zl}
}

// New は appCfg の Mode に応じた Logger を返します。
//
// NOTE: 環境（Mode）→出力方式/レベルの選択は暫定でここに置いています。
// 後続で di（provideLogger）へ移し、本関数と config 依存は撤去する予定です。
func New(appCfg *config.ApplicationConfig) (Logger, error) {
	switch {
	case appCfg.IsProductionMode():
		return NewJSONLogger(zapcore.InfoLevel, zapcore.ErrorLevel), nil
	case appCfg.IsDevelopmentMode():
		return NewConsoleLogger(zapcore.DebugLevel, zapcore.WarnLevel), nil
	default:
		return nil, fmt.Errorf("unknown app mode: %s", appCfg.Mode())
	}
}
