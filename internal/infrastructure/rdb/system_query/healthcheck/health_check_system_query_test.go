package healthcheck

import (
	"context"
	"testing"

	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &systemQuery{
		tracer: tf.Infra(),
		db:     loggingDB,
	}
	actual := New(loggingDB, tf)

	assert.Equal(t, expected, actual)
}

func Test_healthCheckSystemQuery_GetDBHealth(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionRunner(t)

	s := &systemQuery{
		tracer: lt,
		db:     loggingDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのヘルスチェックが成功する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				res, err := s.CheckDBHealth(ctx)
				require.NoError(t, err)
				assert.True(t, res.Ready)
				require.Positive(t, res.Latency.Microseconds())
				require.NotZero(t, res.ResponsedAt)
			})
		})
	})
}
