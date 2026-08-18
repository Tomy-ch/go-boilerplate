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

	"go-boilerplate/internal/apperror"
	authjwt "go-boilerplate/internal/infrastructure/auth/jwt"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/pkg/xerrors"
)

// JWKS Rotation の E2E。golden JWKS と署名 PEM を //go:embed で取り込み、
// HTTP 境界（Bearer 抽出 → Authenticate → JWKS 解決 → 検証 → 200/401）ごと検証する。
// Phase の定義とリゾルバの契約は docs/design/auth.md §3.4 が持つ。
// testdata を package 配下へ置いているのは、depguard が integration からの os 直読みを
// 禁じており embed 以外に読み込む手段が無いため。時刻は注入 clock で決定的に前進させる。

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

// blockingJWKS は、release されるまで応答を返さない JWKS 供給元。取得回数も数える。
type blockingJWKS struct {
	body    []byte
	entered chan struct{}
	release chan struct{}

	mu    sync.Mutex
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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Phase 遷移で新旧 Token を検証し、退役鍵を拒否する", func(t *testing.T) {
			t.Parallel()
			clk := &rotClock{now: time.Now()}
			src := &rotatingJWKS{body: phase1}
			srv := startRotationServer(t, src, clk)

			// Phase1: key-a 署名 Token を受理。初回の 1 回だけ JWKS を取得する。
			require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyA, rotKIDPrimary, clk)).StatusCode)
			require.Equal(t, 1, src.count())

			require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyA, rotKIDPrimary, clk)).StatusCode)
			require.Equal(t, 1, src.count(), "既知 kid の通常リクエストで再取得しない")

			clk.advance(rotCooldown + time.Second)
			src.setPhase(phase2)
			require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyB, rotKIDRotation, clk)).StatusCode)
			require.Equal(t, 2, src.count(), "未知 kid の初回のみ再取得する")

			require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyA, rotKIDPrimary, clk)).StatusCode)
			require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyB, rotKIDRotation, clk)).StatusCode)
			require.Equal(t, 2, src.count(), "移行期の新旧受理で再取得しない")

			clk.advance(time.Hour + time.Minute)
			src.setPhase(phase3)
			AssertErrorResponse(t, get(srv, mintRotToken(t, keyA, rotKIDPrimary, clk)), http.StatusUnauthorized)
			require.Equal(t, 3, src.count(), "cacheTTL 経過後に再取得して退役を反映する")
			require.Equal(t, http.StatusOK, get(srv, mintRotToken(t, keyB, rotKIDRotation, clk)).StatusCode)
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
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JWKS に存在しない kid の Token を拒否する（unknown-kid / old-key profile 相当）", func(t *testing.T) {
			t.Parallel()
			clk := &rotClock{now: time.Now()}
			src := &rotatingJWKS{body: phase1}
			srv := startRotationServer(t, src, clk)

			// 有効な鍵で署名しても、JWKS に無い kid では鍵解決に失敗し 401。
			token := mintRotToken(t, keyA, "mock-key-retired", clk)
			AssertErrorResponse(t, get(srv, token), http.StatusUnauthorized)
		})
	})
}

func (s *blockingJWKS) Do(ctx context.Context, _ *httpclient.Request) (*httpclient.Response, error) {
	s.mu.Lock()
	s.calls++
	first := s.calls == 1
	s.mu.Unlock()

	if first {
		close(s.entered)
	}
	select {
	case <-ctx.Done():
		// 実 substrate が返す分類に揃える（httpclient.normalizeTransportError と同じ写像）。
		return nil, xerrors.Wrap(apperror.ErrCanceled, "canceled")
	case <-s.release:
		return &httpclient.Response{StatusCode: 200, Body: s.body}, nil
	}
}

func (s *blockingJWKS) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestJWKSRefreshDetachIntegration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クライアントが切断しても JWKS 取得は完走し cooldown が次の取得を抑える", func(t *testing.T) {
			t.Parallel()
			keyA := loadProviderKey(t, rotKIDPrimary)
			clk := &rotClock{now: time.Now()}
			src := &blockingJWKS{body: loadGoldenJWKS(t, 1), entered: make(chan struct{}), release: make(chan struct{})}
			srv := startRotationServer(t, src, clk)

			// 実クライアントが、JWKS 取得の最中に接続を切る。
			ctx, cancel := context.WithCancel(t.Context())
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.baseURL+"/protected", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+mintRotToken(t, keyA, rotKIDPrimary, clk))

			aborted := make(chan struct{})
			go func() {
				defer close(aborted)
				res, doErr := srv.client.Do(req)
				assert.Error(t, doErr, "切断したリクエストは応答を受け取らない")
				assert.Nil(t, res)
			}()

			<-src.entered
			cancel()
			<-aborted

			// 切断がサーバへ伝わる猶予。取得が呼び出し元の ctx を継いでいれば、この間に打ち切られる。
			time.Sleep(200 * time.Millisecond)
			close(src.release)

			// 切断された取得が完走していれば鍵集合はキャッシュ済みで、後続は追加取得なしで検証できる。
			// このリクエストは取得権の獲得で先行取得の完了を待つため、判定は待ち合わせ済みの状態で行われる。
			res := srv.DoJSON(http.MethodGet, "/protected", nil, bearerHeader(mintRotToken(t, keyA, rotKIDPrimary, clk)))
			require.Equal(t, http.StatusOK, res.StatusCode, "切断された取得の結果が次のリクエストで使える")
			assert.Equal(t, 1, src.count(), "cooldown が起動しており再取得しない")
		})
	})
}
