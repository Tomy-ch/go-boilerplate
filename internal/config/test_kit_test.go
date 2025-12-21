package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewTestLocation(t *testing.T) {
	t.Parallel()

	cfg := MockConfigForTest(t)
	osCfg := NewOSConfig(cfg)

	expected, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)

	actual := NewTestLocation(t)
	require.Equal(t, expected, actual)
}
