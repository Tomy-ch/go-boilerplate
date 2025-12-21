package healthcheck

import (
	"context"
	"testing"

	"boilerplate-go/internal/infrastructure/rdb/rdbtest"
	"boilerplate-go/internal/observability"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	db, provider := rdbtest.NewTestDBWithLoggingProvider(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &systemQuery{
		tracer:   tf.Infra(),
		db:       db,
		provider: provider,
	}
	actual := New(db, provider, tf)

	require.Equal(t, expected, actual)
}

func Test_healthCheckSystemQuery_GetDBHealth(t *testing.T) {
	t.Parallel()

	db, provider := rdbtest.NewTestDBWithLoggingProvider(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := rdbtest.NewTestTransactionManager(t)

	s := &systemQuery{
		tracer:   lt,
		db:       db,
		provider: provider,
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
			})
		})
	})
}
