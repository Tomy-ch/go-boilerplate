package auth

import (
	"testing"

	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		provider := ProviderMock
		scopes := []string{}
		claims := map[string]any{"role": "user"}

		t.Run("subjectをuuidとして解釈できる場合、uuidはnilにならない", func(t *testing.T) {
			t.Parallel()
			subject := "550e8400-e29b-41d4-a716-446655440000"

			id, err := uuid.Parse(subject)
			require.NoError(t, err)

			expected := &Authn{
				subject:  subject,
				id:       &id,
				provider: provider,
				scopes:   scopes,
				claims:   claims,
			}

			authn, err := New(subject, provider, scopes, claims)
			require.NoError(t, err)
			require.Equal(t, expected, authn)
		})

		t.Run("subjectをuuidとして解釈できない場合、uuidはnilになる", func(t *testing.T) {
			t.Parallel()
			subject := "non-uuid-subject"

			expected := &Authn{
				subject:  subject,
				provider: provider,
				scopes:   scopes,
				claims:   claims,
			}

			authn, err := New(subject, provider, scopes, claims)
			require.NoError(t, err)
			require.Equal(t, expected, authn)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("subjectが空の場合、エラーになる", func(t *testing.T) {
			t.Parallel()
			subject := ""
			provider := ProviderMock
			scopes := []string{}
			claims := map[string]any{}

			authn, err := New(subject, provider, scopes, claims)
			require.Nil(t, authn)
			require.ErrorIs(t, err, ErrUnauthorizedSubjectMissing)
		})
	})
}

func TestAuthn_Subject(t *testing.T) {
	t.Parallel()

	t.Run("Subjectはコンストラクタで与えたsubjectを返す", func(t *testing.T) {
		t.Parallel()
		subject := "test-subject"
		authn, err := New(subject, ProviderMock, []string{}, map[string]any{})
		require.NoError(t, err)

		require.Equal(t, subject, authn.Subject())
	})
}

func TestAuthn_HasID(t *testing.T) {
	t.Parallel()

	t.Run("HasIDはUUIDとして解釈できた場合にtrueを返す", func(t *testing.T) {
		t.Parallel()
		subject := "550e8400-e29b-41d4-a716-446655440000"
		authn, err := New(subject, ProviderMock, []string{}, map[string]any{})
		require.NoError(t, err)

		require.True(t, authn.HasID())
	})

	t.Run("HasIDはUUIDとして解釈できなかった場合にfalseを返す", func(t *testing.T) {
		t.Parallel()
		subject := "non-uuid-subject"
		authn, err := New(subject, ProviderMock, []string{}, map[string]any{})
		require.NoError(t, err)

		require.False(t, authn.HasID())
	})
}

func TestAuthn_ID(t *testing.T) {
	t.Parallel()

	t.Run("IDはUUIDとして解釈できた場合にUUIDを返す", func(t *testing.T) {
		t.Parallel()
		subject := "550e8400-e29b-41d4-a716-446655440000"
		expectedID, err := uuid.Parse(subject)
		require.NoError(t, err)

		authn, err := New(subject, ProviderMock, []string{}, map[string]any{})
		require.NoError(t, err)

		id, err := authn.ID()
		require.NoError(t, err)
		require.Equal(t, expectedID, id)
	})

	t.Run("IDはUUIDとして解釈できなかった場合にエラーを返す", func(t *testing.T) {
		t.Parallel()
		subject := "non-uuid-subject"

		authn, err := New(subject, ProviderMock, []string{}, map[string]any{})
		require.NoError(t, err)

		id, err := authn.ID()
		require.ErrorIs(t, err, ErrInvalidIDMissing)
		require.Equal(t, uuid.UUID{}, id)
	})
}

func TestAuthn_Provider(t *testing.T) {
	t.Parallel()

	t.Run("Providerはコンストラクタで与えたproviderを返す", func(t *testing.T) {
		t.Parallel()
		provider := ProviderMock
		authn, err := New("test-subject", provider, []string{}, map[string]any{})
		require.NoError(t, err)

		require.Equal(t, provider, authn.Provider())
	})
}

func TestAuthn_Scopes(t *testing.T) {
	t.Parallel()

	t.Run("Scopesはコンストラクタで与えたscopesを返す", func(t *testing.T) {
		t.Parallel()
		scopes := []string{"scope1", "scope2"}
		authn, err := New("test-subject", ProviderMock, scopes, map[string]any{})
		require.NoError(t, err)

		require.Equal(t, scopes, authn.Scopes())
	})
}

func TestAuthn_Claims(t *testing.T) {
	t.Parallel()

	t.Run("Claimsはコンストラクタで与えたclaimsを返す", func(t *testing.T) {
		t.Parallel()
		claims := map[string]any{"role": "admin", "exp": 1234567890}
		authn, err := New("test-subject", ProviderMock, []string{}, claims)
		require.NoError(t, err)

		require.Equal(t, claims, authn.Claims())
	})
}
