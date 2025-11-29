package healthcheck

import (
	"context"
	"testing"

	"boilerplate-go/internal/infrastructure/rdb/rdbtest"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	db, z, tf := rdbtest.NewTestInstancesForNew(t)
	expected := &systemQuery{
		tracer: tf.Infra(),
		db:     db,
		z:      z,
	}
	actual := New(db, z, tf)

	require.Equal(t, expected, actual)
}

func Test_healthCheckSystemQuery_GetDBHealth(t *testing.T) {
	t.Parallel()

	db, txm, z, _, tracer := rdbtest.NewTestInstancesForImplementedInfra(t)
	s := &systemQuery{
		tracer: tracer,
		db:     db,
		z:      z,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのヘルスチェックが成功する", func(t *testing.T) {
			t.Parallel()

			err := txm.Do(func(ctx context.Context) error {
				res, err := s.CheckDBHealth(ctx)
				require.NoError(t, err)
				require.True(t, res.Ready)
				require.Positive(t, res.Latency.Microseconds())
				require.NotZero(t, res.ResponsedAt)
				return nil
			})
			require.NoError(t, err)
		})
	})
}
