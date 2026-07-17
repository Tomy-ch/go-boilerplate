package auth

import (
	"testing"

	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		issuer := IssuerMock
		scopes := []string{}
		claims := map[string]any{"role": "user"}

		t.Run("subjectをuuidとして解釈できる場合、userIDはnilにならない", func(t *testing.T) {
			t.Parallel()
			subject := "550e8400-e29b-41d4-a716-446655440000"

			id, err := uuid.Parse(subject)
			require.NoError(t, err)

			expected := &Authn{
				subject: subject,
				userID:  &id,
				issuer:  issuer,
				scopes:  scopes,
				claims:  claims,
			}

			authn, err := New(subject, issuer, scopes, claims)
			require.NoError(t, err)
			assert.Equal(t, expected, authn)
		})

		t.Run("subjectをuuidとして解釈できない場合、userIDはnilになる", func(t *testing.T) {
			t.Parallel()
			subject := "non-uuid-subject"

			expected := &Authn{
				subject: subject,
				issuer:  issuer,
				scopes:  scopes,
				claims:  claims,
			}

			authn, err := New(subject, issuer, scopes, claims)
			require.NoError(t, err)
			assert.Equal(t, expected, authn)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("subjectが空の場合、エラーになる", func(t *testing.T) {
			t.Parallel()
			subject := ""
			issuer := IssuerMock
			scopes := []string{}
			claims := map[string]any{}

			authn, err := New(subject, issuer, scopes, claims)
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrUnauthenticatedSubjectMissing)
		})

		t.Run("subjectが空白のみの場合、エラーになる", func(t *testing.T) {
			t.Parallel()
			authn, err := New("   ", IssuerMock, []string{}, map[string]any{})
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrUnauthenticatedSubjectMissing)
		})
	})
}

func TestAuthn_Subject(t *testing.T) {
	t.Parallel()

	t.Run("Subjectはコンストラクタで与えたsubjectを返す", func(t *testing.T) {
		t.Parallel()
		subject := "test-subject"
		authn, err := New(subject, IssuerMock, []string{}, map[string]any{})
		require.NoError(t, err)

		assert.Equal(t, subject, authn.Subject())
	})
}

func TestAuthn_HasUserID(t *testing.T) {
	t.Parallel()

	t.Run("HasUserIDはUUIDとして解釈できた場合にtrueを返す", func(t *testing.T) {
		t.Parallel()
		subject := "550e8400-e29b-41d4-a716-446655440000"
		authn, err := New(subject, IssuerMock, []string{}, map[string]any{})
		require.NoError(t, err)

		assert.True(t, authn.HasUserID())
	})

	t.Run("HasUserIDはUUIDとして解釈できなかった場合にfalseを返す", func(t *testing.T) {
		t.Parallel()
		subject := "non-uuid-subject"
		authn, err := New(subject, IssuerMock, []string{}, map[string]any{})
		require.NoError(t, err)

		assert.False(t, authn.HasUserID())
	})
}

func TestAuthn_UserID(t *testing.T) {
	t.Parallel()

	t.Run("UserIDはUUIDとして解釈できた場合にUUIDを返す", func(t *testing.T) {
		t.Parallel()
		subject := "550e8400-e29b-41d4-a716-446655440000"
		expectedID, err := uuid.Parse(subject)
		require.NoError(t, err)

		authn, err := New(subject, IssuerMock, []string{}, map[string]any{})
		require.NoError(t, err)

		id, err := authn.UserID()
		require.NoError(t, err)
		assert.Equal(t, expectedID, id)
	})

	t.Run("UserIDはUUIDとして解釈できなかった場合にエラーを返す", func(t *testing.T) {
		t.Parallel()
		subject := "non-uuid-subject"

		authn, err := New(subject, IssuerMock, []string{}, map[string]any{})
		require.NoError(t, err)

		id, err := authn.UserID()
		require.ErrorIs(t, err, ErrSubjectNotUUID)
		assert.Equal(t, uuid.UUID{}, id)
	})
}

func TestAuthn_Issuer(t *testing.T) {
	t.Parallel()

	t.Run("Issuerはコンストラクタで与えたissuerを返す", func(t *testing.T) {
		t.Parallel()
		issuer := IssuerMock
		authn, err := New("test-subject", issuer, []string{}, map[string]any{})
		require.NoError(t, err)

		assert.Equal(t, issuer, authn.Issuer())
	})
}

func TestAuthn_Scopes(t *testing.T) {
	t.Parallel()

	t.Run("Scopesはコンストラクタで与えたscopesを返す", func(t *testing.T) {
		t.Parallel()
		scopes := []string{"scope1", "scope2"}
		authn, err := New("test-subject", IssuerMock, scopes, map[string]any{})
		require.NoError(t, err)

		assert.Equal(t, scopes, authn.Scopes())
	})

	t.Run("元のscopesや戻り値を変更しても内部状態は不変", func(t *testing.T) {
		t.Parallel()
		scopes := []string{"scope1", "scope2"}
		authn, err := New("test-subject", IssuerMock, scopes, map[string]any{})
		require.NoError(t, err)

		scopes[0] = "mutated"
		got := authn.Scopes()
		got[0] = "mutated-too"

		assert.Equal(t, []string{"scope1", "scope2"}, authn.Scopes())
	})
}

func TestAuthn_Claims(t *testing.T) {
	t.Parallel()

	t.Run("Claimsはコンストラクタで与えたclaimsを返す", func(t *testing.T) {
		t.Parallel()
		claims := map[string]any{"role": "admin", "exp": 1234567890}
		authn, err := New("test-subject", IssuerMock, []string{}, claims)
		require.NoError(t, err)

		assert.Equal(t, claims, authn.Claims())
	})

	t.Run("元のclaimsや戻り値を変更しても内部状態は不変", func(t *testing.T) {
		t.Parallel()
		claims := map[string]any{"role": "user"}
		authn, err := New("test-subject", IssuerMock, []string{}, claims)
		require.NoError(t, err)

		claims["role"] = "admin"
		got := authn.Claims()
		got["role"] = "root"

		assert.Equal(t, map[string]any{"role": "user"}, authn.Claims())
	})
}
