package healthcheck

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &systemQuery{
		tracer: tf.Infra(),
		db:     testDB,
	}
	actual := New(testDB, tf)

	assert.Equal(t, expected, actual)
}

func Test_systemQuery_CheckDBHealth(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionRunner(t)

	s := &systemQuery{
		tracer: lt,
		db:     testDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのヘルスチェックが成功する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				res, err := s.CheckDBHealth(ctx)
				require.NoError(t, err)
				assert.True(t, res.Ready)
				assert.Positive(t, res.Latency.Microseconds())
				assert.NotZero(t, res.RespondedAt)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			res, err := s.CheckDBHealth(ctx)
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.False(t, res.Ready)
		})
	})
}
