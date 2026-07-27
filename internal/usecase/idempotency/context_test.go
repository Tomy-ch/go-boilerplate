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

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Request 未設定の ctx は ok=false を返す", func(t *testing.T) {
			t.Parallel()
			_, ok := requestFromContext(context.Background())

			assert.False(t, ok)
		})

		t.Run("別キーで載せた値は Request として取り出せない", func(t *testing.T) {
			t.Parallel()
			type otherKey struct{}
			ctx := context.WithValue(context.Background(), otherKey{}, Request{Scope: "x"})

			_, ok := requestFromContext(ctx)

			assert.False(t, ok)
		})
	})
}

func Test_requestFromContext(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
