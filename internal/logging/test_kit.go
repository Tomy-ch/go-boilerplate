package logging

import (
	"testing"

	"go-boilerplate/internal/config"

	"go.uber.org/zap/zaptest"
)

// NewTestLogger は、テスト用のLoggerインスタンスを生成します。
func NewTestLogger(t *testing.T) Logger {
	t.Helper()
	return &logger{log: zaptest.NewLogger(t)}
}

// NewTestLogFieldBuilder は、テスト用のLogFieldBuilderインスタンスを生成します。
func NewTestLogFieldBuilder(t *testing.T) LogFieldBuilder {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	osCfg := config.NewOperatingSystemConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	return NewLogFields(obsCfg, osCfg)
}
