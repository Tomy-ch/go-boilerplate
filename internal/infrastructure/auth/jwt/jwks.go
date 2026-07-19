package jwt

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	jwtlib "github.com/golang-jwt/jwt/v5"

	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/pkg/xerrors"
)

const (
	// defaultJWKSCacheTTL は、取得した JWKS を再取得するまでの既定の保持期間です。
	defaultJWKSCacheTTL = 5 * time.Minute
	// jwksRefreshCooldown は、cache miss（未知 kid / 期限切れ）時の再取得の最小間隔です。
	// garbage kid の連打による過剰取得（DoS）を防ぐ下限です。
	jwksRefreshCooldown = 3 * time.Second
	// jwksDownstream は、httpclient substrate の論理依存名です（profile / breaker / budget / metrics のキー）。
	jwksDownstream httpclient.Downstream = "jwks"
)

var (
	// errJWKSMalformed は、JWKS(JSON) の解釈に失敗した場合のエラーです。
	errJWKSMalformed = xerrors.New("jwks: malformed key set")
	// errJWKSNoKeys は、JWKS に利用可能な鍵（kid 付き）が無い場合のエラーです。
	errJWKSNoKeys = xerrors.New("jwks: no usable keys with kid")
)

// jwksResolver は、JWKS エンドポイントから kid で公開鍵を解決する jwt.Keyfunc を提供します。
// 取得は httpclient に委ね、取得した JWK Set をTTL でキャッシュします。期限切れ・未知 kid のときに再取得します（遅延取得）。
// 鍵ローテーション（複数 kid の切替）は後続フェーズの範囲で、本実装は単一 kid + TTL 更新を担います。
type jwksResolver struct {
	client   httpclient.Client
	url      string
	cacheTTL time.Duration
	clk      clock.Clock

	// fetchMu は取得を直列化し、同時に高々 1 回だけ HTTP させます（stampede 防止）。
	fetchMu sync.Mutex

	// mu は keys / fetchedAt / lastAttempt / lastErr を保護します（取得の HTTP I/O 中は保持しません）。
	mu          sync.RWMutex
	keys        map[string]any
	fetchedAt   time.Time
	lastAttempt time.Time
	lastErr     error
}

// NewDownstreamProfile は、JWKS 取得向けの httpclient resilient プロファイルを返します。
// JWKS URL は運用者が設定する信頼できるエンドポイント。
// 外部 IdP へ内部相関 ID を漏らさないよう trace は伝搬しません。
func NewDownstreamProfile() httpclient.DownstreamProfile {
	p := httpclient.DefaultProfile()
	p.PropagateTrace = false
	p.AllowPrivateNetwork = true
	return httpclient.DownstreamProfile{Name: jwksDownstream, Profile: p}
}

// RequiredDownstream は、JWKS 取得に用いる Downstream を返します。
// required_downstreams へ供給することで、起動失敗になることを保証します。
func RequiredDownstream() httpclient.Downstream {
	return jwksDownstream
}

// newJWKSResolver は、JWKS 解決器を生成します（この時点では取得しません＝遅延取得）。
func newJWKSResolver(client httpclient.Client, url string, cacheTTL time.Duration, clk clock.Clock) *jwksResolver {
	if cacheTTL <= 0 {
		cacheTTL = defaultJWKSCacheTTL
	}
	return &jwksResolver{
		client:   client,
		url:      url,
		cacheTTL: cacheTTL,
		clk:      clk,
		keys:     map[string]any{},
	}
}

// keyfunc は、token の kid に対応する公開鍵を返す jwt.Keyfunc です。
// キャッシュに無い / 期限切れの場合は JWKS を再取得します。
func (r *jwksResolver) keyfunc(token *jwtlib.Token) (any, error) {
	kid, _ := token.Header["kid"].(string)

	if key := r.lookup(kid); key != nil {
		return key, nil
	}
	if err := r.refresh(); err != nil {
		return nil, xerrors.Join(ErrJWTAuthenticatorInvalidToken, err)
	}
	if key := r.lookup(kid); key != nil {
		return key, nil
	}
	return nil, xerrors.Wrap(ErrJWTAuthenticatorInvalidToken, "no matching JWKS key for kid")
}

// lookup は、鮮度内のキャッシュから kid に対応する鍵を返します（無ければ nil）。
func (r *jwksResolver) lookup(kid string) any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.fetchedAt.IsZero() || r.clk.Now().Sub(r.fetchedAt) >= r.cacheTTL {
		return nil
	}
	return r.keys[kid]
}

// refresh は、cooldown を尊重して JWKS を再取得します。HTTP I/O 中は mu を保持しません。
func (r *jwksResolver) refresh() error {
	r.fetchMu.Lock()
	defer r.fetchMu.Unlock()

	r.mu.RLock()
	throttled := !r.lastAttempt.IsZero() && r.clk.Now().Sub(r.lastAttempt) < jwksRefreshCooldown
	lastErr := r.lastErr
	r.mu.RUnlock()
	if throttled {
		// cooldown 中は再取得せず直近取得の結果を伝播する。
		// 直近が失敗なら原因を返し（インフラ障害を「無効トークン」と誤認させない）、成功なら nil。
		return lastErr
	}

	keys, err := r.fetch()

	now := r.clk.Now()
	r.mu.Lock()
	r.lastAttempt = now
	r.lastErr = err
	if err == nil {
		r.keys = keys
		r.fetchedAt = now
	}
	r.mu.Unlock()

	return err
}

// fetch は、httpclient substrate 経由で JWKS を取得し、kid→公開鍵のマップへ変換します。
// keyfunc は ctx を受け取らないため、取得のタイムアウトは substrate の profile に委ねます。
func (r *jwksResolver) fetch() (map[string]any, error) {
	resp, err := r.client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), jwksDownstream, r.url))
	if err != nil {
		return nil, err
	}
	return parseJWKSKeys(resp.Body)
}

// parseJWKSKeys は、JWKS(JSON) を kid→公開鍵のマップへパースします。
// KeyID を持つ RSA 公開鍵のみ採用します。
// 1 件も無ければエラーを返します。
func parseJWKSKeys(data []byte) (map[string]any, error) {
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, xerrors.Join(errJWKSMalformed, err)
	}

	keys := make(map[string]any, len(set.Keys))
	for i := range set.Keys {
		k := set.Keys[i]
		pub, ok := k.Key.(*rsa.PublicKey)
		if k.KeyID == "" || !ok {
			continue
		}
		keys[k.KeyID] = pub
	}
	if len(keys) == 0 {
		return nil, errJWKSNoKeys
	}
	return keys, nil
}
