package testauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/controller/ctxhelper"
)

func TestMakeAvailableAuthn(t *testing.T) {
	t.Parallel()

	subject := "test-user-id"
	ctx := context.Background()
	authCtx := MakeAvailableAuthn(ctx, t, subject)

	authn, ok := ctxhelper.GetAuthn(authCtx)
	assert.True(t, ok)
	assert.Equal(t, subject, authn.Subject())
}
