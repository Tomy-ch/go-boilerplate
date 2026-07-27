package jwt

import (
	"context"
	"crypto"
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

	"go-boilerplate/internal/apperror"
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

// testAllowedAlgs は parseJWKSKeys に渡す許可署名アルゴリズムです。
var testAllowedAlgs = []string{"RS256"}

// fakeClock は手動で時刻を前進できるテスト用の clock.Clock です（TTL / cooldown の境界検証用）。
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// countingJWKSClient は fetch 回数を数える httpclient.Client です。
// bodies を呼び出しごとに順に返し（末尾以降は最後を繰り返す）、並行呼び出しでも安全です。
type countingJWKSClient struct {
	mu     sync.Mutex
	calls  int
	bodies [][]byte
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

// signTokenWithTestKID は testKID を kid ヘッダに付けて RS256 で署名したトークン文字列を返します。
// jwksWithKey は、指定 kid の RSA 公開鍵 1 本の JWKS(JSON) を組み立てます。
func jwksWithKey(t *testing.T, kid string, pub *rsa.PublicKey) []byte {
	t.Helper()
	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: pub, KeyID: kid, Use: "sig", Algorithm: "RS256"}}}
	raw, err := json.Marshal(set)
	require.NoError(t, err)
	return raw
}

// jwksWith2Keys は、指定 2 kid の RSA 公開鍵を持つ JWKS(JSON) を組み立てます（ローテ後の集合を模す）。
func jwksWith2Keys(t *testing.T, kid1 string, pub1 *rsa.PublicKey, kid2 string, pub2 *rsa.PublicKey) []byte {
	t.Helper()
	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
		{Key: pub1, KeyID: kid1, Use: "sig", Algorithm: "RS256"},
		{Key: pub2, KeyID: kid2, Use: "sig", Algorithm: "RS256"},
	}}
	raw, err := json.Marshal(set)
	require.NoError(t, err)
	return raw
}

// jwksClientReturning は、Do 呼び出しごとに bodies を順に返す httpclient.Client です（末尾以降は最後を繰り返す）。
func jwksClientReturning(t *testing.T, bodies ...[]byte) httpclient.Client {
	t.Helper()
	ctrl := gomock.NewController(t)
	client := mock_httpclient.NewMockClient(ctrl)
	call := 0
	client.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(context.Context, *httpclient.Request) (*httpclient.Response, error) {
			b := bodies[min(call, len(bodies)-1)]
			call++
			return &httpclient.Response{StatusCode: 200, Body: b}, nil
		}).AnyTimes()
	return client
}

func (c *countingJWKSClient) Do(_ context.Context, _ *httpclient.Request) (*httpclient.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b := c.bodies[min(c.calls, len(c.bodies)-1)]
	c.calls++
	return &httpclient.Response{StatusCode: 200, Body: b}, nil
}

func (c *countingJWKSClient) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func signTokenWithTestKID(t *testing.T, key *rsa.PrivateKey, claims jwtlib.MapClaims) string {
	t.Helper()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = testKID

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

// staticURL は、常に固定 URL を返す jwksResolver 用の URL 供給関数です。
//
//nolint:unparam // テスト用ヘルパ。任意 URL を渡せるよう引数化している
func staticURL(u string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) { return u, nil }
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
		JWKSURL:          "http://jwks.example.test/keys.json",
		AllowInsecureURL: true,
	}, stubJWKSClient(t, jwks))
	require.NoError(t, err)
	return a
}

func TestNewDownstreamProfile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("外部 IdP へ相関 ID を漏らさないよう trace を伝搬しない設定になる", func(t *testing.T) {
			t.Parallel()
			assert.False(t, NewDownstreamProfile(false).Profile.PropagateTrace)
		})

		t.Run("allowPrivateNetwork 引数が Profile.AllowPrivateNetwork に反映される", func(t *testing.T) {
			t.Parallel()
			assert.False(t, NewDownstreamProfile(false).Profile.AllowPrivateNetwork)
			assert.True(t, NewDownstreamProfile(true).Profile.AllowPrivateNetwork)
		})

		t.Run("認証ホットパス向けに全体タイムアウト・試行回数・応答上限を絞る", func(t *testing.T) {
			t.Parallel()
			p := NewDownstreamProfile(false).Profile
			assert.Equal(t, jwksOverallTimeout, p.OverallTimeout)
			assert.Equal(t, jwksMaxAttempts, p.MaxAttempts)
			assert.Equal(t, int64(jwksMaxResponseBytes), p.MaxResponseBytes)
		})

		t.Run("Name が jwks downstream になる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, RequiredDownstream(), NewDownstreamProfile(false).Name)
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
			keys, err := parseJWKSKeys(jwksPublicJSON(t, &key.PublicKey), testAllowedAlgs)
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

			keys, err := parseJWKSKeys(raw, testAllowedAlgs)
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

			keys, err := parseJWKSKeys(raw, testAllowedAlgs)
			require.NoError(t, err)
			assert.Len(t, keys, 1)
			assert.Contains(t, keys, testKID)
			assert.NotContains(t, keys, "ec-key")
		})

		t.Run("暗号化用途（use=enc）の鍵は署名鍵として採用されない", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			set := jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{
					{Key: &key.PublicKey, KeyID: "enc-key", Use: "enc", Algorithm: "RS256"},
					{Key: &key.PublicKey, KeyID: testKID, Use: "sig", Algorithm: "RS256"},
				},
			}
			raw, err := json.Marshal(set)
			require.NoError(t, err)

			keys, err := parseJWKSKeys(raw, testAllowedAlgs)
			require.NoError(t, err)
			assert.Len(t, keys, 1)
			assert.Contains(t, keys, testKID)
			assert.NotContains(t, keys, "enc-key")
		})

		t.Run("JWK が宣言する alg が許可外なら採用されない", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			set := jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{
					{Key: &key.PublicKey, KeyID: "ps-key", Use: "sig", Algorithm: "PS256"},
					{Key: &key.PublicKey, KeyID: testKID, Use: "sig", Algorithm: "RS256"},
				},
			}
			raw, err := json.Marshal(set)
			require.NoError(t, err)

			keys, err := parseJWKSKeys(raw, testAllowedAlgs)
			require.NoError(t, err)
			assert.Len(t, keys, 1)
			assert.Contains(t, keys, testKID)
			assert.NotContains(t, keys, "ps-key")
		})

		t.Run("use / alg を宣言しない鍵も採用される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: testKID}}}
			raw, err := json.Marshal(set)
			require.NoError(t, err)

			keys, err := parseJWKSKeys(raw, testAllowedAlgs)
			require.NoError(t, err)
			assert.Contains(t, keys, testKID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON が不正な場合はエラーになる", func(t *testing.T) {
			t.Parallel()
			_, err := parseJWKSKeys([]byte("{"), testAllowedAlgs)
			require.ErrorIs(t, err, errJWKSMalformed)
		})

		t.Run("利用可能な鍵が無い場合はエラーになる", func(t *testing.T) {
			t.Parallel()
			_, err := parseJWKSKeys([]byte(`{"keys":[]}`), testAllowedAlgs)
			require.ErrorIs(t, err, errJWKSNoKeys)
		})

		t.Run("重複する kid を含む場合は文書ごと不採用になる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			set := jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{
					{Key: &key.PublicKey, KeyID: testKID, Use: "sig", Algorithm: "RS256"},
					{Key: &key.PublicKey, KeyID: testKID, Use: "sig", Algorithm: "RS256"},
				},
			}
			raw, err := json.Marshal(set)
			require.NoError(t, err)

			_, err = parseJWKSKeys(raw, testAllowedAlgs)
			require.ErrorIs(t, err, errJWKSDuplicateKID)
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

			token := signTokenWithTestKID(t, key, validClaims())
			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Equal(t, testSubject, authn.Subject())
		})

		t.Run("JWKSParams の AllowedAlgs / UnknownKidCooldown が resolver へ反映される", func(t *testing.T) {
			t.Parallel()
			a, err := NewJWKS(JWKSParams{
				Params: Params{
					Issuer:      testIssuer,
					Audience:    testAudience,
					AllowedAlgs: []string{"PS256"},
					Clock:       testkit.NewMockClock(t, fixedNow),
				},
				JWKSURL:            "https://idp.example.test/keys.json",
				UnknownKidCooldown: 30 * time.Second,
			}, stubJWKSClient(t, nil))
			require.NoError(t, err)

			authn, ok := a.(*authenticator)
			require.True(t, ok)
			r, ok := authn.keyResolver.(*jwksResolver)
			require.True(t, ok)
			assert.Equal(t, []string{"PS256"}, r.allowedAlgs)
			assert.Equal(t, 30*time.Second, r.cooldown)
		})

		t.Run("JWKSURL 未指定でも discovery 経由で鍵解決しトークンを検証できる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			client := jwksClientReturning(t,
				discoveryDoc(t, testIssuer, testIssuer+"/keys.json"),
				jwksPublicJSON(t, &key.PublicKey),
			)
			a, err := NewJWKS(JWKSParams{
				Params: Params{Issuer: testIssuer, Audience: testAudience, Clock: testkit.NewMockClock(t, fixedNow)},
			}, client)
			require.NoError(t, err)

			token := signTokenWithTestKID(t, key, validClaims())
			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			require.NoError(t, err)
			assert.Equal(t, testSubject, authn.Subject())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JWKS URL と issuer の両方が空の場合は設定エラーになる", func(t *testing.T) {
			t.Parallel()
			a, err := NewJWKS(JWKSParams{
				Params: Params{
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

		t.Run("discovery が失敗する場合は無効トークンへ正規化される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			// discovery 応答の issuer 不一致 → 解決失敗。
			client := jwksClientReturning(t, discoveryDoc(t, "https://evil.example.test", testIssuer+"/keys.json"))
			a, err := NewJWKS(JWKSParams{
				Params: Params{Issuer: testIssuer, Audience: testAudience, Clock: testkit.NewMockClock(t, fixedNow)},
			}, client)
			require.NoError(t, err)

			token := signTokenWithTestKID(t, key, validClaims())
			_, err = a.Authenticate(context.Background(), newCredential(t, token))
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})

		t.Run("kid は一致するが署名鍵が異なるトークンは拒否される", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			otherKey := newRSAKey(t)
			a := newJWKSAuthenticator(t, jwksPublicJSON(t, &key.PublicKey))

			token := signTokenWithTestKID(t, otherKey, validClaims())
			authn, err := a.Authenticate(context.Background(), newCredential(t, token))
			assert.Nil(t, authn)
			require.ErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
		})
	})
}

func Test_jwksResolver_ResolveKey(t *testing.T) {
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
			r := newJWKSResolver(client, staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))

			k1, err := r.ResolveKey(context.Background(), testKID)
			require.NoError(t, err)
			assert.Equal(t, &key.PublicKey, k1)

			k2, err := r.ResolveKey(context.Background(), testKID)
			require.NoError(t, err)
			assert.Equal(t, &key.PublicKey, k2)
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
			r := newJWKSResolver(client, staticURL(jwksTestURL), time.Hour, nil, 0, clk)

			_, err := r.ResolveKey(context.Background(), testKID)
			require.NoError(t, err)

			clk.advance(time.Hour + time.Minute)
			_, err = r.ResolveKey(context.Background(), testKID)
			require.NoError(t, err)
		})

		t.Run("鮮度内でも未知 kid は cooldown 経過後に再取得しローテ後の鍵を解決する", func(t *testing.T) {
			t.Parallel()
			keyA := newRSAKey(t)
			keyB := newRSAKey(t)
			clk := newFakeClock(fixedNow)
			r := newJWKSResolver(
				jwksClientReturning(t, jwksWithKey(t, testKID, &keyA.PublicKey), jwksWithKey(t, "kid-2", &keyB.PublicKey)),
				staticURL(jwksTestURL),
				time.Hour,
				nil,
				0,
				clk,
			)

			kA, err := r.ResolveKey(context.Background(), testKID)
			require.NoError(t, err)
			assert.Equal(t, &keyA.PublicKey, kA)

			clk.advance(jwksRefreshCooldown + time.Second)
			kB, err := r.ResolveKey(context.Background(), "kid-2")
			require.NoError(t, err)
			assert.Equal(t, &keyB.PublicKey, kB)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// ResolveKey は原因のみを持つ素の error を返し、ErrJWTAuthenticatorInvalidToken への
		// 正規化は Authenticate の境界一箇所に集約する。内側でセンチネルを付けない契約を固定する
		// （境界での正規化は TestNewJWKS の Authenticate 経由ケースで担保）。
		t.Run("fetch がエラーの場合、正規化せず原因を素の error として伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil, errBoom).Times(1)
			r := newJWKSResolver(client, staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))

			_, err := r.ResolveKey(context.Background(), testKID)
			require.NotErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorIs(t, err, errBoom)
		})

		t.Run("cooldown 中の再取得は再fetchせず直近の失敗原因を伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			// Times(1): 2 回目は cooldown で throttle され Do を呼ばない（直近エラーを伝播）。
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil, errBoom).Times(1)
			r := newJWKSResolver(client, staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))

			_, err1 := r.ResolveKey(context.Background(), testKID)
			require.ErrorIs(t, err1, errBoom)

			_, err2 := r.ResolveKey(context.Background(), testKID)
			require.NotErrorIs(t, err2, ErrJWTAuthenticatorInvalidToken)
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
			r := newJWKSResolver(client, staticURL(jwksTestURL), time.Hour, nil, 0, clk)

			_, err1 := r.ResolveKey(context.Background(), testKID)
			require.ErrorIs(t, err1, errBoom)

			clk.advance(jwksRefreshCooldown + time.Second)
			k, err2 := r.ResolveKey(context.Background(), testKID)
			require.NoError(t, err2)
			assert.Equal(t, &key.PublicKey, k)
		})

		t.Run("再取得に成功しても要求 kid が JWKS に無い場合は拒否する", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			r := newJWKSResolver(
				stubJWKSClient(t, jwksWithKey(t, testKID, &key.PublicKey)),
				staticURL(jwksTestURL),
				time.Hour,
				nil,
				0,
				newFakeClock(fixedNow),
			)

			_, err := r.ResolveKey(context.Background(), "absent-kid")
			require.NotErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorContains(t, err, "no matching JWKS key")
		})

		t.Run("直近取得が成功なら cooldown 中の未知 kid は再取得せず拒否する", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			// Times(1): 初回取得のみ。cooldown 中の未知 kid は再取得しない。
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: jwksWithKey(t, testKID, &key.PublicKey)}, nil).
				Times(1)
			r := newJWKSResolver(client, staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))

			_, err := r.ResolveKey(context.Background(), testKID)
			require.NoError(t, err)
			_, err = r.ResolveKey(context.Background(), "unknown-kid")
			require.NotErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorContains(t, err, "no matching JWKS key")
		})
	})
}

func Test_jwksResolver_negativeCache(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不在確定 kid は cooldown 経過後も再取得せず拒否する", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			cc := &countingJWKSClient{bodies: [][]byte{jwksWithKey(t, testKID, &key.PublicKey)}}
			clk := newFakeClock(fixedNow)
			r := newJWKSResolver(cc, staticURL(jwksTestURL), time.Hour, nil, 0, clk)

			// 初回の未知 kid は 1 度だけ再取得し、不在を確定して negative へ記録する。
			_, err := r.ResolveKey(context.Background(), "absent-kid")
			require.NotErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorContains(t, err, "no matching JWKS key")
			require.Equal(t, 1, cc.count())

			// cooldown を跨いでも、同一の不在確定 kid は再取得しない（negative cache が抑止する）。
			clk.advance(jwksRefreshCooldown + time.Second)
			_, err = r.ResolveKey(context.Background(), "absent-kid")
			require.NotErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorContains(t, err, "kid known-absent")
			assert.Equal(t, 1, cc.count(), "不在確定 kid は cooldown 経過後も再取得しない")
		})

		t.Run("世代交代（cacheTTL 経過）で negative がクリアされ回転追加の kid を解決する", func(t *testing.T) {
			t.Parallel()
			keyA := newRSAKey(t)
			keyB := newRSAKey(t)
			cc := &countingJWKSClient{bodies: [][]byte{
				jwksWithKey(t, testKID, &keyA.PublicKey),
				jwksWithKey(t, "kid-2", &keyB.PublicKey),
			}}
			clk := newFakeClock(fixedNow)
			r := newJWKSResolver(cc, staticURL(jwksTestURL), time.Hour, nil, 0, clk)

			// 現世代 {testKID} で kid-2 は不在確定（negative 記録）。
			_, err := r.ResolveKey(context.Background(), "kid-2")
			require.NotErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorContains(t, err, "no matching JWKS key")
			require.Equal(t, 1, cc.count())

			// cacheTTL 経過で世代が失効 → 再取得で鍵集合が変わり negative はクリアされ kid-2 を解決する。
			clk.advance(time.Hour + time.Minute)
			kB, err := r.ResolveKey(context.Background(), "kid-2")
			require.NoError(t, err)
			assert.Equal(t, &keyB.PublicKey, kB)
			assert.Equal(t, 2, cc.count())
		})

		t.Run("cooldown 中に問い合わせた kid は negative 記録されず、cooldown 明けの再取得で回転追加を解決する", func(t *testing.T) {
			t.Parallel()
			keyA := newRSAKey(t)
			keyB := newRSAKey(t)
			cc := &countingJWKSClient{bodies: [][]byte{
				jwksWithKey(t, testKID, &keyA.PublicKey),
				jwksWith2Keys(t, testKID, &keyA.PublicKey, "kid-2", &keyB.PublicKey),
			}}
			clk := newFakeClock(fixedNow)
			r := newJWKSResolver(cc, staticURL(jwksTestURL), time.Hour, nil, 0, clk)

			// 初回取得で {testKID} をキャッシュ。
			_, err := r.ResolveKey(context.Background(), testKID)
			require.NoError(t, err)
			require.Equal(t, 1, cc.count())

			// cooldown 中に未知 kid-2 を問い合わせる → throttle で未取得。不在確定として記録してはならない。
			clk.advance(jwksRefreshCooldown / 2)
			_, err = r.ResolveKey(context.Background(), "kid-2")
			require.NotErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorContains(t, err, "no matching JWKS key")
			require.Equal(t, 1, cc.count(), "cooldown 中は再取得しない")

			// cooldown 明け（provider が kid-2 を追加）→ 再取得して kid-2 を解決できる（取りこぼさない）。
			clk.advance(jwksRefreshCooldown)
			kB, err := r.ResolveKey(context.Background(), "kid-2")
			require.NoError(t, err, "cooldown 中の問い合わせで negative に閉じ込められない")
			assert.Equal(t, &keyB.PublicKey, kB)
			assert.Equal(t, 2, cc.count())
		})

		t.Run("鍵集合が変わらない再取得では negative を保持し bogus kid を再取得しない", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			cc := &countingJWKSClient{bodies: [][]byte{jwksWithKey(t, testKID, &key.PublicKey)}}
			clk := newFakeClock(fixedNow)
			r := newJWKSResolver(cc, staticURL(jwksTestURL), time.Hour, nil, 0, clk)

			// 初回取得で bogus kid の不在を確定（negative 記録）。
			_, err := r.ResolveKey(context.Background(), "bogus")
			require.NotErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorContains(t, err, "no matching JWKS key")
			require.Equal(t, 1, cc.count())

			// cacheTTL 経過で世代失効 → 同一集合を再取得（negative は破棄されない）。
			clk.advance(time.Hour + time.Minute)
			_, err = r.ResolveKey(context.Background(), testKID)
			require.NoError(t, err)
			require.Equal(t, 2, cc.count(), "TTL 失効で 1 度だけ再取得する")

			// 集合不変のため negative は保持され、bogus kid は再取得を誘発しない。
			_, err = r.ResolveKey(context.Background(), "bogus")
			require.NotErrorIs(t, err, ErrJWTAuthenticatorInvalidToken)
			require.ErrorContains(t, err, "kid known-absent")
			assert.Equal(t, 2, cc.count(), "集合不変なら bogus kid の negative を保持し再取得しない")
		})

		t.Run("同時の未知 kid 解決は singleflight + cooldown で fetch を 1 回に抑える", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			cc := &countingJWKSClient{bodies: [][]byte{jwksWithKey(t, testKID, &key.PublicKey)}}
			r := newJWKSResolver(cc, staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))

			var wg sync.WaitGroup
			for range 8 {
				wg.Go(func() {
					_, _ = r.ResolveKey(context.Background(), "unknown-kid")
				})
			}
			wg.Wait()

			assert.Equal(t, 1, cc.count(), "同時の未知 kid 更新は 1 回の fetch に抑制される")
		})
	})
}

func Test_jwksResolver_negativelyCached(t *testing.T) {
	t.Parallel()
	// negativelyCached の分岐（fetchedAt zero / TTL 切れ / negative 登録有無）は
	// Test_jwksResolver_negativeCache が ResolveKey 経由で網羅している。命名規約充足のためのスキップ。
	t.Skip("Test_jwksResolver_negativeCache が振る舞い経由で網羅している")
}

func Test_jwksResolver_recordAbsent(t *testing.T) {
	t.Parallel()
	// recordAbsent（実取得世代でのみ記録・鮮度外は非記録）は Test_jwksResolver_negativeCache が
	// ResolveKey 経由で網羅している。命名規約充足のためのスキップ。
	t.Skip("Test_jwksResolver_negativeCache が振る舞い経由で網羅している")
}

func Test_sameKeySet(t *testing.T) {
	t.Parallel()

	keyA := newRSAKey(t)
	keyB := newRSAKey(t)
	base := map[string]crypto.PublicKey{testKID: &keyA.PublicKey, "kid-2": &keyB.PublicKey}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一 kid 集合は true", func(t *testing.T) {
			t.Parallel()
			other := map[string]crypto.PublicKey{"kid-2": &keyB.PublicKey, testKID: &keyA.PublicKey}
			assert.True(t, sameKeySet(base, other))
		})

		t.Run("kid が増減すると false", func(t *testing.T) {
			t.Parallel()
			assert.False(t, sameKeySet(base, map[string]crypto.PublicKey{testKID: &keyA.PublicKey}))
		})

		t.Run("要素数が同じでも kid が異なると false", func(t *testing.T) {
			t.Parallel()
			other := map[string]crypto.PublicKey{testKID: &keyA.PublicKey, "kid-3": &keyB.PublicKey}
			assert.False(t, sameKeySet(base, other))
		})
	})
}

func Test_parseJWKSKeys(t *testing.T) {
	t.Parallel()
	// parseJWKSKeys は TestParseJWKSKeys が全分岐（kid 有無 / 非 RSA / 不正 JSON / 鍵ゼロ）を
	// 直接呼び出しで網羅済みのため、専用テストは重複となる。命名規約充足のためのスキップ。
	t.Skip("parseJWKSKeys は TestParseJWKSKeys が直接呼び出しで網羅している")
}

func Test_newJWKSResolver(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cacheTTL が正なら指定値をそのまま保持する", func(t *testing.T) {
			t.Parallel()
			r := newJWKSResolver(stubJWKSClient(t, nil), staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))
			assert.Equal(t, time.Hour, r.cacheTTL)
		})

		t.Run("cacheTTL が 0 以下なら既定 TTL が適用される", func(t *testing.T) {
			t.Parallel()
			r := newJWKSResolver(stubJWKSClient(t, nil), staticURL(jwksTestURL), 0, nil, 0, newFakeClock(fixedNow))
			assert.Equal(t, defaultJWKSCacheTTL, r.cacheTTL)
		})

		t.Run("cooldown が 0 以下なら既定 cooldown が適用される", func(t *testing.T) {
			t.Parallel()
			r := newJWKSResolver(stubJWKSClient(t, nil), staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))
			assert.Equal(t, jwksRefreshCooldown, r.cooldown)
		})

		t.Run("allowedAlgs が空なら既定の許可アルゴリズムが適用される", func(t *testing.T) {
			t.Parallel()
			r := newJWKSResolver(stubJWKSClient(t, nil), staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))
			assert.Equal(t, defaultAllowedAlgs, r.allowedAlgs)
		})
	})
}

func Test_jwksResolver_lookup(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("鮮度内キャッシュに kid があれば対応する鍵を返す", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			r := newJWKSResolver(stubJWKSClient(t, nil), staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))
			r.keys = map[string]crypto.PublicKey{testKID: &key.PublicKey}
			r.fetchedAt = fixedNow

			assert.Equal(t, &key.PublicKey, r.lookup(testKID))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一度も取得していない（fetchedAt がゼロ値）場合は nil を返す", func(t *testing.T) {
			t.Parallel()
			r := newJWKSResolver(stubJWKSClient(t, nil), staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))
			assert.Nil(t, r.lookup(testKID))
		})

		t.Run("TTL を超過して鮮度切れの場合は nil を返す", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			clk := newFakeClock(fixedNow)
			r := newJWKSResolver(stubJWKSClient(t, nil), staticURL(jwksTestURL), time.Hour, nil, 0, clk)
			r.keys = map[string]crypto.PublicKey{testKID: &key.PublicKey}
			r.fetchedAt = fixedNow
			clk.advance(time.Hour + time.Minute)

			assert.Nil(t, r.lookup(testKID))
		})

		t.Run("鮮度内でも未知の kid は nil を返す", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			r := newJWKSResolver(stubJWKSClient(t, nil), staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))
			r.keys = map[string]crypto.PublicKey{testKID: &key.PublicKey}
			r.fetchedAt = fixedNow

			assert.Nil(t, r.lookup("unknown-kid"))
		})
	})
}

func Test_jwksResolver_refresh(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得成功時は keys / fetchedAt を更新する", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			r := newJWKSResolver(
				stubJWKSClient(t, jwksPublicJSON(t, &key.PublicKey)),
				staticURL(jwksTestURL),
				time.Hour,
				nil,
				0,
				newFakeClock(fixedNow),
			)

			fetched, err := r.refresh(context.Background())
			require.NoError(t, err)
			assert.True(t, fetched)
			assert.Equal(t, fixedNow, r.fetchedAt)
			assert.Contains(t, r.keys, testKID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得失敗時は原因を返し lastErr へ保持する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil, errBoom).Times(1)
			r := newJWKSResolver(client, staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))

			_, err := r.refresh(context.Background())
			require.ErrorIs(t, err, errBoom)
			assert.True(t, r.fetchedAt.IsZero())
		})

		t.Run("cooldown 中の再取得は再fetchせず直近の失敗原因を伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			// Times(1): 2 回目は cooldown で throttle され Do を呼ばない。
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil, errBoom).Times(1)
			r := newJWKSResolver(client, staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))

			fetched1, err1 := r.refresh(context.Background())
			require.ErrorIs(t, err1, errBoom)
			assert.True(t, fetched1, "初回は実 fetch する")
			fetched2, err2 := r.refresh(context.Background())
			require.ErrorIs(t, err2, errBoom)
			assert.False(t, fetched2, "cooldown 中は throttle され fetch しない")
		})

		t.Run("呼び出し元 ctx のキャンセルは cooldown を汚染せず次回即再取得できる", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			// 1 回目 ctx キャンセル → 2 回目成功の順で応答する。
			firstFetch := client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(nil, xerrors.Wrap(apperror.ErrCanceled, "canceled")).Times(1)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: jwksPublicJSON(t, &key.PublicKey)}, nil).
				Times(1).After(firstFetch)
			r := newJWKSResolver(client, staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))

			fetchedCanceled, errCanceled := r.refresh(context.Background())
			require.ErrorIs(t, errCanceled, apperror.ErrCanceled)
			assert.False(t, fetchedCanceled, "キャンセルは未取得扱い（negative へ載せない）")
			_, err := r.refresh(context.Background())
			require.NoError(t, err)
		})
	})
}

func Test_jwksResolver_fetch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得したレスポンスボディを kid→公開鍵へパースする", func(t *testing.T) {
			t.Parallel()
			key := newRSAKey(t)
			r := newJWKSResolver(
				stubJWKSClient(t, jwksPublicJSON(t, &key.PublicKey)),
				staticURL(jwksTestURL),
				time.Hour,
				nil,
				0,
				newFakeClock(fixedNow),
			)

			keys, err := r.fetch(context.Background())
			require.NoError(t, err)
			assert.Contains(t, keys, testKID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("httpclient がエラーを返す場合はそのまま原因を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil, errBoom).Times(1)
			r := newJWKSResolver(client, staticURL(jwksTestURL), time.Hour, nil, 0, newFakeClock(fixedNow))

			keys, err := r.fetch(context.Background())
			assert.Nil(t, keys)
			require.ErrorIs(t, err, errBoom)
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
