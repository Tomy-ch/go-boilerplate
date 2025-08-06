package echorecover

import (
	"testing"

	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/require"
)

func TestDevelopmentConfig(t *testing.T) {
	t.Parallel()
	expected := middleware.RecoverConfig{
		StackSize:         10 << 10,
		DisableStackAll:   false,
		DisablePrintStack: false,
		LogLevel:          log.DEBUG,
	}

	actual := developmentConfig()

	require.Equal(t, expected, actual)
}

func TestProductionConfig(t *testing.T) {
	t.Parallel()
	expected := middleware.RecoverConfig{
		StackSize:         4 << 10,
		DisableStackAll:   true,
		DisablePrintStack: true,
		LogLevel:          log.ERROR,
	}

	actual := productionConfig()
	require.Equal(t, expected, actual)
}
