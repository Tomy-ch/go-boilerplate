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
			assert.Equal(t, authbd.ProviderJWT, authn.Issuer())
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

func TestNewWithKeyfunc(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注入した鍵解決関数で正当なトークンを検証できる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a, err := NewWithKeyfunc(Params{
				Issuer:   testIssuer,
				Audience: testAudience,
				Clock:    testkit.NewMockClock(t, fixedNow),
			}, func(*jwtlib.Token) (any, error) { return &key.PublicKey, nil })
			require.NoError(t, err)

			token := signToken(t, key, jwtlib.SigningMethodRS256, "", validClaims())
			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Equal(t, testSubject, authn.Subject())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("鍵解決関数が nil の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			a, err := NewWithKeyfunc(Params{
				Issuer:   testIssuer,
				Audience: testAudience,
				Clock:    testkit.NewMockClock(t, fixedNow),
			}, nil)
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})
	})
}
