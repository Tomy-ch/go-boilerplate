package outbound

import (
	"net/http"
	"net/http/httptest"
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

// stubAllowPolicy は、DI 配線テスト用の AllowPolicy スタブです。
type stubAllowPolicy struct{}

func (stubDetailPolicy) Allows(*http.Request) bool { return false }

func (stubAllowPolicy) Allow(*http.Request) string { return "" }

func Test_provideErrorHandlerServeConfig(t *testing.T) {
	t.Parallel()

	log := logging.NewTestLogger(t)
	obsCfg := config.NewObservabilityConfig(config.MockConfigForTest(t))
	lf := logging.NewTestLogFieldBuilder(t)

	out := provideErrorHandlerServeConfig(stubDetailPolicy{}, stubAllowPolicy{}, log, lf, obsCfg)
	require.NotNil(t, out.SrvCfg.Config)
	assert.Equal(t, "errorhandler", out.SrvCfg.Name)

	e := &echo.Echo{}
	out.SrvCfg.Config(e)
	assert.NotNil(t, e.HTTPErrorHandler)
}

// newDetailOptInSpec は、GET /probe の 400 レスポンスだけが details を opt-in した最小 spec を返します。
func newDetailOptInSpec(t *testing.T) *openapi3.T {
	t.Helper()

	op := openapi3.NewOperation()
	op.OperationID = "Probe"
	op.Responses = openapi3.NewResponses(openapi3.WithStatus(http.StatusBadRequest, &openapi3.ResponseRef{
		Value: openapi3.NewResponse().WithJSONSchema(
			openapi3.NewObjectSchema().WithProperty("details", openapi3.NewStringSchema()),
		),
	}))

	spec := &openapi3.T{Paths: openapi3.NewPaths()}
	spec.Paths.Set("/probe", &openapi3.PathItem{Get: op})
	return spec
}

func Test_provideDetailPolicy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引数の spec から構築した DetailPolicy を返す", func(t *testing.T) {
			t.Parallel()
			policy, err := provideDetailPolicy(newDetailOptInSpec(t))
			require.NoError(t, err)
			require.NotNil(t, policy)

			// /probe は実 spec に存在しないため、引数を無視して別の spec を使う実装では false になる。
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/probe", nil)
			assert.True(t, policy.Allows(req))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("router 構築に失敗する spec の場合、エラーを握り潰さず返す", func(t *testing.T) {
			t.Parallel()
			spec := &openapi3.T{Paths: openapi3.NewPaths()}
			spec.Paths.Set("/{id:(}", &openapi3.PathItem{Get: openapi3.NewOperation()})

			policy, err := provideDetailPolicy(spec)
			require.Error(t, err)
			assert.Nil(t, policy)
		})
	})
}

func Test_provideAllowPolicy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引数の spec から構築した AllowPolicy を返す", func(t *testing.T) {
			t.Parallel()
			policy, err := provideAllowPolicy(newDetailOptInSpec(t))
			require.NoError(t, err)
			require.NotNil(t, policy)

			// /probe は実 spec に存在しないため、引数を無視して別の spec を使う実装では空文字になる。
			req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/probe", nil)
			assert.Equal(t, "OPTIONS, GET", policy.Allow(req))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("router 構築に失敗する spec の場合、エラーを握り潰さず返す", func(t *testing.T) {
			t.Parallel()
			spec := &openapi3.T{Paths: openapi3.NewPaths()}
			spec.Paths.Set("/{id:(}", &openapi3.PathItem{Get: openapi3.NewOperation()})

			policy, err := provideAllowPolicy(spec)
			require.Error(t, err)
			assert.Nil(t, policy)
		})
	})
}
