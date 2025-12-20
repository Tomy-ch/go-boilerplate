package datetime

import (
	"testing"
	"time"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNewTestLocation(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	osCfg := config.NewOSConfig(cfg)

	expected, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)

	actual := NewTestLocation(t)
	require.Equal(t, expected, actual)
}
