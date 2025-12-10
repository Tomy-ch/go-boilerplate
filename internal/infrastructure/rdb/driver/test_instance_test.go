package driver

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNewMockInstance(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)

	actual := NewTestInstance(t, dbCfg)
	require.NotNil(t, actual)
	require.IsType(t, &dbDriver{}, actual)
}

func TestNewTestInstance(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)

	actual := NewTestInstance(t, dbCfg)
	require.NotNil(t, actual)
	require.IsType(t, &dbDriver{}, actual)

	dbDriver := actual.(*dbDriver)
	require.Equal(t, dbCfg, dbDriver.dbCfg)
}
