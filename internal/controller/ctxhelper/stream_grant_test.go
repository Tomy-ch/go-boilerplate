package ctxhelper

import (
	"context"
	"testing"

	rt "go-boilerplate/internal/usecase/boundary/realtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStreamGrant() rt.StreamGrant {
	return rt.StreamGrant{Subject: "subject-1", Destination: "stream-1", Scope: "read", InitialCursor: 7}
}

func TestWithStreamGrant(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロットを仕込むとSetStreamGrantが成功する", func(t *testing.T) {
			t.Parallel()
			ctx := WithStreamGrant(context.Background())
			assert.True(t, SetStreamGrant(ctx, newTestStreamGrant()))
		})
	})
}

func TestSetStreamGrant(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロットがあれば書き込めGetStreamGrantで読める", func(t *testing.T) {
			t.Parallel()
			want := newTestStreamGrant()
			ctx := WithStreamGrant(context.Background())

			require.True(t, SetStreamGrant(ctx, want))

			got, ok := GetStreamGrant(ctx)
			assert.True(t, ok)
			assert.Equal(t, want, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロット未仕込みなら書き込めずfalseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, SetStreamGrant(context.Background(), newTestStreamGrant()))
		})
	})
}

func TestGetStreamGrant(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("書き込み済みならok=trueで値を返す", func(t *testing.T) {
			t.Parallel()
			ctx := WithStreamGrant(context.Background())
			require.True(t, SetStreamGrant(ctx, newTestStreamGrant()))

			got, ok := GetStreamGrant(ctx)
			assert.True(t, ok)
			assert.Equal(t, "subject-1", got.Subject)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロットはあるが未設定ならok=false", func(t *testing.T) {
			t.Parallel()
			_, ok := GetStreamGrant(WithStreamGrant(context.Background()))
			assert.False(t, ok)
		})

		t.Run("スロット未仕込みならok=false", func(t *testing.T) {
			t.Parallel()
			_, ok := GetStreamGrant(context.Background())
			assert.False(t, ok)
		})
	})
}

func TestRequireStreamGrant(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("書き込み済みなら値を返す", func(t *testing.T) {
			t.Parallel()
			want := newTestStreamGrant()
			ctx := WithStreamGrant(context.Background())
			require.True(t, SetStreamGrant(ctx, want))

			got, err := RequireStreamGrant(ctx)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未設定ならErrStreamGrantMissingを返す", func(t *testing.T) {
			t.Parallel()
			_, err := RequireStreamGrant(WithStreamGrant(context.Background()))
			require.ErrorIs(t, err, ErrStreamGrantMissing)
		})
	})
}
