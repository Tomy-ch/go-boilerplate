// Package logging は、アプリケーションのロギング機能を提供します。
package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// stacktraceKey は stacktrace の出力キー名。EncoderConfig.StacktraceKey と
// JSON array 化ラッパ（wrapStacktraceCore）の双方が参照する単一の出所。
const stacktraceKey = "stacktrace"

// stdoutSyncer / stderrSyncer は標準出力 / 標準エラーへの WriteSyncer です。
// logging は os を直接 import できない（depguard）制約があるため zap.Open で解決します。
var (
	stdoutSyncer = mustOpenSink("stdout")
	stderrSyncer = mustOpenSink("stderr")
)

// mustOpenSink は path の WriteSyncer を返します。開けない場合は panic します。
func mustOpenSink(path string) zapcore.WriteSyncer {
	ws, _, err := zap.Open(path)
	if err != nil {
		panic("logging: failed to open sink " + path + ": " + err.Error())
	}
	return ws
}

// encoderConfig は JSON / console 共通のエンコーダ設定を返します。
// encodeLevel は呼び出し側が指定します。
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

// NewJSONLogger は JSON 構造化ロガーを生成します。
// level は出力レベル、stacktraceLevel は stacktrace を付与し始めるレベルです。
func NewJSONLogger(level, stacktraceLevel Level) Logger {
	enc := zapcore.NewJSONEncoder(encoderConfig(zapcore.LowercaseLevelEncoder))
	return buildLogger(enc, stdoutSyncer, level.zl, stacktraceLevel.zl, true)
}

// NewConsoleLogger は人間可読な console ロガーを生成します。
// level は出力レベル、stacktraceLevel は stacktrace を付与し始めるレベルです。
func NewConsoleLogger(level, stacktraceLevel Level) Logger {
	enc := zapcore.NewConsoleEncoder(encoderConfig(zapcore.CapitalColorLevelEncoder))
	return buildLogger(enc, stdoutSyncer, level.zl, stacktraceLevel.zl, false)
}

// buildLogger は encoder と出力先から Logger を構築します。
// jsonArrayStacktrace=true のとき、stacktrace を行配列として出力します。
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
		zap.ErrorOutput(stderrSyncer),
	)
	return &logger{log: zl}
}
