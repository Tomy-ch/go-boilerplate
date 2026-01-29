package local

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	authbd "boilerplate-go/internal/usecase/boundary/auth"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("local.Authenticator のインスタンスが生成される", func(t *testing.T) {
		t.Parallel()
		expected := &authenticator{
			cfg: &config{
				provider: localProvider,
				prefix:   localPrefix,
			},
		}
		actual := New()
		require.Equal(t, expected, actual)
	})
}

func Test_authenticator_Authenticate(t *testing.T) {
	t.Parallel()

	t.Run("トークン文字列がそのまま Subject として返される", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		authenticator := New()
		cred, err := authbd.NewCredential("debug:some-subject")
		require.NoError(t, err)

		authn, err := authenticator.Authenticate(ctx, cred)
		require.NoError(t, err)
		require.Equal(t, "some-subject", authn.Subject())
		require.Equal(t, localProvider, authn.Provider())
	})

	t.Run("トークン文字列が prefix を含まない場合はエラーになる", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		authenticator := New()
		cred, err := authbd.NewCredential("invalid-token")
		require.NoError(t, err)

		authn, err := authenticator.Authenticate(ctx, cred)
		require.Nil(t, authn)
		require.ErrorIs(t, err, ErrLocalMockAuthenticatorInvalidToken)
	})
}

func Test_authenticator_resolveSubject(t *testing.T) {
	t.Parallel()

	t.Run("prefix を含むトークン文字列から prefix を除いた部分が返される", func(t *testing.T) {
		t.Parallel()
		authenticator := &authenticator{
			cfg: &config{
				provider: localProvider,
				prefix:   localPrefix,
			},
		}

		subject := authenticator.resolveSubject("debug:example-subject")
		require.Equal(t, "example-subject", subject)
	})

	t.Run("prefix を含まないトークン文字列の場合、空文字が返される", func(t *testing.T) {
		t.Parallel()
		authenticator := &authenticator{
			cfg: &config{
				provider: localProvider,
				prefix:   localPrefix,
			},
		}
		token := "invalid-token"

		subject := authenticator.resolveSubject(token)
		require.Empty(t, subject)
	})
}
