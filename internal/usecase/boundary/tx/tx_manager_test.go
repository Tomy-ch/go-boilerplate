package tx_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go-boilerplate/internal/usecase/boundary/tx"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// passthroughManager は、fn をそのまま実行する Manager mock を返します。
func passthroughManager(t *testing.T) tx.Manager {
	t.Helper()
	m := mock_tx.NewMockManager(gomock.NewController(t))
	m.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	return m
}

// afterErrManager は、fn 成功後に afterErr を返す Manager mock を返します（コミット失敗の模擬）。
func afterErrManager(t *testing.T, afterErr error) tx.Manager {
	t.Helper()
	m := mock_tx.NewMockManager(gomock.NewController(t))
	m.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			if err := fn(ctx); err != nil {
				return err
			}
			return afterErr
		})
	return m
}

// recoveringManager は、fn の panic を recover して error に変換する Manager mock を返します。
func recoveringManager(t *testing.T) tx.Manager {
	t.Helper()
	m := mock_tx.NewMockManager(gomock.NewController(t))
	m.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) (err error) {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("panic recovered: %v", p)
				}
			}()
			return fn(ctx)
		})
	return m
}

func TestDoWithResult(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("fnが値を返しエラーなし", func(t *testing.T) {
			t.Parallel()
			m := passthroughManager(t)
			ctx := context.Background()
			expected := 42

			actual, err := tx.DoWithResult(ctx, m, func(_ context.Context) (int, error) {
				return expected, nil
			})

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("fnがエラーを返した場合はゼロ値を返す", func(t *testing.T) {
			t.Parallel()
			m := passthroughManager(t)
			ctx := context.Background()
			expected := 0

			actual, err := tx.DoWithResult(ctx, m, func(_ context.Context) (int, error) {
				return expected, errors.New("fn failed")
			})

			require.Error(t, err)
			assert.Equal(t, expected, actual)
		})

		t.Run("fnは成功したがManager側でエラー（例: コミット失敗）", func(t *testing.T) {
			t.Parallel()
			m := afterErrManager(t, errors.New("commit failed"))
			ctx := context.Background()

			actual, err := tx.DoWithResult(ctx, m, func(_ context.Context) (string, error) {
				return "ok", nil
			})

			require.Error(t, err)
			assert.Empty(t, actual)
		})

		t.Run("正常系: contextの値がfnに伝搬される", func(t *testing.T) {
			t.Parallel()
			type ctxKey struct{}
			const expected = "propagated"

			m := passthroughManager(t)
			base := context.WithValue(context.Background(), ctxKey{}, expected)

			actual, err := tx.DoWithResult(base, m, func(ctx context.Context) (string, error) {
				val, _ := ctx.Value(ctxKey{}).(string)
				return val, nil
			})

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})

		t.Run("panicをManagerがrecoverした場合はエラーとゼロ値を返す", func(t *testing.T) {
			t.Parallel()
			m := recoveringManager(t)
			ctx := context.Background()

			actual, err := tx.DoWithResult(ctx, m, func(_ context.Context) (int, error) {
				panic("boom")
			})

			require.Error(t, err)
			assert.Zero(t, actual)
		})

		t.Run("panicをManagerがrecoverしない場合はpanicが外へ伝播する", func(t *testing.T) {
			t.Parallel()
			m := passthroughManager(t)
			ctx := context.Background()

			require.Panics(t, func() {
				_, _ = tx.DoWithResult(ctx, m, func(_ context.Context) (int, error) {
					panic("boom")
				})
			})
		})
	})
}
