package integration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	jose "github.com/go-jose/go-jose/v4"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/httpstack/oapi/auth"
	authjwt "go-boilerplate/internal/infrastructure/auth/jwt"
	"go-boilerplate/internal/infrastructure/httpclient"
	mock_httpclient "go-boilerplate/internal/infrastructure/httpclient/mock"
	"go-boilerplate/internal/infrastructure/system"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/uuid"
)

const (
	jwtTestIssuer      = "https://mock-auth.example.com"
	jwtTestAudience    = "go-boilerplate-api"
	jwtTestSubject     = "user-active"
	jwtTestKID         = "mock-key-1"
	jwtAccessTokenType = "at+jwt"

	// jwtTestSubjectDeleted / jwtTestSubjectUnknown は IdentityResolver 段での失敗経路を検証するための subject。
	jwtTestSubjectDeleted = "user-deleted"
	jwtTestSubjectUnknown = "user-unknown"
	// jwtTestResolvedUserID は stub resolver が subject を解決した内部 UserID。
	jwtTestResolvedUserID = "22222222-2222-2222-2222-222222222222"
)

// stubIdentityResolver は、DB を用いずに IdentityResolver の配線（認証成功後の内部ユーザー解決）を
// 検証するためのインメモリ実装。subject に応じて解決成功 / 削除済み / 未登録を返す。
type stubIdentityResolver struct {
	userID uuid.UUID
}

func (r stubIdentityResolver) Resolve(_ context.Context, authn *authbd.Authn) (*authbd.Authn, error) {
	switch authn.Subject() {
	case jwtTestSubjectDeleted:
		return nil, authbd.ErrUserUnavailable
	case jwtTestSubjectUnknown:
		return nil, authbd.ErrIdentityNotFound
	default:
		return authn.WithUserID(r.userID), nil
	}
}

// newJWTKey はテスト用の RSA 鍵ペアを生成する。
func newJWTKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

// jwksJSON は公開鍵 1 本を持つ JWKS（公開部のみ）の JSON を go-jose で組み立てる。
func jwksJSON(t *testing.T, pub *rsa.PublicKey, kid string) []byte {
	t.Helper()
	set := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{{Key: pub, KeyID: kid, Use: "sig", Algorithm: "RS256"}},
	}
	raw, err := json.Marshal(set)
	require.NoError(t, err)
	return raw
}

// validAccessClaims は検証を通過する access token の標準クレームを返す（実時刻基準）。
func validAccessClaims() jwtlib.MapClaims {
	now := time.Now()
	return jwtlib.MapClaims{
		"iss": jwtTestIssuer,
		"aud": jwtTestAudience,
		"sub": jwtTestSubject,
		"iat": jwtlib.NewNumericDate(now),
		"nbf": jwtlib.NewNumericDate(now.Add(-time.Hour)),
		"exp": jwtlib.NewNumericDate(now.Add(time.Hour)),
	}
}

// signAccessToken は kid を付け、指定の署名鍵・typ・クレームでトークンを署名する。
func signAccessToken(t *testing.T, key *rsa.PrivateKey, method jwtlib.SigningMethod, typ string, claims jwtlib.MapClaims) string {
	t.Helper()
	token := jwtlib.NewWithClaims(method, claims)
	token.Header["kid"] = jwtTestKID
	if typ != "" {
		token.Header["typ"] = typ
	}

	if method == jwtlib.SigningMethodNone {
		signed, err := token.SignedString(jwtlib.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)
		return signed
	}

	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

// bearerHeader は Bearer トークンの Authorization ヘッダを組み立てる。
func bearerHeader(token string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+token)
	return h
}

// startJWTAuthServer は、authenticator を auth ミドルウェア経由で配線した保護付き Echo を起動する。
// JWKS は httpclient substrate のモックから供給し、Bearer 抽出 → Credential → Authenticate → JWKS 解決 → 検証 → 401/200 の HTTP 境界経路を実検証する。
func startJWTAuthServer(t *testing.T, key *rsa.PrivateKey) *Server {
	t.Helper()

	ctrl := gomock.NewController(t)
	client := mock_httpclient.NewMockClient(ctrl)
	client.EXPECT().Do(gomock.Any(), gomock.Any()).
		Return(&httpclient.Response{StatusCode: 200, Body: jwksJSON(t, &key.PublicKey, jwtTestKID)}, nil).AnyTimes()

	authenticator, err := authjwt.NewJWKS(authjwt.JWKSParams{
		Params: authjwt.Params{
			Issuer:       jwtTestIssuer,
			Audience:     jwtTestAudience,
			ExpectedType: jwtAccessTokenType,
			Clock:        system.NewClock(),
		},
		JWKSURL:          "http://jwks.example.test/keys.json",
		AllowInsecureURL: true,
	}, client)
	require.NoError(t, err)

	return newProtectedServer(t, authenticator)
}

// newProtectedServer は、与えた authenticator を auth ミドルウェアへ配線した保護付き Echo（GET /protected）を起動する。
// Bearer 抽出 → Credential → Authenticate → IdentityResolver → 401/200 の HTTP 境界経路を実検証するための土台。
func newProtectedServer(t *testing.T, authenticator authbd.Authenticator) *Server {
	t.Helper()

	resolvedUserID, err := uuid.Parse(jwtTestResolvedUserID)
	require.NoError(t, err)
	authFunc := auth.NewAuthenticator(authenticator, stubIdentityResolver{userID: resolvedUserID})

	e := echo.New()
	UseAppErrorHandler(t, e)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			req = req.WithContext(ctxhelper.WithAuthn(req.Context()))
			c.SetRequest(req)

			input := &openapi3filter.AuthenticationInput{
				RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req},
			}
			if err := authFunc(req.Context(), input); err != nil {
				return err
			}
			return next(c)
		}
	})
	// 解決済み UserID がハンドラまで伝播することを検証するため、Authn.UserID をレスポンスに載せる。
	e.GET("/protected", func(c echo.Context) error {
		authn, ok := ctxhelper.GetAuthn(c.Request().Context())
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "authn missing"})
		}
		userID, err := authn.UserID()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user id unresolved"})
		}
		return c.JSON(http.StatusOK, map[string]string{"user_id": userID.String()})
	})

	return StartServer(t, e)
}

func TestJWTAuthIntegration(t *testing.T) {
	t.Parallel()

	key := newJWTKey(t)
	srv := startJWTAuthServer(t, key)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正当な access token を提示すると保護リソースへ到達し解決済み UserID が伝播する", func(t *testing.T) {
			t.Parallel()
			token := signAccessToken(t, key, jwtlib.SigningMethodRS256, jwtAccessTokenType, validAccessClaims())
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			require.Equal(t, http.StatusOK, res.StatusCode)

			var body map[string]string
			require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
			assert.Equal(t, jwtTestResolvedUserID, body["user_id"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トークン未提示は401になる", func(t *testing.T) {
			t.Parallel()
			res := srv.DoJSON(http.MethodGet, "/protected", nil, nil)
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})

		t.Run("別の鍵で署名した不正署名トークンは401になる", func(t *testing.T) {
			t.Parallel()
			otherKey := newJWTKey(t)
			token := signAccessToken(t, otherKey, jwtlib.SigningMethodRS256, jwtAccessTokenType, validAccessClaims())
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})

		t.Run("期限切れトークンは401になる", func(t *testing.T) {
			t.Parallel()
			claims := validAccessClaims()
			claims["exp"] = jwtlib.NewNumericDate(time.Now().Add(-time.Hour))
			token := signAccessToken(t, key, jwtlib.SigningMethodRS256, jwtAccessTokenType, claims)
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})

		t.Run("nbf未達トークンは401になる", func(t *testing.T) {
			t.Parallel()
			claims := validAccessClaims()
			claims["nbf"] = jwtlib.NewNumericDate(time.Now().Add(time.Hour))
			token := signAccessToken(t, key, jwtlib.SigningMethodRS256, jwtAccessTokenType, claims)
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})

		t.Run("issuer不一致トークンは401になる", func(t *testing.T) {
			t.Parallel()
			claims := validAccessClaims()
			claims["iss"] = "https://evil.example.com"
			token := signAccessToken(t, key, jwtlib.SigningMethodRS256, jwtAccessTokenType, claims)
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})

		t.Run("audience不一致トークンは401になる", func(t *testing.T) {
			t.Parallel()
			claims := validAccessClaims()
			claims["aud"] = "other-api"
			token := signAccessToken(t, key, jwtlib.SigningMethodRS256, jwtAccessTokenType, claims)
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})

		t.Run("subject欠落トークンは401になる", func(t *testing.T) {
			t.Parallel()
			claims := validAccessClaims()
			delete(claims, "sub")
			token := signAccessToken(t, key, jwtlib.SigningMethodRS256, jwtAccessTokenType, claims)
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})

		t.Run("typ が at+jwt でない ID Token は401になる", func(t *testing.T) {
			t.Parallel()
			// ID Token は typ=at+jwt を持たないため、access token 用の typ 検証で拒否される。
			token := signAccessToken(t, key, jwtlib.SigningMethodRS256, "JWT", validAccessClaims())
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})

		t.Run("alg=none の非対称署名なしトークンは401になる", func(t *testing.T) {
			t.Parallel()
			token := signAccessToken(t, key, jwtlib.SigningMethodNone, jwtAccessTokenType, validAccessClaims())
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})

		t.Run("削除済みユーザーの identity は検証を通っても401になる", func(t *testing.T) {
			t.Parallel()
			claims := validAccessClaims()
			claims["sub"] = jwtTestSubjectDeleted
			token := signAccessToken(t, key, jwtlib.SigningMethodRS256, jwtAccessTokenType, claims)
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})

		t.Run("未登録の identity は検証を通っても401になる", func(t *testing.T) {
			t.Parallel()
			claims := validAccessClaims()
			claims["sub"] = jwtTestSubjectUnknown
			token := signAccessToken(t, key, jwtlib.SigningMethodRS256, jwtAccessTokenType, claims)
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertErrorResponse(t, res, http.StatusUnauthorized)
		})
	})
}
