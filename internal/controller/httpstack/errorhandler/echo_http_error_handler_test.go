package errorhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-boilerplate/internal/controller/error/response"
	"go-boilerplate/internal/controller/httpstack/oapi"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	"go-boilerplate/pkg/xerrors"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_normalizeEchoHTTPError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("EchoHTTPErrorでステータス範囲内の場合、NewHTTPErrorFromStatusベースのレスポンスが返る", func(t *testing.T) {
			t.Parallel()

			inner := xerrors.New("inner failure")
			ehe := echo.NewHTTPError(http.StatusForbidden, "").Wrap(inner)

			actual := normalizeEchoHTTPError(ehe)
			require.NotNil(t, actual)

			expectedBase := response.NewHTTPErrorFromStatus(http.StatusForbidden, nil)
			assert.Equal(t, expectedBase.Code, actual.Code)
			assert.Equal(t, expectedBase.Message, actual.Message)
			assert.Equal(t, expectedBase.HTTPStatus, actual.HTTPStatus)
			assert.Nil(t, actual.Details)
			require.Error(t, actual.Internal)
			assert.Contains(t, actual.Internal.Error(), "inner failure")
			assert.Contains(t, actual.Internal.Error(), "echo HTTP error")
		})

		t.Run("detailsを渡した場合、Detailsにセットされる", func(t *testing.T) {
			t.Parallel()

			inner := xerrors.New("inner2")
			ehe := echo.NewHTTPError(http.StatusConflict, "").Wrap(inner)

			actual := normalizeEchoHTTPError(ehe, "d1", "d2")
			require.NotNil(t, actual)

			expectedBase := response.NewHTTPErrorFromStatus(http.StatusConflict, nil)
			assert.Equal(t, expectedBase.Code, actual.Code)
			assert.Equal(t, expectedBase.Message, actual.Message)
			assert.Equal(t, expectedBase.HTTPStatus, actual.HTTPStatus)
			require.NotNil(t, actual.Details)
			assert.Equal(t, []string{"d1", "d2"}, *actual.Details)
			require.Error(t, actual.Internal)
			assert.Contains(t, actual.Internal.Error(), "inner2")
		})

		t.Run("Echoの定義済みエラーの場合、ステータスが解決されレスポンスが返る", func(t *testing.T) {
			t.Parallel()

			actual := normalizeEchoHTTPError(echo.ErrNotFound)
			require.NotNil(t, actual)

			expectedBase := response.NewHTTPErrorFromStatus(http.StatusNotFound, nil)
			assert.Equal(t, expectedBase.Code, actual.Code)
			assert.Equal(t, expectedBase.HTTPStatus, actual.HTTPStatus)
		})

		t.Run("メソッド不許可の場合、405とMETHOD_NOT_ALLOWEDが返る", func(t *testing.T) {
			t.Parallel()

			actual := normalizeEchoHTTPError(echo.ErrMethodNotAllowed)
			require.NotNil(t, actual)

			assert.Equal(t, http.StatusMethodNotAllowed, actual.HTTPStatus)
			assert.Equal(t, "METHOD_NOT_ALLOWED", actual.Code)
		})

		t.Run("OpenAPIバリデーション失敗の場合、ミドルウェアが決めた400が解決される", func(t *testing.T) {
			t.Parallel()

			spec, err := validator.GetValidator()
			require.NoError(t, err)
			skipper := func(*echo.Context) bool { return false }
			authFunc := func(context.Context, *openapi3filter.AuthenticationInput) error { return nil }
			mw := oapi.Middleware(spec, skipper, authFunc)

			req := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				"/v1/users",
				strings.NewReader("{}"),
			)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			c := echo.New().NewContext(req, httptest.NewRecorder())

			validationErr := mw(func(*echo.Context) error { return nil })(c)
			require.Error(t, validationErr)

			actual := normalizeEchoHTTPError(validationErr)
			require.NotNil(t, actual)

			expectedBase := response.NewHTTPErrorFromStatus(http.StatusBadRequest, nil)
			assert.Equal(t, expectedBase.HTTPStatus, actual.HTTPStatus)
			assert.Equal(t, expectedBase.Code, actual.Code)
			require.Error(t, actual.Internal)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非EchoHTTPErrorの場合、nilが返る", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, normalizeEchoHTTPError(xerrors.New("just an error")))
		})

		t.Run("EchoHTTPErrorだがステータス範囲外の場合、nilが返る", func(t *testing.T) {
			t.Parallel()
			ehe := echo.NewHTTPError(http.StatusContinue, "")
			assert.Nil(t, normalizeEchoHTTPError(ehe))
		})
	})
}

func newAllowTestContext(t *testing.T, echoAllow any) *echo.Context {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/t", nil)
	c := echo.New().NewContext(req, httptest.NewRecorder())
	if echoAllow != nil {
		c.Set(echo.ContextKeyHeaderAllow, echoAllow)
	}
	return c
}

func Test_setAllowHeader(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("405でEchoのルータが許可メソッドを解決済みの場合、その値がAllowヘッダーになる", func(t *testing.T) {
			t.Parallel()

			c := newAllowTestContext(t, "OPTIONS, GET")
			setAllowHeader(c, stubAllowPolicy{allow: "OPTIONS, POST"}, http.StatusMethodNotAllowed)

			assert.Equal(t, "OPTIONS, GET", c.Response().Header().Get(echo.HeaderAllow))
		})

		t.Run("Echoのルータが解決していない場合、specから解決した値がAllowヘッダーになる", func(t *testing.T) {
			t.Parallel()

			c := newAllowTestContext(t, nil)
			setAllowHeader(c, stubAllowPolicy{allow: "OPTIONS, GET"}, http.StatusMethodNotAllowed)

			assert.Equal(t, "OPTIONS, GET", c.Response().Header().Get(echo.HeaderAllow))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("405以外のステータスの場合、Allowヘッダーは付かない", func(t *testing.T) {
			t.Parallel()

			c := newAllowTestContext(t, "OPTIONS, GET")
			setAllowHeader(c, stubAllowPolicy{allow: "OPTIONS, GET"}, http.StatusNotFound)

			assert.Empty(t, c.Response().Header().Get(echo.HeaderAllow))
		})

		t.Run("どちらの情報源からも解決できない場合、Allowヘッダーは付かない", func(t *testing.T) {
			t.Parallel()

			c := newAllowTestContext(t, nil)
			setAllowHeader(c, stubAllowPolicy{}, http.StatusMethodNotAllowed)

			assert.Empty(t, c.Response().Header().Get(echo.HeaderAllow))
		})
	})
}

func Test_allowFromEchoRouter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ルータが解決した許可メソッドが返る", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "OPTIONS, GET", allowFromEchoRouter(newAllowTestContext(t, "OPTIONS, GET")))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ルータが許可メソッドを解決していない場合、空文字が返る", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, allowFromEchoRouter(newAllowTestContext(t, nil)))
		})

		t.Run("許可メソッドが空文字の場合、空文字が返る", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, allowFromEchoRouter(newAllowTestContext(t, "")))
		})

		t.Run("許可メソッドが文字列でない場合、空文字が返る", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, allowFromEchoRouter(newAllowTestContext(t, 405)))
		})
	})
}
