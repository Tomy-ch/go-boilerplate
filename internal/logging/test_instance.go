package logging

import (
	"testing"

	"go.uber.org/zap"
)

// NewTestInstance は、テスト用のLoggerインスタンスを生成します。
func NewTestInstance(t *testing.T) Logger {
	t.Helper()
	return &logger{log: zap.NewNop()}
}
