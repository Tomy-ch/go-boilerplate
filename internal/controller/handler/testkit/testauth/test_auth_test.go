package testauth

import (
	"context"
	"testing"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/usecase/boundary/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeAvailableAuthn(t *testing.T) {
	t.Parallel()

	t.Run("正常系_subjectとproviderが認証情報としてコンテキストに設定される", func(t *testing.T) {
		t.Parallel()

		subject := "test-user-id"
		authCtx := MakeAvailableAuthn(context.Background(), t, subject)

		authn, ok := ctxhelper.GetAuthn(authCtx)
		require.True(t, ok)
		assert.Equal(t, subject, authn.Subject())
		assert.Equal(t, auth.ProviderMock, authn.Provider())
	})
}
