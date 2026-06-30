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

func TestAuthorizer_Authorize(t *testing.T) {
	t.Parallel()

	newAuthn := func(t *testing.T) *authbd.Authn {
		t.Helper()
		authn, err := authbd.New("11111111-1111-1111-1111-111111111111", authbd.ProviderMock, nil, nil)
		require.NoError(t, err)

		return authn
	}

	ownerID := uuid.NewTestFromSalt(t, "owner")

	cases := map[string]struct {
		action   authzbd.Action
		resource *authzbd.Resource
	}{
		"取得操作_所有者ありリソース": {
			action:   authzbd.ActionUserGet,
			resource: authzbd.NewResource("user", &ownerID),
		},
		"更新操作_所有者ありリソース": {
			action:   authzbd.ActionUserUpdate,
			resource: authzbd.NewResource("user", &ownerID),
		},
		"削除操作_所有者ありリソース": {
			action:   authzbd.ActionUserDelete,
			resource: authzbd.NewResource("user", &ownerID),
		},
		"resourceがnilでも許可する": {
			action:   authzbd.ActionUserDelete,
			resource: nil,
		},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		for name, tc := range cases {
			t.Run("action_resourceによらず常に許可してnilを返す_"+name, func(t *testing.T) {
				t.Parallel()

				err := New().Authorize(context.Background(), newAuthn(t), tc.action, tc.resource)

				require.NoError(t, err)
			})
		}
	})
}
