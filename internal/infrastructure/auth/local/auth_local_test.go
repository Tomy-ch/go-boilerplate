package local

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authbd "go-boilerplate/internal/usecase/boundary/auth"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("local.Authenticator のインスタンスが生成される", func(t *testing.T) {
			t.Parallel()
			expected := &authenticator{}
			actual := New()
			assert.Equal(t, expected, actual)
		})
	})
}

func Test_authenticator_Authenticate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トークン文字列がそのまま Subject として返される", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			authenticator := New()
			cred, err := authbd.NewCredential(authbd.SchemeBearer, "debug:some-subject")
			require.NoError(t, err)

			authn, err := authenticator.Authenticate(ctx, cred)
			require.NoError(t, err)
			assert.Equal(t, "some-subject", authn.Subject())
			assert.Equal(t, authbd.IssuerMock, authn.Issuer())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トークン文字列が prefix を含まない場合はエラーになる", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			authenticator := New()
			cred, err := authbd.NewCredential(authbd.SchemeBearer, "invalid-token")
			require.NoError(t, err)

			authn, err := authenticator.Authenticate(ctx, cred)
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrLocalMockAuthenticatorInvalidToken)
		})

		t.Run("cred が nil の場合は panic せずエラーになる", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			authenticator := New()

			authn, err := authenticator.Authenticate(ctx, nil)
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrLocalMockAuthenticatorInvalidToken)
		})
	})
}

func Test_authenticator_resolveSubject(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("prefix を含むトークン文字列から prefix を除いた部分が返される", func(t *testing.T) {
			t.Parallel()
			authenticator := &authenticator{}

			subject := authenticator.resolveSubject("debug:example-subject")
			assert.Equal(t, "example-subject", subject)
		})

		t.Run("prefix を含まないトークン文字列の場合、空文字が返される", func(t *testing.T) {
			t.Parallel()
			authenticator := &authenticator{}
			token := "invalid-token"

			subject := authenticator.resolveSubject(token)
			assert.Empty(t, subject)
		})

		t.Run("prefixのみでsubjectが空の場合、空文字が返される", func(t *testing.T) {
			t.Parallel()
			authenticator := &authenticator{}
			assert.Empty(t, authenticator.resolveSubject("debug:"))
		})

		t.Run("prefixありでsubjectが空白のみの場合、空文字が返される", func(t *testing.T) {
			t.Parallel()
			authenticator := &authenticator{}
			assert.Empty(t, authenticator.resolveSubject("debug:   "))
		})
	})
}
