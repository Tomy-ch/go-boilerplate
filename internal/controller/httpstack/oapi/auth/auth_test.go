package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	mock_auth "go-boilerplate/internal/usecase/boundary/auth/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// ctxKeyForTest は、バリデータが渡す context を識別するためのテスト用キーです。
type ctxKeyForTest struct{}

func TestNewAuthenticator(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トークン抽出からAuthenticate呼出しを経てスロットにAuthnがセットされる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			want, _ := authbd.New("user123", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
			mr.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(want, nil)

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer user123")
			req = req.WithContext(ctxhelper.WithAuthn(req.Context()))

			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.NoError(t, err)

			got, ok := ctxhelper.GetAuthn(req.Context())
			assert.True(t, ok)
			assert.Equal(t, want.Subject(), got.Subject())
		})

		t.Run("AuthenticateとResolveがバリデータの引数ではなくリクエストのcontextを受け取る", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			want, _ := authbd.New("user123", "mock", nil, nil)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer user123")
			req = req.WithContext(ctxhelper.WithAuthn(req.Context()))

			validatorCtx := context.WithValue(context.Background(), ctxKeyForTest{}, "validator")

			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, _ *authbd.Credential) (*authbd.Authn, error) {
					assert.Equal(t, req.Context(), ctx)
					assert.NotEqual(t, validatorCtx, ctx)
					return want, nil
				})
			mr.EXPECT().Resolve(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, _ *authbd.Authn) (*authbd.Authn, error) {
					assert.Equal(t, req.Context(), ctx)
					return want, nil
				})

			fn := NewAuthenticator(m, mr)

			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(validatorCtx, in)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Authnスロットが未仕込みの場合、ErrAuthnSlotNotFoundを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			want, _ := authbd.New("user123", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
			mr.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(want, nil)

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer user123")
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.ErrorIs(t, err, ErrAuthnSlotNotFound)
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusInternalServerError, he.Code)
		})

		t.Run("authenticatorが認証失敗を返すと401のErrUnauthorizedInvalidTokenを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, xerrors.Wrap(apperror.ErrUnauthenticated, "invalid token"))

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.ErrorIs(t, err, ErrUnauthorizedInvalidToken)
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusUnauthorized, he.Code)
		})

		t.Run("authenticatorが検証不能(鍵の取得不能)を返すと503を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, xerrors.Wrap(apperror.ErrUnavailable, "jwks unavailable"))

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusServiceUnavailable, he.Code)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("resolverがキャンセルを返すと499を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			want, _ := authbd.New("user123", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
			mr.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(nil, xerrors.Wrap(apperror.ErrCanceled, "client gone"))

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, 499, he.Code)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})

		t.Run("分類の無いエラーは401に潰さず500を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, xerrors.New("bad"))

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusInternalServerError, he.Code)
		})

		t.Run("認証情報が取得できない場合はErrUnauthorizedTokenNotProvidedを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.ErrorIs(t, err, ErrUnauthorizedTokenNotProvided)
		})

		t.Run("Authenticateがnil,nilを返す場合はErrUnauthorizedTokenNotProvidedを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, nil)

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.ErrorIs(t, err, ErrUnauthorizedTokenNotProvided)
		})

		t.Run("トークンが無効な場合、返すエラーと同じものをスロットへ記録する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, xerrors.Wrap(apperror.ErrUnauthenticated, "bad"))

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")
			req = req.WithContext(ctxhelper.WithAuthn(req.Context()))
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.ErrorIs(t, err, ErrUnauthorizedInvalidToken)
			assert.Equal(t, err, ctxhelper.AuthnFailure(req.Context()))
		})

		t.Run("identity解決がinfra障害の場合、503を持つエラーをスロットへ記録する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			authn, _ := authbd.New("user123", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(authn, nil)
			mr.EXPECT().Resolve(gomock.Any(), gomock.Any()).
				Return(nil, xerrors.Wrap(apperror.ErrUnavailable, "database is down"))

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")
			req = req.WithContext(ctxhelper.WithAuthn(req.Context()))
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.Error(t, err)

			recorded := ctxhelper.AuthnFailure(req.Context())
			require.Equal(t, err, recorded)

			var he *echo.HTTPError
			require.True(t, xerrors.As(recorded, &he))
			assert.Equal(t, http.StatusServiceUnavailable, he.Code)
		})

		t.Run("トークン未提示は失敗として記録しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)

			fn := NewAuthenticator(m, mr)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req = req.WithContext(ctxhelper.WithAuthn(req.Context()))
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.ErrorIs(t, err, ErrUnauthorizedTokenNotProvided)
			assert.NoError(t, ctxhelper.AuthnFailure(req.Context()))
		})
	})
}

func Test_authExtractor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Authenticateの結果を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			want, _ := authbd.New("subj", "mock", nil, nil)
			resolved, err := want.WithUserID(uuidtestkit.NewTestFromSalt(t, "resolved"))
			require.NoError(t, err)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
			mr.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(resolved, nil)
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			got, err := authExtractor(context.Background(), req, m, mr)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, want.Subject(), got.Subject())
			// authExtractor は Resolve の戻り値（UserID 解決済み）を返す。認証前の Authn ではないことを区別する。
			assert.True(t, got.HasUserID())
		})

		t.Run("トークンが空なら認証スキップとしてnil,nilを返す", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			authn, err := authExtractor(ctx, httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil), nil, nil)
			require.NoError(t, err)
			assert.Nil(t, authn)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Authenticateが認証失敗を返すとErrUnauthorizedInvalidTokenへ差し替える", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, xerrors.Wrap(apperror.ErrUnauthenticated, "invalid token"))
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			authn, err := authExtractor(context.Background(), req, m, mr)
			require.ErrorIs(t, err, ErrUnauthorizedInvalidToken)
			assert.Nil(t, authn)
		})

		t.Run("Authenticateが検証不能を返すと認証失敗へ寄せず分類を保つ", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, xerrors.Wrap(apperror.ErrUnavailable, "jwks unavailable"))
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			authn, err := authExtractor(context.Background(), req, m, mr)
			assert.Nil(t, authn)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.NotErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("Authenticateが分類の無いエラーを返すと認証失敗へ寄せずそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			bad := xerrors.New("bad")
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, bad)
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			authn, err := authExtractor(context.Background(), req, m, mr)
			assert.Nil(t, authn)
			require.ErrorIs(t, err, bad)
			require.NotErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("resolverがinfra障害を返すと分類を保ったまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			want, _ := authbd.New("subj", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
			mr.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(nil, xerrors.Wrap(apperror.ErrInternal, "db down"))
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			authn, err := authExtractor(context.Background(), req, m, mr)
			assert.Nil(t, authn)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("resolverが認証失敗(ErrUnauthenticated系)を返すと*echo.HTTPErrorにせずそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			want, _ := authbd.New("subj", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
			mr.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(nil, authbd.ErrIdentityNotFound)
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			authn, err := authExtractor(context.Background(), req, m, mr)
			assert.Nil(t, authn)
			require.ErrorIs(t, err, authbd.ErrIdentityNotFound)
			var he *echo.HTTPError
			assert.False(t, xerrors.As(err, &he))
		})

		t.Run("resolverがnil,nilを返す場合は防御的にErrIdentityNotFoundを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			want, _ := authbd.New("subj", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
			mr.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(nil, nil)
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			authn, err := authExtractor(context.Background(), req, m, mr)
			assert.Nil(t, authn)
			require.ErrorIs(t, err, authbd.ErrIdentityNotFound)
		})
	})
}

func Test_withHTTPStatus(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証失敗は401を付与し元エラーを保持する", func(t *testing.T) {
			t.Parallel()

			err := withHTTPStatus(ErrUnauthorizedInvalidToken)

			assert.Equal(t, http.StatusUnauthorized, echo.StatusCode(err))
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("キャンセルは499を付与し元エラーを保持する", func(t *testing.T) {
			t.Parallel()

			err := withHTTPStatus(xerrors.Wrap(apperror.ErrCanceled, "client gone"))

			assert.Equal(t, 499, echo.StatusCode(err))
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})

		t.Run("取得不能は503を付与し元エラーを保持する", func(t *testing.T) {
			t.Parallel()

			err := withHTTPStatus(xerrors.Wrap(apperror.ErrUnavailable, "jwks unavailable"))

			assert.Equal(t, http.StatusServiceUnavailable, echo.StatusCode(err))
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("分類の無いエラーは401へ寄せず500を付与する", func(t *testing.T) {
			t.Parallel()
			orig := xerrors.New("bad")

			err := withHTTPStatus(orig)

			assert.Equal(t, http.StatusInternalServerError, echo.StatusCode(err))
			require.ErrorIs(t, err, orig)
		})

		t.Run("既にステータスを持つエラーは変換せずそのまま返る", func(t *testing.T) {
			t.Parallel()

			orig := echo.NewHTTPError(http.StatusServiceUnavailable, "").Wrap(xerrors.Wrap(apperror.ErrUnavailable, "db unavailable"))

			err := withHTTPStatus(orig)

			assert.Same(t, orig, err)
			assert.Equal(t, http.StatusServiceUnavailable, echo.StatusCode(err))
		})
	})
}

func Test_extractBearerToken(t *testing.T) {
	t.Parallel()

	newReq := func(t *testing.T) *http.Request {
		t.Helper()
		return httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Authorization: Bearer 形式ならトークン部分とBearerスキームを抽出する", func(t *testing.T) {
			t.Parallel()
			req := newReq(t)
			req.Header.Set("Authorization", "Bearer abcdef")
			scheme, tok := extractBearerToken(req)
			assert.Equal(t, authbd.SchemeBearer, scheme)
			assert.Equal(t, "abcdef", tok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Authorizationヘッダが未設定なら空を返す", func(t *testing.T) {
			t.Parallel()
			scheme, tok := extractBearerToken(newReq(t))
			assert.Empty(t, scheme)
			assert.Empty(t, tok)
		})

		t.Run("Bearerプレフィックスが無ければ空を返す", func(t *testing.T) {
			t.Parallel()
			req := newReq(t)
			req.Header.Set("Authorization", "Token abcdef")
			scheme, tok := extractBearerToken(req)
			assert.Empty(t, scheme)
			assert.Empty(t, tok)
		})
	})
}
