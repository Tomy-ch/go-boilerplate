package jwt

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"sync"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/infrastructure/httpclient"
	mock_httpclient "go-boilerplate/internal/infrastructure/httpclient/mock"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/pkg/xerrors"
)

const (
	testKID = "mock-key-1"
	// jwksTestURL は resolver テストで使う任意の JWKS エンドポイント URL です（取得はモックで代替）。
	jwksTestURL = "http://jwks.example.test/keys.json"
)

// errBoom は fetch 失敗を模す任意の下流エラーです。
var errBoom = xerrors.New("jwks fetch boom")

// fakeClock は手動で時刻を前進できるテスト用の clock.Clock です（TTL / cooldown の境界検証用）。
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// jwksPublicJSON は testKID を持つ公開鍵 1 本の JWKS（公開部のみ）の JSON を go-jose で組み立てます。
func jwksPublicJSON(t *testing.T, pub *rsa.PublicKey) []byte {
	t.Helper()
	set := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{{Key: pub, KeyID: testKID, Use: "sig", Algorithm: "RS256"}},
	}
	raw, err := json.Marshal(set)
	require.NoError(t, err)
	return raw
}

// signTokenWithKID は kid ヘッダを付けて RS256 で署名したトークン文字列を返します。
func signTokenWithKID(t *testing.T, key *rsa.PrivateKey, kid string, claims jwtlib.MapClaims) string {
	t.Helper()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

// stubJWKSClient は Do が常に body(JWKS) を返す httpclient.Client のモックです。
func stubJWKSClient(t *testing.T, body []byte) httpclient.Client {
	t.Helper()
	ctrl := gomock.NewController(t)
	client := mock_httpclient.NewMockClient(ctrl)
	client.EXPECT().Do(gomock.Any(), gomock.Any()).
		Return(&httpclient.Response{StatusCode: 200, Body: body}, nil).AnyTimes()
	return client
}

// newJWKSAuthenticator は、JWKS を返すモック httpclient で kid 解決する Authenticator を生成します。
func newJWKSAuthenticator(t *testing.T, jwks []byte) authbd.Authenticator {
	t.Helper()
	a, err := NewJWKS(JWKSParams{
		Params: Params{
			Issuer:   testIssuer,
			Audience: testAudience,
			Clock:    testkit.NewMockClock(t, fixedNow),
		},
		JWKSURL: "http://jwks.example.test/keys.json",
	}, stubJWKSClient(t, jwks))
	require.NoError(t, err)
	return a
}

// tokenWithKID は kid ヘッダのみを持つ検証前トークンを組み立てます（resolver.keyfunc の直接検証用）。
func tokenWithKID() *jwtlib.Token {
	return &jwtlib.Token{Header: map[string]any{"kid": testKID}}
}

func TestNewDownstreamProfile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("外部 IdP へ相関 ID を漏らさないよう trace を伝搬しない設定になる", func(t *testing.T) {
			t.Parallel()
			assert.False(t, NewDownstreamProfile().Profile.PropagateTrace)
		})

		t.Run("開発モックの private アドレスへ到達できるよう private network を許可する", func(t *testing.T) {
			t.Parallel()
			assert.True(t, NewDownstreamProfile().Profile.AllowPrivateNetwork)
		})

		t.Run("Name が jwks downstream になる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, RequiredDownstream(), NewDownstreamProfile().Name)
		})
	})
}

func TestRequiredDownstream(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("jwks downstream を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, httpclient.Downstream("jwks"), RequiredDownstream())
		})
	})
}

func TestParseJWKSKeys(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("kid 付き JWKS を kid→公開鍵へパースできる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			keys, err := parseJWKSKeys(jwksPublicJSON(t, &key.PublicKey))
			require.NoError(t, err)
			assert.Contains(t, keys, testKID)
		})

		t.Run("KeyID の無いエントリを含む場合、kid 付き鍵のみ採用される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			set := jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{
					{Key: &key.PublicKey, KeyID: "", Use: "sig", Algorithm: "RS256"},
					{Key: &key.PublicKey, KeyID: testKID, Use: "sig", Algorithm: "RS256"},
				},
			}
			raw, err := json.Marshal(set)
			require.NoError(t, err)

			keys, err := parseJWKSKeys(raw)
			require.NoError(t, err)
			assert.Len(t, keys, 1)
			assert.Contains(t, keys, testKID)
		})

		t.Run("非 RSA 鍵（EC）を含む場合、RSA 公開鍵のみ採用される", func(t *testing.T) {
			t.Parallel()
			rsaKey := newRSAKey(t)
			ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			require.NoError(t, err)
			set := jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{
					{Key: &ecKey.PublicKey, KeyID: "ec-key", Use: "sig", Algorithm: "ES256"},
					{Key: &rsaKey.PublicKey, KeyID: testKID, Use: "sig", Algorithm: "RS256"},
				},
			}
			raw, err := json.Marshal(set)
			require.NoError(t, err)

			keys, err := parseJWKSKeys(raw)
			require.NoError(t, err)
			assert.Len(t, keys, 1)
			assert.Contains(t, keys, testKID)
			assert.NotContains(t, keys, "ec-key")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON が不正な場合はエラーになる", func(t *testing.T) {
			t.Parallel()
			_, err := parseJWKSKeys([]byte("{"))
			require.ErrorIs(t, err, errJWKSMalformed)
		})

		t.Run("利用可能な鍵が無い場合はエラーになる", func(t *testing.T) {
			t.Parallel()
			_, err := parseJWKSKeys([]byte(`{"keys":[]}`))
			require.ErrorIs(t, err, errJWKSNoKeys)
		})
	})
}

func TestNewJWKS(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JWKS の kid に一致する鍵で署名したトークンを検証できる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			a := newJWKSAuthenticator(t, jwksPublicJSON(t, &key.PublicKey))

			token := signTokenWithKID(t, key, testKID, validClaims())
			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Equal(t, testSubject, authn.Subject())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JWKS URL が空の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			a, err := NewJWKS(JWKSParams{
				Params: Params{
					Issuer:   testIssuer,
					Audience: testAudience,
					Clock:    testkit.NewMockClock(t, fixedNow),
				},
				JWKSURL: "",
			}, stubJWKSClient(t, nil))
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})

		t.Run("http client が nil の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			a, err := NewJWKS(JWKSParams{
				Params: Params{
					Issuer:   testIssuer,
					Audience: testAudience,
					Clock:    testkit.NewMockClock(t, fixedNow),
				},
				JWKSURL: "http://jwks.example.test/keys.json",
			}, nil)
			assert.Nil(t, a)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidParams)
		})

		t.Run("JWKS に存在しない鍵で署名したトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			otherKey := newRSAKey(t)
			a := newJWKSAuthenticator(t, jwksPublicJSON(t, &key.PublicKey))

			token := signTokenWithKID(t, otherKey, testKID, validClaims())
			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})
	})
}

func Test_jwksResolver_keyfunc(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("鮮度内キャッシュに kid がある場合、再取得せず解決する", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			// Times(1): 2 回目の解決は鮮度内キャッシュ命中で Do を呼ばない。
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: jwksPublicJSON(t, &key.PublicKey)}, nil).
				Times(1)
			r := newJWKSResolver(client, jwksTestURL, time.Hour, newFakeClock(fixedNow))

			k1, err := r.keyfunc(tokenWithKID())
			require.NoError(t, err)
			assert.NotNil(t, k1)

			k2, err := r.keyfunc(tokenWithKID())
			require.NoError(t, err)
			assert.NotNil(t, k2)
		})

		t.Run("キャッシュ TTL 経過後は再取得する", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			// Times(2): TTL 経過で鮮度切れとなり 2 回目は再取得する。
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: jwksPublicJSON(t, &key.PublicKey)}, nil).
				Times(2)
			clk := newFakeClock(fixedNow)
			r := newJWKSResolver(client, jwksTestURL, time.Hour, clk)

			_, err := r.keyfunc(tokenWithKID())
			require.NoError(t, err)

			clk.advance(time.Hour + time.Minute)
			_, err = r.keyfunc(tokenWithKID())
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fetch がエラーの場合、無効トークンへ正規化され原因を保持する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil, errBoom).Times(1)
			r := newJWKSResolver(client, jwksTestURL, time.Hour, newFakeClock(fixedNow))

			_, err := r.keyfunc(tokenWithKID())
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorIs(t, err, errBoom)
		})

		t.Run("cooldown 中の再取得は再fetchせず直近の失敗原因を伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			// Times(1): 2 回目は cooldown で throttle され Do を呼ばない（直近エラーを伝播）。
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil, errBoom).Times(1)
			r := newJWKSResolver(client, jwksTestURL, time.Hour, newFakeClock(fixedNow))

			_, err1 := r.keyfunc(tokenWithKID())
			require.ErrorIs(t, err1, errBoom)

			_, err2 := r.keyfunc(tokenWithKID())
			require.ErrorIs(t, err2, ErrJWTAuthenticatorInvalidToken)
			require.ErrorIs(t, err2, errBoom)
		})

		t.Run("cooldown 経過後は再取得する", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			// 1 回目は失敗、cooldown 経過後の 2 回目で再取得して回復する。
			firstFetch := client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil, errBoom).Times(1)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: jwksPublicJSON(t, &key.PublicKey)}, nil).
				Times(1).After(firstFetch)
			clk := newFakeClock(fixedNow)
			r := newJWKSResolver(client, jwksTestURL, time.Hour, clk)

			_, err1 := r.keyfunc(tokenWithKID())
			require.ErrorIs(t, err1, errBoom)

			clk.advance(jwksRefreshCooldown + time.Second)
			k, err2 := r.keyfunc(tokenWithKID())
			require.NoError(t, err2)
			assert.NotNil(t, k)
		})
	})
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
