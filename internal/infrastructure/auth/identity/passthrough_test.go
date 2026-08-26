package identity

import (
	"context"
	"testing"

	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("passthroughResolver のインスタンスが生成される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, &passthroughResolver{}, New())
		})
	})
}

func Test_passthroughResolver_Resolve(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みの Authn を UserID 未解決のまま返す", func(t *testing.T) {
			t.Parallel()

			authn, err := authbd.New("subject", "issuer", nil, nil)
			require.NoError(t, err)

			resolved, err := New().Resolve(context.Background(), authn)
			require.NoError(t, err)
			assert.False(t, resolved.HasUserID())
			assert.Equal(t, "subject", resolved.Subject())
			assert.Equal(t, "issuer", resolved.Issuer())
		})
	})
}
