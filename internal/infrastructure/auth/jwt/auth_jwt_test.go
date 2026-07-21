package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/clock/testkit"
)

const (
	testIssuer   = "https://issuer.example.com"
	testAudience = "my-api"
	testSubject  = "user-123"
)

// fixedNow はテストで exp / nbf クレームの基準に使う固定時刻です。
var fixedNow = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// newRSAKey はテスト用の RSA 鍵ペアを生成します。
func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

// publicKeyPEM は RSA 公開鍵を PKIX 形式の PEM 文字列に変換します。
func publicKeyPEM(t *testing.T, pub *rsa.PublicKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

// validClaims は固定時刻を基準にした検証を通過する標準クレームを返します。
func validClaims() jwtlib.MapClaims {
	return jwtlib.MapClaims{
		"iss": testIssuer,
		"aud": testAudience,
		"sub": testSubject,
		"exp": jwtlib.NewNumericDate(fixedNow.Add(time.Hour)),
		"nbf": jwtlib.NewNumericDate(fixedNow.Add(-time.Hour)),
	}
}

// signToken は指定した署名方式・typ・クレームでトークン文字列を組み立てます。
func signToken(t *testing.T, key *rsa.PrivateKey, method jwtlib.SigningMethod, typ string, claims jwtlib.MapClaims) string {
	t.Helper()
	token := jwtlib.NewWithClaims(method, claims)
	if typ != "" {
		token.Header["typ"] = typ
	}

	var (
		signed string
		err    error
	)
	switch method {
	case jwtlib.SigningMethodNone:
		signed, err = token.SignedString(jwtlib.UnsafeAllowNoneSignatureType)
	case jwtlib.SigningMethodHS256:
		signed, err = token.SignedString([]byte("shared-secret"))
	default:
		signed, err = token.SignedString(key)
	}
	require.NoError(t, err)
	return signed
}

// newAuthenticator は公開鍵 PEM と固定クロックを注入した Authenticator を生成します。
func newAuthenticator(t *testing.T, pub *rsa.PublicKey, expectedType string) authbd.Authenticator {
	t.Helper()
	a, err := New(Params{
		PublicKeyPEM: publicKeyPEM(t, pub),
		Issuer:       testIssuer,
		Audience:     testAudience,
		ExpectedType: expectedType,
		Clock:        testkit.NewMockClock(t, fixedNow),
	})
	require.NoError(t, err)
	return a
}

// newAuthenticatorWithLeeway は leeway を注入した Authenticator を生成します（leeway 境界の検証用）。
func newAuthenticatorWithLeeway(t *testing.T, pub *rsa.PublicKey, leeway time.Duration) authbd.Authenticator {
	t.Helper()
	a, err := New(Params{
		PublicKeyPEM: publicKeyPEM(t, pub),
		Issuer:       testIssuer,
		Audience:     testAudience,
		Leeway:       leeway,
		Clock:        testkit.NewMockClock(t, fixedNow),
	})
	require.NoError(t, err)
	return a
}

// newCredential はトークン文字列から Credential を生成します。
func newCredential(t *testing.T, token string) *authbd.Credential {
	t.Helper()
	cred, err := authbd.NewCredential(authbd.SchemeBearer, token)
	require.NoError(t, err)
	return cred
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("妥当なパラメータで生成した Authenticator は正当なトークンを検証できる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a, err := New(Params{
				PublicKeyPEM: publicKeyPEM(t, &key.PublicKey),
				Issuer:       testIssuer,
				Audience:     testAudience,
				Clock:        testkit.NewMockClock(t, fixedNow),
			})
			require.NoError(t, err)

			token := signToken(t, key, jwtlib.SigningMethodRS256, "", validClaims())
			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Equal(t, testSubject, authn.Subject())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("公開鍵 PEM が不正な場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			a, err := New(Params{
				PublicKeyPEM: "not-a-pem",
				Issuer:       testIssuer,
				Audience:     testAudience,
				Clock:        testkit.NewMockClock(t, fixedNow),
			})
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidPublicKey)
		})

		t.Run("clock が nil の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a, err := New(Params{
				PublicKeyPEM: publicKeyPEM(t, &key.PublicKey),
				Issuer:       testIssuer,
				Audience:     testAudience,
				Clock:        nil,
			})
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})

		t.Run("issuer が空の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a, err := New(Params{
				PublicKeyPEM: publicKeyPEM(t, &key.PublicKey),
				Issuer:       "",
				Audience:     testAudience,
				Clock:        testkit.NewMockClock(t, fixedNow),
			})
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})

		t.Run("audience が空の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a, err := New(Params{
				PublicKeyPEM: publicKeyPEM(t, &key.PublicKey),
				Issuer:       testIssuer,
				Audience:     "",
				Clock:        testkit.NewMockClock(t, fixedNow),
			})
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})
	})
}

func Test_authenticator_Authenticate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正当な RS256 トークンから検証済み Authn が生成される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", validClaims())

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Equal(t, testSubject, authn.Subject())
			assert.Equal(t, testIssuer, authn.Issuer())
			assert.Equal(t, testIssuer, authn.Claims()["iss"])
		})

		t.Run("scope クレームが複数スペース区切りでも []string に分割される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			claims := validClaims()
			claims["scope"] = "read  write"
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Equal(t, []string{"read", "write"}, authn.Scopes())
		})

		t.Run("scope クレームが無い場合は scopes が空になる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", validClaims())

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Empty(t, authn.Scopes())
		})

		t.Run("scope クレームが文字列型でない場合は scopes が空になる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			claims := validClaims()
			claims["scope"] = []string{"read", "write"} // 標準外の配列形（方言）は抽出対象外
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Empty(t, authn.Scopes())
		})

		t.Run("exp が leeway 内に収まるトークンは受理される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticatorWithLeeway(t, &key.PublicKey, 60*time.Second)
			claims := validClaims()
			claims["exp"] = jwtlib.NewNumericDate(fixedNow.Add(-59 * time.Second))
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Equal(t, testSubject, authn.Subject())
		})

		t.Run("expectedType と一致する typ を持つトークンは受理される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "at+jwt")
			token := signToken(t, key, jwtlib.SigningMethodRS256, "at+jwt", validClaims())

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Equal(t, testSubject, authn.Subject())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cred が nil の場合は panic せず認証エラーになる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")

			authn, err := a.Authenticate(context.Background(), nil)
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("別の鍵で署名されたトークンは署名不一致で拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			otherKey := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			token := signToken(t, otherKey, jwtlib.SigningMethodRS256, "", validClaims())

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			// トークン内容起因の失敗が apperror.ErrUnauthenticated(= 401)へ収束することを代表ケースで確認する。
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("alg=none のトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			token := signToken(t, key, jwtlib.SigningMethodNone, "", validClaims())

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("HS256 で署名されたトークンは鍵混同を防ぐため拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			token := signToken(t, key, jwtlib.SigningMethodHS256, "", validClaims())

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("期限切れ（exp 経過）のトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			claims := validClaims()
			claims["exp"] = jwtlib.NewNumericDate(fixedNow.Add(-time.Hour))
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("exp クレームが無いトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			claims := validClaims()
			delete(claims, "exp")
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("nbf 未達のトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			claims := validClaims()
			claims["nbf"] = jwtlib.NewNumericDate(fixedNow.Add(time.Hour))
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("iss 不一致のトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			claims := validClaims()
			claims["iss"] = "https://evil.example.com"
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("aud 不一致のトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			claims := validClaims()
			claims["aud"] = "other-api"
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("aud 欠落のトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			claims := validClaims()
			delete(claims, "aud")
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("expectedType 設定時に typ が一致しないトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "at+jwt")
			token := signToken(t, key, jwtlib.SigningMethodRS256, "JWT", validClaims())

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("sub 欠落のトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			claims := validClaims()
			delete(claims, "sub")
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("sub が空白のみのトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticator(t, &key.PublicKey, "")
			claims := validClaims()
			claims["sub"] = "   "
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("exp が leeway を超えて期限切れのトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newAuthenticatorWithLeeway(t, &key.PublicKey, 60*time.Second)
			claims := validClaims()
			claims["exp"] = jwtlib.NewNumericDate(fixedNow.Add(-61 * time.Second))
			token := signToken(t, key, jwtlib.SigningMethodRS256, "", claims)

			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})
	})
}

func Test_authenticator_keyFunc(t *testing.T) {
	t.Parallel()

	// keyFunc は a.keyResolver のみを使うため、鍵解決関数だけ差した authenticator で単体検証する。
	const resolvedKey = "resolved-public-key"
	a := &authenticator{
		keyResolver: fixedKeyResolver{key: resolvedKey},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("RS256 は keyResolver が解決した鍵を返す", func(t *testing.T) {
			t.Parallel()
			got, err := a.keyFunc(context.Background())(&jwtlib.Token{Method: jwtlib.SigningMethodRS256})
			require.NoError(t, err)
			assert.Equal(t, resolvedKey, got)
		})

		t.Run("PS256(RSA-PSS) も keyResolver が解決した鍵を返す", func(t *testing.T) {
			t.Parallel()
			got, err := a.keyFunc(context.Background())(&jwtlib.Token{Method: jwtlib.SigningMethodPS256})
			require.NoError(t, err)
			assert.Equal(t, resolvedKey, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// alg allowlist(WithValidMethods)は第 1 層。ここでは allowlist をすり抜けて keyFunc に
		// 到達した場合の第 2 層(鍵種別不一致)の防御だけを直接検証する（鍵混同攻撃対策の regression）。
		t.Run("HS256(対称鍵)は allowlist をすり抜けても鍵種別不一致で拒否する", func(t *testing.T) {
			t.Parallel()
			got, err := a.keyFunc(context.Background())(&jwtlib.Token{Method: jwtlib.SigningMethodHS256})
			assert.Nil(t, got)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			assert.ErrorContains(t, err, "unexpected signing method type")
		})

		t.Run("ES256(ECDSA)は鍵種別不一致で拒否する", func(t *testing.T) {
			t.Parallel()
			got, err := a.keyFunc(context.Background())(&jwtlib.Token{Method: jwtlib.SigningMethodES256})
			assert.Nil(t, got)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("alg=none は鍵種別不一致で拒否する", func(t *testing.T) {
			t.Parallel()
			got, err := a.keyFunc(context.Background())(&jwtlib.Token{Method: jwtlib.SigningMethodNone})
			assert.Nil(t, got)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})
	})
}

func TestNewWithKeyResolver(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注入した KeyResolver で正当なトークンを検証できる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a, err := NewWithKeyResolver(Params{
				Issuer:   testIssuer,
				Audience: testAudience,
				Clock:    testkit.NewMockClock(t, fixedNow),
			}, fixedKeyResolver{key: &key.PublicKey})
			require.NoError(t, err)

			token := signToken(t, key, jwtlib.SigningMethodRS256, "", validClaims())
			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Equal(t, testSubject, authn.Subject())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("KeyResolver が nil の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			a, err := NewWithKeyResolver(Params{
				Issuer:   testIssuer,
				Audience: testAudience,
				Clock:    testkit.NewMockClock(t, fixedNow),
			}, nil)
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})
	})
}

// newKeyResolver は固定のダミー鍵を返す KeyResolver です（buildAuthenticator 検証用）。
func newKeyResolver() KeyResolver {
	return fixedKeyResolver{key: "dummy-key"}
}

func Test_buildJWKSURLProvider(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JWKSURL を明示した場合は静的にその URL を返す", func(t *testing.T) {
			t.Parallel()
			fn, err := buildJWKSURLProvider(JWKSParams{JWKSURL: "https://idp.example.test/keys.json"}, stubJWKSClient(t, nil))
			require.NoError(t, err)

			got, err := fn(context.Background())
			require.NoError(t, err)
			assert.Equal(t, "https://idp.example.test/keys.json", got)
		})

		t.Run("JWKSURL 未指定なら issuer からの discovery で jwks_uri を返す", func(t *testing.T) {
			t.Parallel()
			client := jwksClientReturning(t, discoveryDoc(t, testIssuer, testIssuer+"/keys.json"))
			fn, err := buildJWKSURLProvider(JWKSParams{
				Params: Params{Issuer: testIssuer, Clock: testkit.NewMockClock(t, fixedNow)},
			}, client)
			require.NoError(t, err)

			got, err := fn(context.Background())
			require.NoError(t, err)
			assert.Equal(t, testIssuer+"/keys.json", got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JWKSURL と issuer の両方が空なら設定エラーになる", func(t *testing.T) {
			t.Parallel()
			fn, err := buildJWKSURLProvider(JWKSParams{}, stubJWKSClient(t, nil))
			assert.Nil(t, fn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})

		t.Run("override URL が http かつ AllowInsecureURL=false なら設定エラーになる", func(t *testing.T) {
			t.Parallel()
			fn, err := buildJWKSURLProvider(JWKSParams{JWKSURL: "http://idp.example.test/keys.json"}, stubJWKSClient(t, nil))
			assert.Nil(t, fn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})

		t.Run("discovery(issuer=http) かつ AllowInsecureURL=false なら設定エラーになる", func(t *testing.T) {
			t.Parallel()
			fn, err := buildJWKSURLProvider(JWKSParams{
				Params: Params{Issuer: "http://issuer.example.test", Clock: testkit.NewMockClock(t, fixedNow)},
			}, stubJWKSClient(t, nil))
			assert.Nil(t, fn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})
	})
}

func Test_fixedKeyResolver_ResolveKey(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("kid によらず保持する固定公開鍵を返す", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			r := fixedKeyResolver{key: &key.PublicKey}

			got, err := r.ResolveKey(context.Background(), "any-kid")
			require.NoError(t, err)
			assert.Equal(t, &key.PublicKey, got)
		})
	})
}

func Test_buildAuthenticator(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("妥当なパラメータで authenticator を構築し expectedType / issuer を保持する", func(t *testing.T) {
			t.Parallel()
			a, err := buildAuthenticator(Params{
				Issuer:       testIssuer,
				Audience:     testAudience,
				ExpectedType: " at+jwt ",
				Clock:        testkit.NewMockClock(t, fixedNow),
			}, newKeyResolver())
			require.NoError(t, err)

			impl, ok := a.(*authenticator)
			require.True(t, ok)
			assert.Equal(t, "at+jwt", impl.expectedType)
			assert.Equal(t, testIssuer, impl.issuer)
		})

		t.Run("AllowedAlgs / Leeway 未指定でも既定値で構築できる", func(t *testing.T) {
			t.Parallel()
			a, err := buildAuthenticator(Params{
				Issuer:   testIssuer,
				Audience: testAudience,
				Clock:    testkit.NewMockClock(t, fixedNow),
			}, newKeyResolver())
			require.NoError(t, err)
			assert.NotNil(t, a)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("clock が nil の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			a, err := buildAuthenticator(Params{
				Issuer:   testIssuer,
				Audience: testAudience,
				Clock:    nil,
			}, newKeyResolver())
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})

		t.Run("issuer が空の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			a, err := buildAuthenticator(Params{
				Issuer:   "  ",
				Audience: testAudience,
				Clock:    testkit.NewMockClock(t, fixedNow),
			}, newKeyResolver())
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})

		t.Run("audience が空の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			a, err := buildAuthenticator(Params{
				Issuer:   testIssuer,
				Audience: "  ",
				Clock:    testkit.NewMockClock(t, fixedNow),
			}, newKeyResolver())
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})
	})
}

func Test_authenticator_verifyType(t *testing.T) {
	t.Parallel()

	newToken := func(typ any) *jwtlib.Token {
		tok := &jwtlib.Token{Header: map[string]any{}}
		if typ != nil {
			tok.Header["typ"] = typ
		}
		return tok
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("expectedType が空なら typ 検証をスキップして常に成功する", func(t *testing.T) {
			t.Parallel()
			a := &authenticator{expectedType: ""}
			require.NoError(t, a.verifyType(newToken("anything")))
		})

		t.Run("typ が expectedType と大文字小文字を無視して一致すれば成功する", func(t *testing.T) {
			t.Parallel()
			a := &authenticator{expectedType: "at+jwt"}
			require.NoError(t, a.verifyType(newToken("AT+JWT")))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("typ が expectedType と一致しない場合は無効トークンになる", func(t *testing.T) {
			t.Parallel()
			a := &authenticator{expectedType: "at+jwt"}
			err := a.verifyType(newToken("JWT"))
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("typ ヘッダが無い場合は無効トークンになる", func(t *testing.T) {
			t.Parallel()
			a := &authenticator{expectedType: "at+jwt"}
			err := a.verifyType(newToken(nil))
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("typ ヘッダが文字列型でない場合は無効トークンになる", func(t *testing.T) {
			t.Parallel()
			a := &authenticator{expectedType: "at+jwt"}
			err := a.verifyType(newToken(123))
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})
	})
}

func Test_extractScopes(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スペース区切りの scope 文字列を []string へ分割する", func(t *testing.T) {
			t.Parallel()
			got := extractScopes(jwtlib.MapClaims{"scope": "read  write"})
			assert.Equal(t, []string{"read", "write"}, got)
		})

		t.Run("scope クレームが無い場合は nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, extractScopes(jwtlib.MapClaims{}))
		})

		t.Run("scope クレームが文字列型でない場合は nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, extractScopes(jwtlib.MapClaims{"scope": []string{"read"}}))
		})

		t.Run("scope が空文字列の場合は空になる", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, extractScopes(jwtlib.MapClaims{"scope": "   "}))
		})
	})
}
