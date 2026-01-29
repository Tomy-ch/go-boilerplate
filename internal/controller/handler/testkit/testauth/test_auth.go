// Package testauth は、テスト用の認証ヘルパーを提供します。
package testauth

import (
	"context"
	"testing"

	"boilerplate-go/internal/controller/ctxhelper"
	"boilerplate-go/internal/usecase/boundary/auth"

	"github.com/stretchr/testify/require"
)

// MakeAvailableAuthn は、テスト用のコンテキストに認証情報を設定します。
//
// この関数は、指定されたユーザーIDを持つ認証情報を作成し、テスト用のコンテキストに設定します。
func MakeAvailableAuthn(ctx context.Context, t *testing.T, subject string) context.Context {
	t.Helper()
	authn, err := auth.New(
		subject,
		"local-mock",
		nil,
		nil,
	)
	require.NoError(t, err)

	return ctxhelper.SetAuthn(ctx, *authn)
}
