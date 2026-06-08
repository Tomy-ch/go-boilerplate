package logging

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestLogger(t *testing.T) {
	t.Parallel()

	lg := NewTestLogger(t)

	// 返る Logger が実際に使えること（メソッドチェーンと出力で panic しない）を検証する。
	require.NotPanics(t, func() {
		lg.Named("test").CallerSkip(1).Info("message", String("key", "value"))
	})
}

func TestNewTestLogFieldBuilder(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	osCfg := config.NewOperatingSystemConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)

	actual := NewTestLogFieldBuilder(t)

	expected := &logFieldBuilder{
		obsCfg: obsCfg,
		osCfg:  osCfg,
	}
	assert.Equal(t, expected, actual)
}
