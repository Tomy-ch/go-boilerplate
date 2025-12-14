package loggingdb

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"

	"github.com/stretchr/testify/require"
)

func TestNewLoggingDBProvider(t *testing.T) {
	t.Parallel()

	db := driver.NewTestInstance(t)
	l := logging.NewTestLogger(t)

	tracer := observability.NewNoopTracerFactory(t)

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)

	expected := &provider{
		db:     db,
		dbCfg:  dbCfg,
		l:      l,
		lf:     lf,
		tracer: tracer.Infra(),
	}

	provider := NewLoggingDBProvider(db, dbCfg, l, lf, tracer)
	require.Equal(t, expected, provider)
}

func TestLoggingDBProvider_NewLoggingDB(t *testing.T) {
	t.Parallel()

	db := driver.NewTestInstance(t)
	l := logging.NewTestLogger(t)

	lf := logging.NewTestLogFieldBuilder(t)

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

func Test_provider_Logger(t *testing.T) {
	t.Parallel()

	db := driver.NewTestInstance(t)
	l := logging.NewTestLogger(t)

	provider := &provider{
		db: db,
		l:  l,
	}

	require.Equal(t, l, provider.Logger())
}

func Test_provider_LogFields(t *testing.T) {
	t.Parallel()

	db := driver.NewTestInstance(t)

	lf := logging.NewTestLogFieldBuilder(t)

	provider := &provider{
		db: db,
		lf: lf,
	}

	require.Equal(t, lf, provider.LogFields())
}

func Test_provider_DBConfig(t *testing.T) {
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

func Test_provider_LayerTracer(t *testing.T) {
	t.Parallel()

	tracer := observability.NewNoopTracerFactory(t)

	expectedTracer := tracer.Infra()

	provider := &provider{
		tracer: expectedTracer,
	}

	require.Equal(t, expectedTracer, provider.LayerTracer())
}
