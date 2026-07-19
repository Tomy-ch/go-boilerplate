package module

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/infrastructure/httpclient"
)

// authModule は jwks の Downstream プロファイルと required downstream を value group へ寄与する。
// registry は起動時に required に対応する profile の充足を検証するため、fx.ValidateApp（結線のみ）では
// この寄与を検証できない。実アプリを起動し、jwks required が authModule の profile で充足されて
// httpclient.Client が構築されることを確認する。
//
//nolint:paralleltest // httpclient_test と同様、fx アプリ起動は global registerer 競合回避のため非並列
func Test_authModule_ProvidesJWKSProfile(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("jwks required に対応する profile が authModule で揃い Client が起動する", func(t *testing.T) {
			var client httpclient.Client
			app := newHTTPClientTestApp(t, authModule(), fx.Populate(&client))

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, client)
		})
	})
}
