package testauth

import (
	"context"
	"testing"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeAvailableAuthn(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("subjectとproviderが認証情報としてコンテキストに設定される", func(t *testing.T) {
			t.Parallel()

			subject := "test-user-id"
			authCtx := MakeAvailableAuthn(context.Background(), t, subject)

			authn, ok := ctxhelper.GetAuthn(authCtx)
			assert.True(t, ok)
			assert.Equal(t, subject, authn.Subject())
			assert.Equal(t, auth.IssuerMock, authn.Issuer())
		})

		t.Run("subjectがUUIDとして解釈できない場合、内部UserIDは未解決のままになる", func(t *testing.T) {
			t.Parallel()

			authCtx := MakeAvailableAuthn(context.Background(), t, "not-a-uuid")

			authn, ok := ctxhelper.GetAuthn(authCtx)
			require.True(t, ok)
			assert.False(t, authn.HasUserID())
		})

		t.Run("subjectがUUIDとして解釈できる場合、内部UserIDも解決済みになる", func(t *testing.T) {
			t.Parallel()

			expectedID := uuid.NewTestFromSalt(t, "make_available_authn")
			authCtx := MakeAvailableAuthn(context.Background(), t, expectedID.String())

			authn, ok := ctxhelper.GetAuthn(authCtx)
			require.True(t, ok)
			require.True(t, authn.HasUserID())
			gotID, err := authn.UserID()
			require.NoError(t, err)
			assert.Equal(t, expectedID, gotID)
		})
	})
}
