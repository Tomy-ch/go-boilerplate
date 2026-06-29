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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("action_resourceによらず常に許可してnilを返す", func(t *testing.T) {
			t.Parallel()
			authn, err := authbd.New("11111111-1111-1111-1111-111111111111", authbd.ProviderMock, nil, nil)
			require.NoError(t, err)
			id := uuid.NewTestFromSalt(t, "owner")

			err = New().Authorize(context.Background(), authn, authzbd.ActionUserDelete, authzbd.NewResource("user", &id))

			require.NoError(t, err)
		})
	})
}
