package tx

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeManager 単純にfnを呼ぶ。
type fakeManager struct {
	called   bool
	afterErr error
}

// recoveringManager panicをrecoverしてerrorに変換する。
type recoveringManager struct{}

// passthroughManager panicをそのまま外へ伝播させる。
type passthroughManager struct{}

func (m *fakeManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	m.called = true
	if err := fn(ctx); err != nil {
		return err
	}
	if m.afterErr != nil {
		return m.afterErr
	}
	return nil
}

func (m *recoveringManager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic recovered: %v", p)
		}
	}()
	return fn(ctx)
}

func (m *passthroughManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestDoWithResult(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("fnが値を返しエラーなし", func(t *testing.T) {
			t.Parallel()
			m := &fakeManager{}
			ctx := context.Background()
			expected := 42

			actual, err := DoWithResult(ctx, m, func(_ context.Context) (int, error) {
				return expected, nil
			})

			require.NoError(t, err)
			require.Equal(t, expected, actual)
			require.True(t, m.called)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("fnがエラーを返した場合はゼロ値を返す", func(t *testing.T) {
			t.Parallel()
			m := &fakeManager{}
			ctx := context.Background()
			expected := 0

			actual, err := DoWithResult(ctx, m, func(_ context.Context) (int, error) {
				return expected, errors.New("fn failed")
			})

			require.Error(t, err)
			require.Equal(t, expected, actual)
			require.True(t, m.called)
		})

		t.Run("fnは成功したがManager側でエラー（例: コミット失敗）", func(t *testing.T) {
			t.Parallel()
			m := &fakeManager{afterErr: errors.New("commit failed")}
			ctx := context.Background()

			actual, err := DoWithResult(ctx, m, func(_ context.Context) (string, error) {
				return "ok", nil
			})

			require.Error(t, err)
			require.Empty(t, actual)
			require.True(t, m.called)
		})

		t.Run("正常系: contextの値がfnに伝搬される", func(t *testing.T) {
			t.Parallel()
			type ctxKey struct{}
			const expected = "propagated"

			m := &fakeManager{}
			base := context.WithValue(context.Background(), ctxKey{}, expected)

			actual, err := DoWithResult(base, m, func(ctx context.Context) (string, error) {
				val, _ := ctx.Value(ctxKey{}).(string)
				return val, nil
			})

			require.NoError(t, err)
			require.Equal(t, expected, actual)
			require.True(t, m.called)
		})

		t.Run("panicをManagerがrecoverした場合はエラーとゼロ値を返す", func(t *testing.T) {
			t.Parallel()
			m := &recoveringManager{}
			ctx := context.Background()

			actual, err := DoWithResult(ctx, m, func(_ context.Context) (int, error) {
				panic("boom")
			})

			require.Error(t, err)
			require.Zero(t, actual)
		})

		t.Run("panicをManagerがrecoverしない場合はpanicが外へ伝播する", func(t *testing.T) {
			t.Parallel()
			m := &passthroughManager{}
			ctx := context.Background()

			require.Panics(t, func() {
				_, _ = DoWithResult(ctx, m, func(_ context.Context) (int, error) {
					panic("boom")
				})
			})
		})
	})
}
