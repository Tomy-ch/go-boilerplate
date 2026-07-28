package outbound

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDetailPolicy は、DI 配線テスト用の DetailPolicy スタブです。
type stubDetailPolicy struct{}

func (stubDetailPolicy) Allows(*http.Request) bool { return false }

func Test_provideErrorHandlerServeConfig(t *testing.T) {
	t.Parallel()

	log := logging.NewTestLogger(t)
	obsCfg := config.NewObservabilityConfig(config.MockConfigForTest(t))
	lf := logging.NewTestLogFieldBuilder(t)

	out := provideErrorHandlerServeConfig(stubDetailPolicy{}, log, lf, obsCfg)
	require.NotNil(t, out.SrvCfg.Config)
	assert.Equal(t, "errorhandler", out.SrvCfg.Name)

	e := &echo.Echo{}
	out.SrvCfg.Config(e)
	assert.NotNil(t, e.HTTPErrorHandler)
}

func Test_provideDetailPolicy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解析可能な spec から DetailPolicy を構築して返す", func(t *testing.T) {
			t.Parallel()

			spec := &openapi3.T{Paths: openapi3.NewPaths()}
			spec.Paths.Set("/v1/things", &openapi3.PathItem{Get: openapi3.NewOperation()})

			policy, err := provideDetailPolicy(spec)

			require.NoError(t, err)
			assert.NotNil(t, policy)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("router 構築に失敗する spec ではエラーを伝播し fx の起動を中断させる", func(t *testing.T) {
			t.Parallel()

			// gorilla/mux がコンパイルできない正規表現をパス変数に含む spec は router 構築に失敗する。
			spec := &openapi3.T{Paths: openapi3.NewPaths()}
			spec.Paths.Set("/{id:(}", &openapi3.PathItem{Get: openapi3.NewOperation()})

			policy, err := provideDetailPolicy(spec)

			require.Error(t, err)
			assert.Nil(t, policy)
		})
	})
}
