package testauth

import (
	"context"
	"testing"

	"boilerplate-go/internal/controller/ctxhelper"

	"github.com/stretchr/testify/require"
)

func TestMakeAvailableAuthn(t *testing.T) {
	t.Parallel()

	subject := "test-user-id"
	ctx := context.Background()
	authCtx := MakeAvailableAuthn(ctx, t, subject)

	authn, ok := ctxhelper.GetAuthn(authCtx)
	require.True(t, ok)
	require.Equal(t, subject, authn.Subject())
}
