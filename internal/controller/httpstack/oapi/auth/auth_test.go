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
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewAuthenticator(t *testing.T) {
	t.Parallel()

	t.Run("エコーコンテキストがない場合はエラー", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		ac := config.NewAuthConfig(cfg)

		ctrl := gomock.NewController(t)
		m := mock_auth.NewMockAuthenticator(ctrl)
		ctx := context.Background()

		fn := NewAuthenticator(ac, m)

		// リクエストに Echo コンテキストがセットされていない
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

		err := fn(context.Background(), in)
		require.ErrorIs(t, err, ErrUnauthorizedEchoContextNotFound)
	})

	t.Run("authenticator がエラーを返すと ErrUnauthorizedInvalidToken を返す", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		ac := config.NewAuthConfig(cfg)

		ctrl := gomock.NewController(t)
		ctx := context.Background()
		m := mock_auth.NewMockAuthenticator(ctrl)
		m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, errors.New("bad"))

		fn := NewAuthenticator(ac, m)

		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		e := echo.New()
		echoCtx := e.NewContext(req, rec)
		ctxWithEcho := ctxhelper.SetEchoContext(req.Context(), echoCtx)
		req = req.WithContext(ctxWithEcho)

		in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

		//nolint:gosec // G124: テスト用のリクエストクッキー。Secure/HttpOnly/SameSite はサーバが Set-Cookie で付与する属性で、リクエスト側のクッキーには適用されないため対応不要
		req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "tok"})
		in.RequestValidationInput.Request = req

		err := fn(context.Background(), in)
		require.ErrorIs(t, err, ErrUnauthorizedInvalidToken)
	})

	t.Run("認証情報が取得できない場合は ErrUnauthorizedTokenNotProvided を返す", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		ac := config.NewAuthConfig(cfg)

		ctrl := gomock.NewController(t)
		ctx := context.Background()
		m := mock_auth.NewMockAuthenticator(ctrl)

		fn := NewAuthenticator(ac, m)

		// echo context はあるが token は与えない -> authExtractor は nil,nil を返す
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		e := echo.New()
		echoCtx := e.NewContext(req, rec)
		ctxWithEcho := ctxhelper.SetEchoContext(req.Context(), echoCtx)
		req = req.WithContext(ctxWithEcho)
		in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

		err := fn(context.Background(), in)
		require.ErrorIs(t, err, ErrUnauthorizedTokenNotProvided)
	})

	t.Run("Authenticate が nil, nil を返す場合は ErrUnauthorizedTokenNotProvided を返す", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		ac := config.NewAuthConfig(cfg)

		ctrl := gomock.NewController(t)
		ctx := context.Background()
		m := mock_auth.NewMockAuthenticator(ctrl)
		m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, nil)

		fn := NewAuthenticator(ac, m)

		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		//nolint:gosec // G124: テスト用のリクエストクッキー。Secure/HttpOnly/SameSite はサーバが Set-Cookie で付与する属性で、リクエスト側のクッキーには適用されないため対応不要
		req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "tok"})
		rec := httptest.NewRecorder()
		e := echo.New()
		echoCtx := e.NewContext(req, rec)
		ctxWithEcho := ctxhelper.SetEchoContext(req.Context(), echoCtx)
		req = req.WithContext(ctxWithEcho)
		in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

		err := fn(context.Background(), in)
		require.ErrorIs(t, err, ErrUnauthorizedTokenNotProvided)
	})

	t.Run("正常系: トークン抽出 -> Authenticate 呼出し -> Echo に Authn セット", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		ac := config.NewAuthConfig(cfg)

		// mock authenticator: return expected Authn
		ctrl := gomock.NewController(t)
		m := mock_auth.NewMockAuthenticator(ctrl)
		want, _ := authbd.New("user123", "mock", nil, nil)
		m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)

		fn := NewAuthenticator(ac, m)

		// リクエストと Echo コンテキストを作成し、request.Context に Echo コンテキストを格納する
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		//nolint:gosec // G124: テスト用のリクエストクッキー。Secure/HttpOnly/SameSite はサーバが Set-Cookie で付与する属性で、リクエスト側のクッキーには適用されないため対応不要
		req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "user123"})
		rec := httptest.NewRecorder()
		e := echo.New()
		echoCtx := e.NewContext(req, rec)
		// リクエストのコンテキストに echo.Context をセット
		ctxWithEcho := ctxhelper.SetEchoContext(req.Context(), echoCtx)
		req = req.WithContext(ctxWithEcho)

		in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

		err := fn(context.Background(), in)
		require.NoError(t, err)

		// echoCtx に Authn がセットされていることを確認
		got, ok := ctxhelper.GetAuthnFromEcho(echoCtx)
		require.True(t, ok)
		require.Equal(t, want.Subject(), got.Subject())
	})
}

func Test_authExtractor(t *testing.T) {
	t.Run("トークンが空なら (cookie/header なし) nil,nil を返す", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		ac := config.NewAuthConfig(cfg)
		ctx := context.Background()

		// モックは不要
		authn, err := authExtractor(ctx, httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil), ac, nil)
		require.NoError(t, err)
		require.Nil(t, authn)
	})

	t.Run("Authenticate がエラーを返すと ErrUnauthorizedInvalidToken を返す", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		ac := config.NewAuthConfig(cfg)

		ctrl := gomock.NewController(t)
		m := mock_auth.NewMockAuthenticator(ctrl)
		m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, errors.New("bad"))
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		// cookie からトークンを取る想定
		//nolint:gosec // G124: テスト用のリクエストクッキー。Secure/HttpOnly/SameSite はサーバが Set-Cookie で付与する属性で、リクエスト側のクッキーには適用されないため対応不要
		req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "tok"})

		authn, err := authExtractor(context.Background(), req, ac, m)
		require.ErrorIs(t, err, ErrUnauthorizedInvalidToken)
		require.Nil(t, authn)
	})

	t.Run("正常系: Authenticate の結果を返す", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		ac := config.NewAuthConfig(cfg)

		ctrl := gomock.NewController(t)
		m := mock_auth.NewMockAuthenticator(ctrl)
		want, _ := authbd.New("subj", "mock", nil, nil)
		m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		//nolint:gosec // G124: テスト用のリクエストクッキー。Secure/HttpOnly/SameSite はサーバが Set-Cookie で付与する属性で、リクエスト側のクッキーには適用されないため対応不要
		req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "tok"})

		got, err := authExtractor(context.Background(), req, ac, m)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, want.Subject(), got.Subject())
	})
}

func Test_extractToken(t *testing.T) {
	cfg := config.MockConfigForTest(t)

	t.Run("Cookie から抽出できる", func(t *testing.T) {
		ac := config.NewAuthConfig(cfg)

		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		//nolint:gosec // G124: テスト用のリクエストクッキー。Secure/HttpOnly/SameSite はサーバが Set-Cookie で付与する属性で、リクエスト側のクッキーには適用されないため対応不要
		req.AddCookie(&http.Cookie{Name: ac.CookieName(), Value: "cookieTok"})
		tok := extractToken(req, ac)
		require.Equal(t, "cookieTok", tok)
	})

	t.Run("Header: Bearer 形式の場合抽出される", func(t *testing.T) {
		ac := config.NewAuthConfig(cfg)
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		req.Header.Set(ac.HeaderName(), "Bearer abcdef")
		tok := extractToken(req, ac)
		require.Equal(t, "abcdef", tok)
	})

	t.Run("CookieNameがあるが、HeaderNameが空文字列の場合は空を返す", func(t *testing.T) {
		ac := config.NewAuthConfig(cfg)
		ac.SetHeaderName(t, "")

		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		req.Header.Set("authorization", "Bearer smallcase")
		tok := extractToken(req, ac)
		require.Empty(t, tok)
	})

	t.Run("Header: Bearer 期待する場合に prefix がなければ空", func(t *testing.T) {
		ac := config.NewAuthConfig(cfg)
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		req.Header.Set(ac.HeaderName(), "Token abcdef")
		tok := extractToken(req, ac)
		require.Empty(t, tok)
	})

	t.Run("Header: AllowedHeaderBearer=false の場合はヘッダ値をそのまま返す", func(t *testing.T) {
		ac := config.NewAuthConfig(cfg)
		ac.SetHeaderName(t, "X-API-KEY")
		ac.SetAllowedHeaderBearer(t, false)

		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		req.Header.Set("X-API-KEY", "apikey-123")
		tok := extractToken(req, ac)
		require.Equal(t, "apikey-123", tok)
	})

	t.Run("Authorization ヘッダかつ AllowedHeaderBearer=false の場合は raw を返す", func(t *testing.T) {
		ac := config.NewAuthConfig(cfg)
		ac.SetAllowedHeaderBearer(t, false)

		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		req.Header.Set(ac.HeaderName(), "Bearer secret")
		tok := extractToken(req, ac)
		require.Equal(t, "Bearer secret", tok)
	})

	t.Run("Header が空文字列なら空を返す", func(t *testing.T) {
		ac := config.NewAuthConfig(cfg)
		ctx := context.Background()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		tok := extractToken(req, ac)
		require.Empty(t, tok)
	})
}
