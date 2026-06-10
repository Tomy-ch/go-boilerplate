package logging

import (
	"testing"

	"go-boilerplate/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

// NewTestLogger は、テスト用のLoggerインスタンスを生成します。
func NewTestLogger(t *testing.T) Logger {
	t.Helper()
	return &logger{log: zaptest.NewLogger(t)}
}

// NewObservedTestLogger は、出力を捕捉できるテスト用 Logger と観測ログを返します。
// ログレベル・出力有無を検証したいテストで使用します。
func NewObservedTestLogger(t *testing.T) (Logger, *observer.ObservedLogs) {
	t.Helper()
	core, observed := observer.New(zapcore.DebugLevel)
	return &logger{log: zap.New(core)}, observed
}

// NewObservedTestLoggerWithCaller は、caller 情報付きで出力を捕捉できるテスト用 Logger と観測ログを返します。
// caller（発生源）を検証したいテストで使用します。
func NewObservedTestLoggerWithCaller(t *testing.T) (Logger, *observer.ObservedLogs) {
	t.Helper()
	core, observed := observer.New(zapcore.DebugLevel)
	return &logger{log: zap.New(core, zap.AddCaller())}, observed
}

// NewTestLogFieldBuilder は、テスト用のLogFieldBuilderインスタンスを生成します。
func NewTestLogFieldBuilder(t *testing.T) LogFieldBuilder {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	osCfg := config.NewOperatingSystemConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	return NewLogFields(obsCfg, osCfg)
}
