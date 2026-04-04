package logging

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("production mode", func(t *testing.T) {
		appCfg := config.NewApplicationConfig(&config.Config{})
		appCfg.SetApplicationMode(t, "production")

		logger, err := New(appCfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("development mode", func(t *testing.T) {
		appCfg := config.NewApplicationConfig(&config.Config{})
		appCfg.SetApplicationMode(t, "development")

		logger, err := New(appCfg)
		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("unknown mode", func(t *testing.T) {
		appCfg := config.NewApplicationConfig(&config.Config{})
		appCfg.SetApplicationMode(t, "unknown")

		logger, err := New(appCfg)
		require.Error(t, err)
		require.Nil(t, logger)
	})
}

func TestNewProductionLogger(t *testing.T) {
	logger, err := NewProductionLogger()
	require.NoError(t, err)
	require.NotNil(t, logger)
}

func TestNewDevelopmentLogger(t *testing.T) {
	logger, err := NewDevelopmentLogger()
	require.NoError(t, err)
	require.NotNil(t, logger)
}
