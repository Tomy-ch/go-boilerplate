package oapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/httpstack/oapi/auth"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	mock_auth "go-boilerplate/internal/usecase/boundary/auth/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// testSpec は、security の 3 つの形（完全保護 / 完全公開 / 任意認証）を 1 本ずつ持つ spec。
// 任意認証は本番 spec にまだ存在しないため、ここで宣言して実際のバリデーションパイプラインへ通す。
const testSpec = `
openapi: 3.0.3
info:
  title: fail-closed test
  version: 1.0.0
paths:
  /protected:
    get:
      operationId: getProtected
      security:
        - BearerAuth: []
      responses:
        "200":
          description: ok
  /public:
    get:
      operationId: getPublic
      security: []
      responses:
        "200":
          description: ok
  /optional:
    get:
      operationId: getOptional
      security:
        - BearerAuth: []
        - {}
      responses:
        "200":
          description: ok
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
`

// authnStub は、Authenticate / Resolve の結果を宣言してから本物の authFunc を組み立てるための指定。
type authnStub struct {
	authenticateErr error
	resolveErr      error
}

func newTestAuthFunc(t *testing.T, stub authnStub) openapi3filter.AuthenticationFunc {
	t.Helper()

	ctrl := gomock.NewController(t)
	authenticator := mock_auth.NewMockAuthenticator(ctrl)
	resolver := mock_auth.NewMockIdentityResolver(ctrl)

	authn, err := authbd.New("user-john-doe", authbd.IssuerMock, nil, nil)
	require.NoError(t, err)

	switch {
	case stub.authenticateErr != nil:
		authenticator.EXPECT().Authenticate(gomock.Any(), gomock.Any()).
			Return(nil, stub.authenticateErr).AnyTimes()
	case stub.resolveErr != nil:
		authenticator.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(authn, nil).AnyTimes()
		resolver.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(nil, stub.resolveErr).AnyTimes()
	default:
		authenticator.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(authn, nil).AnyTimes()
		resolver.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(authn, nil).AnyTimes()
	}

	return auth.NewAuthenticator(authenticator, resolver)
}

// serveTestSpec は、合成 spec のミドルウェアへ 1 リクエスト通し、ステータスとハンドラ到達の有無を返す。
func serveTestSpec(t *testing.T, stub authnStub, path, authorization string) (int, bool) {
	t.Helper()

	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromData([]byte(testSpec))
	require.NoError(t, err)
	require.NoError(t, spec.Validate(context.Background()))

	e := echo.New()
	mw := Middleware(spec, nil, newTestAuthFunc(t, stub))

	reached := false
	handler := mw(func(c *echo.Context) error {
		reached = true
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	if authorization != "" {
		req.Header.Set(echo.HeaderAuthorization, authorization)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath(path)

	if herr := handler(c); herr != nil {
		var he *echo.HTTPError
		if xerrors.As(herr, &he) {
			return he.Code, reached
		}
		return http.StatusInternalServerError, reached
	}
	return rec.Code, reached
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Authnスロットが仕込まれている", func(t *testing.T) {
			t.Parallel()

			spec := &openapi3.T{}
			mw := Middleware(spec, nil, nil)

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e := echo.New()
			c := e.NewContext(req, rec)

			handler := mw(func(_ *echo.Context) error {
				return nil
			})

			_ = handler(c)

			// ミドルウェアが Authn スロットを仕込んでいることを確認（Set が成功し Get で読める）。
			a, err := authbd.New("u1", authbd.IssuerMock, nil, nil)
			require.NoError(t, err)
			require.True(t, ctxhelper.SetAuthn(c.Request().Context(), *a))

			got, ok := ctxhelper.GetAuthn(c.Request().Context())
			assert.True(t, ok)
			assert.Equal(t, a.Subject(), got.Subject())
		})

		t.Run("完全公開のoperationは認証を要求しない", func(t *testing.T) {
			t.Parallel()

			status, reached := serveTestSpec(t, authnStub{}, "/public", "")
			assert.Equal(t, http.StatusOK, status)
			assert.True(t, reached)
		})

		t.Run("完全公開のoperationは無効なトークンを付けても素通りする", func(t *testing.T) {
			t.Parallel()

			stub := authnStub{authenticateErr: xerrors.New("expired")}
			status, reached := serveTestSpec(t, stub, "/public", "Bearer broken")
			assert.Equal(t, http.StatusOK, status)
			assert.True(t, reached)
		})

		t.Run("完全保護のoperationは有効なトークンで到達する", func(t *testing.T) {
			t.Parallel()

			status, reached := serveTestSpec(t, authnStub{}, "/protected", "Bearer valid")
			assert.Equal(t, http.StatusOK, status)
			assert.True(t, reached)
		})

		t.Run("任意認証のoperationはトークンが無ければゲストとして到達する", func(t *testing.T) {
			t.Parallel()

			status, reached := serveTestSpec(t, authnStub{}, "/optional", "")
			assert.Equal(t, http.StatusOK, status)
			assert.True(t, reached)
		})

		t.Run("任意認証のoperationは有効なトークンで到達する", func(t *testing.T) {
			t.Parallel()

			status, reached := serveTestSpec(t, authnStub{}, "/optional", "Bearer valid")
			assert.Equal(t, http.StatusOK, status)
			assert.True(t, reached)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("完全保護のoperationは無効なトークンを401で拒否する", func(t *testing.T) {
			t.Parallel()

			stub := authnStub{authenticateErr: xerrors.New("expired")}
			status, reached := serveTestSpec(t, stub, "/protected", "Bearer broken")
			assert.Equal(t, http.StatusUnauthorized, status)
			assert.False(t, reached)
		})

		t.Run("任意認証のoperationは無効なトークンを401で拒否する", func(t *testing.T) {
			t.Parallel()

			stub := authnStub{authenticateErr: xerrors.New("expired")}
			status, reached := serveTestSpec(t, stub, "/optional", "Bearer broken")
			assert.Equal(t, http.StatusUnauthorized, status)
			assert.False(t, reached)
		})

		t.Run("任意認証のoperationはidentity解決の利用不能を503で返す", func(t *testing.T) {
			t.Parallel()

			stub := authnStub{resolveErr: xerrors.Wrap(apperror.ErrUnavailable, "database is down")}
			status, reached := serveTestSpec(t, stub, "/optional", "Bearer valid")
			assert.Equal(t, http.StatusServiceUnavailable, status)
			assert.False(t, reached)
		})

		t.Run("任意認証のoperationはidentity解決のその他の障害を500で返す", func(t *testing.T) {
			t.Parallel()

			stub := authnStub{resolveErr: xerrors.Wrap(apperror.ErrInternal, "unexpected")}
			status, reached := serveTestSpec(t, stub, "/optional", "Bearer valid")
			assert.Equal(t, http.StatusInternalServerError, status)
			assert.False(t, reached)
		})

		t.Run("任意認証のoperationはidentityが見つからない場合を401で拒否する", func(t *testing.T) {
			t.Parallel()

			stub := authnStub{resolveErr: xerrors.Wrap(apperror.ErrUnauthenticated, "identity not found")}
			status, reached := serveTestSpec(t, stub, "/optional", "Bearer valid")
			assert.Equal(t, http.StatusUnauthorized, status)
			assert.False(t, reached)
		})
	})
}

func Test_failClosed(t *testing.T) {
	t.Parallel()

	newContext := func(t *testing.T, ctx context.Context) *echo.Context {
		t.Helper()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		return echo.New().NewContext(req, httptest.NewRecorder())
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗が記録されていなければハンドラを呼ぶ", func(t *testing.T) {
			t.Parallel()

			called := false
			handler := failClosed(func(_ *echo.Context) error {
				called = true
				return nil
			})

			require.NoError(t, handler(newContext(t, ctxhelper.WithAuthn(context.Background()))))
			assert.True(t, called)
		})

		t.Run("スロット自体が無ければハンドラを呼ぶ", func(t *testing.T) {
			t.Parallel()

			called := false
			handler := failClosed(func(_ *echo.Context) error {
				called = true
				return nil
			})

			require.NoError(t, handler(newContext(t, context.Background())))
			assert.True(t, called)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗が記録されていればハンドラを呼ばずその失敗を返す", func(t *testing.T) {
			t.Parallel()

			want := xerrors.New("authentication failed")
			ctx := ctxhelper.WithAuthn(context.Background())
			require.True(t, ctxhelper.SetAuthnFailure(ctx, want))

			called := false
			handler := failClosed(func(_ *echo.Context) error {
				called = true
				return nil
			})

			require.ErrorIs(t, handler(newContext(t, ctx)), want)
			assert.False(t, called)
		})
	})
}
