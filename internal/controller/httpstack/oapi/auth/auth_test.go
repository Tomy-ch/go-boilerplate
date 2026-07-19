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
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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
		})

		t.Run("authenticatorがエラーを返すとErrUnauthorizedInvalidTokenを返す", func(t *testing.T) {
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
			require.ErrorIs(t, err, ErrUnauthorizedInvalidToken)
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
			resolved := want.WithUserID(uuid.NewTestFromSalt(t, "resolved"))
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

		t.Run("AuthenticateがエラーならErrUnauthorizedInvalidTokenを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, xerrors.New("bad"))
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			authn, err := authExtractor(context.Background(), req, m, mr)
			require.ErrorIs(t, err, ErrUnauthorizedInvalidToken)
			assert.Nil(t, authn)
		})

		t.Run("resolverがinfra障害(ErrInternal)を返すと500の*echo.HTTPErrorを返す", func(t *testing.T) {
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
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusInternalServerError, he.Code)
			// 元エラーが Internal に保持され、分類（ErrInternal）が透過することを確認する。
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("resolverがErrUnavailableを返すと503の*echo.HTTPErrorを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			mr := mock_auth.NewMockIdentityResolver(ctrl)
			want, _ := authbd.New("subj", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
			mr.EXPECT().Resolve(gomock.Any(), gomock.Any()).Return(nil, xerrors.Wrap(apperror.ErrUnavailable, "db unavailable"))
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			authn, err := authExtractor(context.Background(), req, m, mr)
			assert.Nil(t, authn)
			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusServiceUnavailable, he.Code)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
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

func Test_infraErrorToHTTP(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ErrUnavailableは503の*echo.HTTPErrorへ変換され元エラーを保持する", func(t *testing.T) {
			t.Parallel()
			orig := xerrors.Wrap(apperror.ErrUnavailable, "db unavailable")

			err := infraErrorToHTTP(orig)

			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusServiceUnavailable, he.Code)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("ErrUnavailable以外は500の*echo.HTTPErrorへ変換され元エラーを保持する", func(t *testing.T) {
			t.Parallel()
			orig := xerrors.Wrap(apperror.ErrInternal, "db down")

			err := infraErrorToHTTP(orig)

			var he *echo.HTTPError
			require.ErrorAs(t, err, &he)
			assert.Equal(t, http.StatusInternalServerError, he.Code)
			require.ErrorIs(t, err, apperror.ErrInternal)
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
