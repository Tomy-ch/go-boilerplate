package idempotency

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRequest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("載せた Request を Scope/Key/Fingerprint まで取り出せる", func(t *testing.T) {
			t.Parallel()
			want := Request{
				Scope:       "user-1",
				Key:         "key-1",
				Fingerprint: []byte("fp"),
				Method:      "POST",
				Path:        "/v1/resources",
				OperationID: "PostResources",
			}

			got, ok := requestFromContext(WithRequest(context.Background(), want))

			require.True(t, ok)
			assert.Equal(t, want, got)
		})

		t.Run("他 context 値と衝突せず Request を取り出せる", func(t *testing.T) {
			t.Parallel()
			type otherKey struct{}
			want := Request{Scope: "user-1", Key: "key-1", Fingerprint: []byte("fp")}
			ctx := context.WithValue(WithRequest(context.Background(), want), otherKey{}, "noise")

			got, ok := requestFromContext(ctx)

			require.True(t, ok)
			assert.Equal(t, want, got)
		})
	})
}

func Test_requestFromContext(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("冪等キーで載った Request をそのまま返し ok=true になる", func(t *testing.T) {
			t.Parallel()
			want := Request{Scope: "user-1", Key: "key-1", Fingerprint: []byte("fp"), Method: "POST", Path: "/v1/resources"}

			got, ok := requestFromContext(context.WithValue(context.Background(), requestCtxKey{}, want))

			require.True(t, ok)
			assert.Equal(t, want, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Request 未設定の ctx はゼロ値と ok=false を返す", func(t *testing.T) {
			t.Parallel()
			got, ok := requestFromContext(context.Background())

			assert.False(t, ok)
			assert.Equal(t, Request{}, got)
		})

		t.Run("別キーで載せた値は Request として取り出せない", func(t *testing.T) {
			t.Parallel()
			type otherKey struct{}
			ctx := context.WithValue(context.Background(), otherKey{}, Request{Scope: "x"})

			got, ok := requestFromContext(ctx)

			assert.False(t, ok)
			assert.Equal(t, Request{}, got)
		})

		t.Run("冪等キーに Request 以外の型が載っている場合はゼロ値と ok=false を返す", func(t *testing.T) {
			t.Parallel()
			ctx := context.WithValue(context.Background(), requestCtxKey{}, "not-a-request")

			got, ok := requestFromContext(ctx)

			assert.False(t, ok)
			assert.Equal(t, Request{}, got)
		})
	})
}
