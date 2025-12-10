package loggingdb

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
)

func TestNewLoggingDBProvider(t *testing.T) {
	t.Parallel()

	db := driver.NewTestInstance(t)
	l := logging.NewTestInstance(t)

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	expected := &provider{
		db:    db,
		dbCfg: dbCfg,
		l:     l,
		lf:    lf,
	}

	provider := NewLoggingDBProvider(db, dbCfg, l, lf)
	require.Equal(t, expected, provider)
}

func TestLoggingDBProvider_NewLoggingDB(t *testing.T) {
	t.Parallel()

	db := driver.NewTestInstance(t)
	l := logging.NewTestInstance(t)

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	provider := &provider{
		db: db,
		l:  l,
		lf: lf,
	}
	ctx := context.Background()
	loggingDB := provider.NewLoggingDB(ctx)

	require.IsType(t, &dbWithLogging{}, loggingDB)
	dwl := loggingDB.(*dbWithLogging)
	require.Equal(t, ctx, dwl.ctx)
	require.Equal(t, provider, dwl.provider)
}

func TestLoggingDBProvider_logger(t *testing.T) {
	t.Parallel()

	db := driver.NewTestInstance(t)
	l := logging.NewTestInstance(t)

	provider := &provider{
		db: db,
		l:  l,
	}

	require.Equal(t, l, provider.Logger())
}

func TestLoggingDBProvider_logFields(t *testing.T) {
	t.Parallel()

	db := driver.NewTestInstance(t)

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	provider := &provider{
		db: db,
		lf: lf,
	}

	require.Equal(t, lf, provider.LogFields())
}

func TestLoggingDBProvider_dbConfig(t *testing.T) {
	t.Parallel()

	db := driver.NewTestInstance(t)
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)

	provider := &provider{
		db:    db,
		dbCfg: dbCfg,
	}

	require.Equal(t, dbCfg, provider.DBConfig())
}
