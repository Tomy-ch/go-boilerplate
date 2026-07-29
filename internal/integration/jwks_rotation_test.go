package integration

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"embed"
	"encoding/pem"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authjwt "go-boilerplate/internal/infrastructure/auth/jwt"
	"go-boilerplate/internal/infrastructure/httpclient"
)

// JWKS Rotation E2E（#605 / #585-C）。
//
// provider（mock-auth-server）が各 Phase で公開する golden JWKS と署名 PEM を //go:embed で取り込み、
// 同じ provider の PEM 秘密鍵で署名した Token を、保護付き Echo（GET /protected）へ Bearer で送って
// HTTP 境界（Bearer 抽出 → Authenticate → JWKS 解決 → 検証 → 200/401）ごと検証する。golden と PEM を
// provider と共有することで、手組み mock の silent 乖離を排除する。testdata 配下のコピーは provider の
// gen-jwks-fixtures スクリプトが同一バイトで生成する（depguard が integration からの os 直読みを禁止するため、
// embed 可能な package 配下に置く）。時刻は注入 clock で決定的に前進させる。
//
// | Phase | JWKS 公開 | 署名鍵 |
// | 1 | mock-key-1 | mock-key-1 |
// | 2 | mock-key-1, mock-key-2 | mock-key-2 |
// | 3 | mock-key-2 | mock-key-2 |

const (
	rotKIDPrimary  = "mock-key-1"
	rotKIDRotation = "mock-key-2"

	// rotCooldown は未知 kid 再取得の最小間隔で、JWKS Authenticator の既定（AUTH_JWKS_UNKNOWN_KID_COOLDOWN=60s）と同値。
	// Phase 遷移時に cooldown を跨いで再取得を許すために用いる。
	rotCooldown = 60 * time.Second
)

//go:embed testdata/jwks/phase1.json testdata/jwks/phase2.json testdata/jwks/phase3.json
//go:embed testdata/keys/mock-key-1.pem testdata/keys/mock-key-2.pem
var rotFS embed.FS

// rotClock は手動で前進できる clock.Clock 実装（cache TTL / cooldown / exp 検証を決定的に駆動する）。
type rotClock struct {
	mu  sync.Mutex
	now time.Time
}

// rotatingJWKS は現在の Phase の golden JWKS を返す httpclient.Client（fetch 回数を数える）。
type rotatingJWKS struct {
	mu    sync.Mutex
	body  []byte
	calls int
}

func (c *rotClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *rotClock) Sleep(context.Context, time.Duration) error { return nil }

func (c *rotClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func (s *rotatingJWKS) Do(context.Context, *httpclient.Request) (*httpclient.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return &httpclient.Response{StatusCode: 200, Body: s.body}, nil
}

func (s *rotatingJWKS) setPhase(body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.body = body
}

func (s *rotatingJWKS) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// loadGoldenJWKS は provider が生成した golden JWKS（phase<n>.json）を embed から読む。
func loadGoldenJWKS(t *testing.T, phase int) []byte {
	t.Helper()
	body, err := rotFS.ReadFile(fmt.Sprintf("testdata/jwks/phase%d.json", phase))
	require.NoError(t, err)
	return body
}

// loadProviderKey は provider の PEM 秘密鍵（PKCS#8）を embed から読み、golden の公開鍵に対応する署名鍵を返す。
func loadProviderKey(t *testing.T, kid string) *rsa.PrivateKey {
	t.Helper()
	data, err := rotFS.ReadFile("testdata/keys/" + kid + ".pem")
	require.NoError(t, err)
	block, _ := pem.Decode(data)
	require.NotNil(t, block, "PEM ブロックを decode できる")
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	require.NoError(t, err)
	key, ok := parsed.(*rsa.PrivateKey)
	require.True(t, ok, "RSA 秘密鍵である")
	return key
}

// mintRotToken は clk の現在時刻を基準に、指定 kid・provider 鍵で署名した有効な access token を発行する。
func mintRotToken(t *testing.T, key *rsa.PrivateKey, kid string, clk *rotClock) string {
	t.Helper()
	now := clk.Now()
	claims := jwtlib.MapClaims{
		"iss": jwtTestIssuer,
		"aud": jwtTestAudience,
		"sub": jwtTestSubject,
		"iat": jwtlib.NewNumericDate(now),
		"nbf": jwtlib.NewNumericDate(now),
		"exp": jwtlib.NewNumericDate(now.Add(time.Hour)),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	token.Header["typ"] = jwtAccessTokenType
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

// startRotationServer は golden JWKS を供給する mock と注入 clock で JWKS Authenticator を構築し、
// 保護付き Echo（GET /protected）へ配線して起動する。
func startRotationServer(t *testing.T, src httpclient.Client, clk *rotClock) *Server {
	t.Helper()
	authenticator, err := authjwt.NewJWKS(authjwt.JWKSParams{
		Params: authjwt.Params{
			Issuer:       jwtTestIssuer,
			Audience:     jwtTestAudience,
			ExpectedType: jwtAccessTokenType,
			Clock:        clk,
		},
		JWKSURL:          "http://mock-auth.example.test/jwks.json",
		AllowInsecureURL: true,
	}, src)
	require.NoError(t, err)
	return newProtectedServer(t, authenticator)
}

func TestJWKSRotationIntegration(t *testing.T) {
	t.Parallel()

	keyA := loadProviderKey(t, rotKIDPrimary)
	keyB := loadProviderKey(t, rotKIDRotation)
	phase1, phase2, phase3 := loadGoldenJWKS(t, 1), loadGoldenJWKS(t, 2), loadGoldenJWKS(t, 3)

	// get は Bearer トークンで GET /protected を叩き、HTTP レスポンスを返す。
	get := func(srv *Server, token string) *http.Response {
		return srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(token))
	}

	t.Run("Phase 遷移で新旧 Token を検証し、退役鍵を拒否する", func(t *testing.T) {
		t.Parallel()
		clk := &rotClock{now: time.Now()}
		src := &rotatingJWKS{body: phase1}
		srv := startRotationServer(t, src, clk)

		// Phase1: key-a 署名 Token を受理。初回の 1 回だけ JWKS を取得する。
		require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyA, rotKIDPrimary, clk)).StatusCode)
		require.Equal(t, 1, src.count())

		// 通常リクエスト（既知 kid）では JWKS を再取得しない（受入条件 1）。
		require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyA, rotKIDPrimary, clk)).StatusCode)
		require.Equal(t, 1, src.count(), "既知 kid の通常リクエストで再取得しない")

		// Phase2: 新鍵 key-b を公開集合へ追加。未知 kid の初回で 1 度だけ再取得する（受入条件 2）。
		clk.advance(rotCooldown + time.Second)
		src.setPhase(phase2)
		require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyB, rotKIDRotation, clk)).StatusCode)
		require.Equal(t, 2, src.count(), "未知 kid の初回のみ再取得する")

		// 移行期は新旧両署名 Token を受理し、追加の再取得は起きない（受入条件 3）。
		require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyA, rotKIDPrimary, clk)).StatusCode)
		require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyB, rotKIDRotation, clk)).StatusCode)
		require.Equal(t, 2, src.count(), "移行期の新旧受理で再取得しない")

		// Phase3: 旧鍵 key-a を退役。cacheTTL 経過で再取得し、退役 kid の Token を拒否する（受入条件 4）。
		clk.advance(time.Hour + time.Minute)
		src.setPhase(phase3)
		AssertErrorResponse(t, get(srv, mintRotToken(t, keyA, rotKIDPrimary, clk)), http.StatusUnauthorized)
		require.Equal(t, 3, src.count(), "cacheTTL 経過後に再取得して退役を反映する")
		require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyB, rotKIDRotation, clk)).StatusCode)
	})

	t.Run("JWKS に存在しない kid の Token を拒否する（unknown-kid / old-key profile 相当）", func(t *testing.T) {
		t.Parallel()
		clk := &rotClock{now: time.Now()}
		src := &rotatingJWKS{body: phase1}
		srv := startRotationServer(t, src, clk)

		// 有効な鍵で署名しても、JWKS に無い kid では鍵解決に失敗し 401。
		token := mintRotToken(t, keyA, "mock-key-retired", clk)
		AssertErrorResponse(t, get(srv, token), http.StatusUnauthorized)
	})

	t.Run("同時の未知 kid 検証は JWKS 更新を 1 回に抑制する（受入条件 5）", func(t *testing.T) {
		t.Parallel()
		clk := &rotClock{now: time.Now()}
		src := &rotatingJWKS{body: phase1}
		srv := startRotationServer(t, src, clk)

		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				_ = get(srv, mintRotToken(t, keyB, rotKIDRotation, clk)).Body.Close()
			})
		}
		wg.Wait()

		assert.Equal(t, 1, src.count(), "同時 JWKS 更新は singleflight で 1 回に抑制される")
	})
}
