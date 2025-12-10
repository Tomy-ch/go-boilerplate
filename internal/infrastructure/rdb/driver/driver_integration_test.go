package driver

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestExecContext_ContextTimeout(t *testing.T) {
	t.Parallel()

	defaultTimeout := 500 * time.Millisecond
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbCfg.SetDefaultTimeout(t, defaultTimeout)

	db := NewTestInstance(t, dbCfg).(*dbDriver).db

	d := &dbDriver{
		db:    db,
		dbCfg: dbCfg,
	}

	ctx := context.Background()
	start := time.Now()

	res, err := d.ExecContext(ctx, `SELECT pg_sleep(2);`)
	require.Nil(t, res)
	require.Error(t, err)

	require.ErrorContains(t, err, "context deadline exceeded")
	elapsed := time.Since(start)
	require.Less(t, elapsed, 2*time.Second)
}

func TestQueryContext_ContextTimeout(t *testing.T) {
	t.Parallel()

	defaultTimeout := 500 * time.Millisecond
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbCfg.SetDefaultTimeout(t, defaultTimeout)

	db := NewTestInstance(t, dbCfg).(*dbDriver).db

	d := &dbDriver{
		db:    db,
		dbCfg: dbCfg,
	}

	ctx := context.Background()
	start := time.Now()

	rows, err := d.QueryContext(ctx, `SELECT pg_sleep(2);`) //nolint:rowserrcheck // rows はエラー時にnilになるため
	require.Error(t, err)
	require.Nil(t, rows)

	require.ErrorContains(t, err, "context deadline exceeded")

	elapsed := time.Since(start)
	require.Less(t, elapsed, 2*time.Second)
}

func TestQueryRowContext_ContextTimeout(t *testing.T) {
	t.Parallel()

	defaultTimeout := 500 * time.Millisecond
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbCfg.SetDefaultTimeout(t, defaultTimeout)

	db := NewTestInstance(t, dbCfg).(*dbDriver).db

	d := &dbDriver{
		db:    db,
		dbCfg: dbCfg,
	}

	ctx := context.Background()
	start := time.Now()

	row := d.QueryRowContext(ctx, `SELECT pg_sleep(2);`)

	var dummy any
	err := row.Scan(&dummy)
	require.Error(t, err)
	require.ErrorContains(t, err, "context deadline exceeded")

	elapsed := time.Since(start)
	require.Less(t, elapsed, 2*time.Second)
}

func TestPrepareContext_Succeeds(t *testing.T) {
	t.Parallel()

	defaultTimeout := 500 * time.Millisecond
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbCfg.SetDefaultTimeout(t, defaultTimeout)

	db := NewTestInstance(t, dbCfg).(*dbDriver).db

	d := &dbDriver{
		db:    db,
		dbCfg: dbCfg,
	}

	ctx := context.Background()

	stmt, err := d.PrepareContext(ctx, `SELECT 1;`)
	require.NoError(t, err)
	require.NotNil(t, stmt)
	defer func() {
		require.NoError(t, stmt.Close())
	}()
}
