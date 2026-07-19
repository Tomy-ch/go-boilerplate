// Package testauth は、テスト用の認証ヘルパーを提供します。
package testauth

import (
	"context"
	"testing"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/require"
)

// MakeAvailableAuthn は、テスト用のコンテキストに認証情報を設定します。
// subject が UUID として解釈できる場合は内部 UserID も解決済みにします。
func MakeAvailableAuthn(ctx context.Context, t *testing.T, subject string) context.Context {
	t.Helper()
	authn, err := auth.New(
		subject,
		auth.IssuerMock,
		[]string{},
		map[string]any{},
	)
	require.NoError(t, err)

	if id, perr := uuid.Parse(subject); perr == nil {
		authn = authn.WithUserID(id)
	}

	ctx = ctxhelper.WithAuthn(ctx)
	require.True(t, ctxhelper.SetAuthn(ctx, *authn))
	return ctx
}
