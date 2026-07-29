package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/handler/health"
	"go-boilerplate/internal/controller/httpstack/cookie"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/instrumentation"
	"go-boilerplate/internal/di/server/extension/security"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRequestIDStackServer は、requestId を読み戻す 2 つの消費者（errorhandler と logging の
// res.After フック）を含む requestid / logging / secure_cookie を、本番と同じ provider・
// 同じ優先度解決で適用した Echo を起動する。requestId はミドルウェアの適用順に依存するため、
// 順序を手書きせず DI 側の Priority をそのまま経路に載せる。
// logger は logging ミドルウェアへ渡すもので、アクセスログの内容を検証する場合に観測可能な
// Logger を指定する。
func newRequestIDStackServer(t *testing.T, logger logging.Logger) *Server {
	t.Helper()

	e := echo.New()
	UseAppErrorHandler(t, e)

	secCookie := cookie.NewSecurityCookie(config.NewSecureCookieConfig(config.MockConfigForTest(t)))
	require.NoError(t, extension.ApplyUseMiddlewares(e, logging.NewTestLogger(t), []extension.UseMiddleware{
		instrumentation.RequestIDMiddleware().Middleware,
		instrumentation.LoggingMiddleware(logger, logging.NewTestLogFieldBuilder(t)).Middleware,
		security.CookieMiddleware(secCookie).Middleware,
	}))

	health.BindHandler(e, observability.NewNoopTracerFactory(t))

	return StartServer(t, e)
}

func TestErrorRequestID_Integration(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エラーボディのrequestIdがX-Request-Idヘッダーと一致する", func(t *testing.T) {
			t.Parallel()

			headers := http.Header{}
			headers.Set(echo.HeaderXRequestID, "integration-rid-749")

			srv := newRequestIDStackServer(t, logging.NewTestLogger(t))
			actual := srv.DoJSON(http.MethodGet, "/no-such-path", nil, headers)

			errResp := AssertErrorResponseBody(t, actual, http.StatusNotFound)
			assert.Equal(t, "integration-rid-749", actual.Header.Get(echo.HeaderXRequestID))
			assert.Equal(t, "integration-rid-749", errResp.RequestId)
		})

		t.Run("クライアントがX-Request-Idを送らない場合も生成された値がエラーボディに載る", func(t *testing.T) {
			t.Parallel()

			srv := newRequestIDStackServer(t, logging.NewTestLogger(t))
			actual := srv.DoJSON(http.MethodGet, "/no-such-path", nil, nil)

			errResp := AssertErrorResponseBody(t, actual, http.StatusNotFound)
			assert.NotEmpty(t, errResp.RequestId)
			assert.Equal(t, actual.Header.Get(echo.HeaderXRequestID), errResp.RequestId)
		})

		t.Run("アクセスログのrequest_idがX-Request-Idヘッダーと一致する", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)

			headers := http.Header{}
			headers.Set(echo.HeaderXRequestID, "integration-rid-749-log")

			srv := newRequestIDStackServer(t, logger)
			actual := srv.DoJSON(http.MethodGet, "/no-such-path", nil, headers)
			AssertErrorResponseBody(t, actual, http.StatusNotFound)

			entries := observed.FilterMessage("request handled").All()
			require.Len(t, entries, 1)
			assert.Equal(t, "integration-rid-749-log", entries[0].ContextMap()[logging.RequestIDKey])
		})
	})
}
