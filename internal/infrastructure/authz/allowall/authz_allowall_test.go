package allowall

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	authzbd "go-boilerplate/internal/usecase/boundary/authz"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	newAppCfg := func(t *testing.T, env string) *config.ApplicationConfig {
		t.Helper()
		appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
		appCfg.SetApplicationEnv(t, env)

		return appCfg
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		// 全許可を許容する非本番環境。
		for _, env := range []string{config.EnvLocal, config.EnvCI, config.EnvTest} {
			t.Run(env+"環境ではAuthorizerを生成する", func(t *testing.T) {
				t.Parallel()
				authorizer, err := New(newAppCfg(t, env))
				require.NoError(t, err)
				assert.NotNil(t, authorizer)
			})
		}
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// 本番相当の環境では生成を拒否し、危険な全許可が配線されないこと（fail-closed）。
		for _, env := range []string{config.EnvDevelopment, config.EnvStaging, config.EnvProduction} {
			t.Run(env+"環境では生成を拒否してエラーを返す", func(t *testing.T) {
				t.Parallel()
				authorizer, err := New(newAppCfg(t, env))
				require.ErrorIs(t, err, errNonLocalEnv)
				require.ErrorContains(t, err, env)
				assert.Nil(t, authorizer)
			})
		}
	})
}

func Test_authorizer_Authorize(t *testing.T) {
	t.Parallel()

	newAuthn := func(t *testing.T) *authbd.Authn {
		t.Helper()
		authn, err := authbd.New("11111111-1111-1111-1111-111111111111", authbd.IssuerMock, nil, nil)
		require.NoError(t, err)

		return authn
	}

	ownerID := uuidtestkit.NewTestFromSalt(t, "owner")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得操作_所有者ありリソースを許可してnilを返す", func(t *testing.T) {
			t.Parallel()
			err := (&authorizer{}).Authorize(context.Background(), newAuthn(t), authzbd.ActionUserGet, authzbd.NewResource("user", &ownerID))
			require.NoError(t, err)
		})

		t.Run("更新操作_所有者ありリソースを許可してnilを返す", func(t *testing.T) {
			t.Parallel()
			err := (&authorizer{}).Authorize(context.Background(), newAuthn(t), authzbd.ActionUserUpdate, authzbd.NewResource("user", &ownerID))
			require.NoError(t, err)
		})

		t.Run("削除操作_所有者ありリソースを許可してnilを返す", func(t *testing.T) {
			t.Parallel()
			err := (&authorizer{}).Authorize(context.Background(), newAuthn(t), authzbd.ActionUserDelete, authzbd.NewResource("user", &ownerID))
			require.NoError(t, err)
		})

		t.Run("resourceがnilでも許可してnilを返す", func(t *testing.T) {
			t.Parallel()
			err := (&authorizer{}).Authorize(context.Background(), newAuthn(t), authzbd.ActionUserDelete, nil)
			require.NoError(t, err)
		})
	})
}
