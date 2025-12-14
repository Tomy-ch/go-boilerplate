package logging

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNewTestLogger(t *testing.T) {
	t.Parallel()

	actual := NewTestLogger(t)
	require.NotNil(t, actual)
}

func TestNewTestLogFieldBuilder(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	osCfg := config.NewOSConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)

	actual := NewTestLogFieldBuilder(t)

	expected := &logFieldBuilder{
		obsCfg: obsCfg,
		osCfg:  osCfg,
	}
	require.Equal(t, expected, actual)
}
