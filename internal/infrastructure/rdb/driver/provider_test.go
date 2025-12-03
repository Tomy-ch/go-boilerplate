package driver

import (
	"context"
	"database/sql"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewLoggingDBProvider(t *testing.T) {
	t.Parallel()

	db := &dbDriver{&sql.DB{}}
	l := zap.NewNop()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	expected := &loggingDBProvider{
		db: db,
		l:  l,
		lf: lf,
	}

	provider := NewLoggingDBProvider(db, l, lf)
	require.Equal(t, expected, provider)
}

func TestLoggingDBProvider_NewLoggingDB(t *testing.T) {
	t.Parallel()

	db := &dbDriver{&sql.DB{}}
	l := zap.NewNop()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	provider := &loggingDBProvider{
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
