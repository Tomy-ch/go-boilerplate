package allowall

import (
	"context"
	"testing"

	authbd "go-boilerplate/internal/usecase/boundary/auth"
	authzbd "go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Authorizerを生成する", func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, New())
		})
	})
}

func Test_authorizer_Authorize(t *testing.T) {
	t.Parallel()

	newAuthn := func(t *testing.T) *authbd.Authn {
		t.Helper()
		authn, err := authbd.New("11111111-1111-1111-1111-111111111111", authbd.ProviderMock, nil, nil)
		require.NoError(t, err)

		return authn
	}

	ownerID := uuid.NewTestFromSalt(t, "owner")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得操作_所有者ありリソースを許可してnilを返す", func(t *testing.T) {
			t.Parallel()
			err := New().Authorize(context.Background(), newAuthn(t), authzbd.ActionUserGet, authzbd.NewResource("user", &ownerID))
			require.NoError(t, err)
		})

		t.Run("更新操作_所有者ありリソースを許可してnilを返す", func(t *testing.T) {
			t.Parallel()
			err := New().Authorize(context.Background(), newAuthn(t), authzbd.ActionUserUpdate, authzbd.NewResource("user", &ownerID))
			require.NoError(t, err)
		})

		t.Run("削除操作_所有者ありリソースを許可してnilを返す", func(t *testing.T) {
			t.Parallel()
			err := New().Authorize(context.Background(), newAuthn(t), authzbd.ActionUserDelete, authzbd.NewResource("user", &ownerID))
			require.NoError(t, err)
		})

		t.Run("resourceがnilでも許可してnilを返す", func(t *testing.T) {
			t.Parallel()
			err := New().Authorize(context.Background(), newAuthn(t), authzbd.ActionUserDelete, nil)
			require.NoError(t, err)
		})
	})
}
