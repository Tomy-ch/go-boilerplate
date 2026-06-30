package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/ctxhelper"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	mock_auth "go-boilerplate/internal/usecase/boundary/auth/mock"

	"github.com/getkin/kin-openapi/openapi3filter"
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
			cfg := config.MockConfigForTest(t)
			ac := config.NewAuthConfig(cfg)

			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			want, _ := authbd.New("user123", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)

			fn := NewAuthenticator(ac, m)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			//nolint:gosec // G124: テスト用のリクエストクッキー
			req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "user123"})
			req = req.WithContext(ctxhelper.WithAuthn(req.Context()))

			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.NoError(t, err)

			got, ok := ctxhelper.GetAuthn(req.Context())
			require.True(t, ok)
			assert.Equal(t, want.Subject(), got.Subject())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Authnスロットが未仕込みの場合、ErrAuthnSlotNotFoundを返す", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			ac := config.NewAuthConfig(cfg)

			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			want, _ := authbd.New("user123", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)

			fn := NewAuthenticator(ac, m)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			//nolint:gosec // G124: テスト用のリクエストクッキー
			req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "user123"})
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.ErrorIs(t, err, ErrAuthnSlotNotFound)
		})

		t.Run("authenticatorがエラーを返すとErrUnauthorizedInvalidTokenを返す", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			ac := config.NewAuthConfig(cfg)

			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, errors.New("bad"))

			fn := NewAuthenticator(ac, m)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			//nolint:gosec // G124: テスト用のリクエストクッキー
			req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "tok"})
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.ErrorIs(t, err, ErrUnauthorizedInvalidToken)
		})

		t.Run("認証情報が取得できない場合はErrUnauthorizedTokenNotProvidedを返す", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			ac := config.NewAuthConfig(cfg)

			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)

			fn := NewAuthenticator(ac, m)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.ErrorIs(t, err, ErrUnauthorizedTokenNotProvided)
		})

		t.Run("Authenticateがnil,nilを返す場合はErrUnauthorizedTokenNotProvidedを返す", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			ac := config.NewAuthConfig(cfg)

			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, nil)

			fn := NewAuthenticator(ac, m)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			//nolint:gosec // G124: テスト用のリクエストクッキー
			req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "tok"})
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
			cfg := config.MockConfigForTest(t)
			ac := config.NewAuthConfig(cfg)

			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			want, _ := authbd.New("subj", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			//nolint:gosec // G124: テスト用のリクエストクッキー
			req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "tok"})

			got, err := authExtractor(context.Background(), req, ac, m)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, want.Subject(), got.Subject())
		})

		t.Run("トークンが空なら認証スキップとしてnil,nilを返す", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			ac := config.NewAuthConfig(cfg)
			ctx := context.Background()

			authn, err := authExtractor(ctx, httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil), ac, nil)
			require.NoError(t, err)
			assert.Nil(t, authn)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("AuthenticateがエラーならErrUnauthorizedInvalidTokenを返す", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			ac := config.NewAuthConfig(cfg)

			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, errors.New("bad"))
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			//nolint:gosec // G124: テスト用のリクエストクッキー
			req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "tok"})

			authn, err := authExtractor(context.Background(), req, ac, m)
			require.ErrorIs(t, err, ErrUnauthorizedInvalidToken)
			assert.Nil(t, authn)
		})
	})
}

func Test_extractToken(t *testing.T) {
	t.Parallel()

	// 各サブテストは ac.SetHeaderName / SetAllowedHeaderBearer で
	// MockConfig 内部状態を書き換えるため、サブテストごとに専用の cfg/ac を生成する。
	newAuthConfig := func(t *testing.T) *config.AuthConfig {
		t.Helper()
		return config.NewAuthConfig(config.MockConfigForTest(t))
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Cookieから抽出できる", func(t *testing.T) {
			t.Parallel()
			ac := newAuthConfig(t)
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			//nolint:gosec // G124: テスト用のリクエストクッキー
			req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "cookieTok"})
			tok := extractToken(req, ac)
			assert.Equal(t, "cookieTok", tok)
		})

		t.Run("CookieとHeaderが両方ある場合_Cookieが優先されHeaderは無視される", func(t *testing.T) {
			t.Parallel()
			ac := newAuthConfig(t)
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			//nolint:gosec // G124: テスト用のリクエストクッキー
			req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "cookieTok"})
			req.Header.Set(ac.HeaderName(), "Bearer headerTok")
			tok := extractToken(req, ac)
			assert.Equal(t, "cookieTok", tok)
		})

		t.Run("Bearer形式のヘッダの場合、トークン部分が抽出される", func(t *testing.T) {
			t.Parallel()
			ac := newAuthConfig(t)
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			req.Header.Set(ac.HeaderName(), "Bearer abcdef")
			tok := extractToken(req, ac)
			assert.Equal(t, "abcdef", tok)
		})

		t.Run("AllowedHeaderBearer=falseの場合はヘッダ値をそのまま返す", func(t *testing.T) {
			t.Parallel()
			ac := newAuthConfig(t)
			ac.SetHeaderName(t, "X-API-KEY")
			ac.SetAllowedHeaderBearer(t, false)

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			req.Header.Set("X-Api-Key", "apikey-123")
			tok := extractToken(req, ac)
			assert.Equal(t, "apikey-123", tok)
		})

		t.Run("AuthorizationヘッダかつAllowedHeaderBearer=falseの場合はrawを返す", func(t *testing.T) {
			t.Parallel()
			ac := newAuthConfig(t)
			ac.SetAllowedHeaderBearer(t, false)

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			req.Header.Set(ac.HeaderName(), "Bearer secret")
			tok := extractToken(req, ac)
			assert.Equal(t, "Bearer secret", tok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("HeaderNameが空文字列の場合_ヘッダ値があっても空を返す", func(t *testing.T) {
			t.Parallel()
			ac := newAuthConfig(t)
			ac.SetHeaderName(t, "")

			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer smallcase")
			tok := extractToken(req, ac)
			assert.Empty(t, tok)
		})

		t.Run("Bearer期待時にprefixがなければ空を返す", func(t *testing.T) {
			t.Parallel()
			ac := newAuthConfig(t)
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			req.Header.Set(ac.HeaderName(), "Token abcdef")
			tok := extractToken(req, ac)
			assert.Empty(t, tok)
		})

		t.Run("Headerが未設定なら空を返す", func(t *testing.T) {
			t.Parallel()
			ac := newAuthConfig(t)
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			tok := extractToken(req, ac)
			assert.Empty(t, tok)
		})
	})
}
