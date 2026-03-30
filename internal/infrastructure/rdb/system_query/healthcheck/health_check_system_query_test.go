package healthcheck

import (
	"context"
	"testing"

	"boilerplate-go/internal/infrastructure/rdb/testkit"
	"boilerplate-go/internal/observability"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	dbPool := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &systemQuery{
		tracer: tf.Infra(),
		db:     loggingDB,
		dbPool: dbPool,
	}
	actual := New(loggingDB, dbPool, tf)

	require.Equal(t, expected, actual)
}

func Test_healthCheckSystemQuery_GetDBHealth(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	dbPool := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionManager(t)

	s := &systemQuery{
		tracer: lt,
		db:     loggingDB,
		dbPool: dbPool,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのヘルスチェックが成功する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				res, err := s.CheckDBHealth(ctx)
				require.NoError(t, err)
				require.True(t, res.Ready)
				require.Positive(t, res.Latency.Microseconds())
				require.NotZero(t, res.ResponsedAt)
				require.Positive(t, res.TotalConnections)
				require.Positive(t, res.IdleConnections)
				require.Positive(t, res.AcquiredCount)
				require.Positive(t, res.MaxConnections)
			})
		})
	})
}
