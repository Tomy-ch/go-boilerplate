package ctxhelper

import (
	"context"
	"testing"

	"go-boilerplate/internal/usecase/boundary/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuthn(t *testing.T, subject string) auth.Authn {
	t.Helper()
	a, err := auth.New(subject, auth.IssuerMock, nil, nil)
	require.NoError(t, err)
	return *a
}

func TestWithAuthn(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロットを仕込むとSetAuthnが成功する", func(t *testing.T) {
			t.Parallel()
			ctx := WithAuthn(context.Background())
			assert.True(t, SetAuthn(ctx, newTestAuthn(t, "u1")))
		})
	})
}

func TestSetAuthn(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロットがあれば書き込めGetAuthnで読める", func(t *testing.T) {
			t.Parallel()
			want := newTestAuthn(t, "u1")
			ctx := WithAuthn(context.Background())

			require.True(t, SetAuthn(ctx, want))

			got, ok := GetAuthn(ctx)
			assert.True(t, ok)
			assert.Equal(t, want.Subject(), got.Subject())
			assert.Equal(t, want.Issuer(), got.Issuer())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロット未仕込みなら書き込めずfalseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, SetAuthn(context.Background(), newTestAuthn(t, "u1")))
		})
	})
}

func TestGetAuthn(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スロット未仕込みならokはfalse", func(t *testing.T) {
			t.Parallel()
			_, ok := GetAuthn(context.Background())
			assert.False(t, ok)
		})

		t.Run("スロットはあるが未設定ならokはfalse", func(t *testing.T) {
			t.Parallel()
			ctx := WithAuthn(context.Background())
			_, ok := GetAuthn(ctx)
			assert.False(t, ok)
		})
	})
}
