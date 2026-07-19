package integration

import (
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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/httpstack/oapi/auth"
	authjwt "go-boilerplate/internal/infrastructure/auth/jwt"
	"go-boilerplate/internal/infrastructure/httpclient"
	mock_httpclient "go-boilerplate/internal/infrastructure/httpclient/mock"
	"go-boilerplate/internal/infrastructure/system"
)

const (
	jwtTestIssuer      = "https://mock-auth.example.com"
	jwtTestAudience    = "go-boilerplate-api"
	jwtTestSubject     = "11111111-1111-1111-1111-111111111111"
	jwtTestKID         = "mock-key-1"
	jwtAccessTokenType = "at+jwt"
)

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
		JWKSURL: "http://jwks.example.test/keys.json",
	}, client)
	require.NoError(t, err)

	authFunc := auth.NewAuthenticator(authenticator)

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
	e.GET("/protected", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	return StartServer(t, e)
}

func TestJWTAuthIntegration(t *testing.T) {
	t.Parallel()

	key := newJWTKey(t)
	srv := startJWTAuthServer(t, key)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正当な access token を提示すると保護リソースへ到達できる", func(t *testing.T) {
			t.Parallel()
			token := signAccessToken(t, key, jwtlib.SigningMethodRS256, jwtAccessTokenType, validAccessClaims())
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
			AssertJSONResponseType[map[string]string](t, res)
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
	})
}
