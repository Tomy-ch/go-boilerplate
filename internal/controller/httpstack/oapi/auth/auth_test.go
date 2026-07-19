package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/controller/ctxhelper"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	mock_auth "go-boilerplate/internal/usecase/boundary/auth/mock"
	"go-boilerplate/pkg/xerrors"

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
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			want, _ := authbd.New("user123", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)

			fn := NewAuthenticator(m)

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
			want, _ := authbd.New("user123", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)

			fn := NewAuthenticator(m)

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
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, xerrors.New("bad"))

			fn := NewAuthenticator(m)

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

			fn := NewAuthenticator(m)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			in := &openapi3filter.AuthenticationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req}}

			err := fn(context.Background(), in)
			require.ErrorIs(t, err, ErrUnauthorizedTokenNotProvided)
		})

		t.Run("Authenticateがnil,nilを返す場合はErrUnauthorizedTokenNotProvidedを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mock_auth.NewMockAuthenticator(ctrl)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, nil)

			fn := NewAuthenticator(m)

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
			want, _ := authbd.New("subj", "mock", nil, nil)
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(want, nil)
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			got, err := authExtractor(context.Background(), req, m)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, want.Subject(), got.Subject())
		})

		t.Run("トークンが空なら認証スキップとしてnil,nilを返す", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			authn, err := authExtractor(ctx, httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil), nil)
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
			m.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, xerrors.New("bad"))
			ctx := context.Background()
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer tok")

			authn, err := authExtractor(context.Background(), req, m)
			require.ErrorIs(t, err, ErrUnauthorizedInvalidToken)
			assert.Nil(t, authn)
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
