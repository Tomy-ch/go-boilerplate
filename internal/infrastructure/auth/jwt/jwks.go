package jwt

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/json"
	"slices"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/pkg/xerrors"
)

const (
	// defaultJWKSCacheTTL は、取得した JWKS を再取得するまでの既定の保持期間です（config 既定 AUTH_JWKS_CACHE_TTL と同値）。
	defaultJWKSCacheTTL = 1 * time.Hour
	// jwksRefreshCooldown は、cache miss（未知 kid / 期限切れ / 取得失敗）時の再取得の最小間隔です（config 既定 AUTH_JWKS_UNKNOWN_KID_COOLDOWN と同値）。
	// 未知 kid の連打抑止に加え、取得失敗後のリトライ間隔も兼ねます（DoS 抑止）。
	jwksRefreshCooldown = 60 * time.Second
	// jwksDownstream は、httpclient substrate の論理依存名です（profile / breaker / budget / metrics のキー）。
	jwksDownstream httpclient.Downstream = "jwks"

	// jwksOverallTimeout は、JWKS/discovery 取得の全体タイムアウトです（substrate 既定 10s から短縮）。
	// 認証は未知 kid 時に同期ブロックするホットパスのため、全体上限を短くする。
	jwksOverallTimeout = 5 * time.Second
	// jwksMaxAttempts は、JWKS/discovery 取得の最大試行回数です（初回 + retry）。
	jwksMaxAttempts = 2
	// jwksMaxResponseBytes は、読み込む JWKS/discovery 応答の上限バイト数です（substrate 既定 4 MiB から縮小）。
	// JWKS/discovery 文書は KB オーダーのため、メモリ DoS を抑制する。
	jwksMaxResponseBytes = 1 << 20 // 1 MiB
)

var (
	// errJWKSMalformed は、JWKS(JSON) の解釈に失敗した場合のエラーです。
	errJWKSMalformed = xerrors.New("jwks: malformed key set")
	// errJWKSNoKeys は、JWKS に利用可能な鍵（kid 付き）が無い場合のエラーです。
	errJWKSNoKeys = xerrors.New("jwks: no usable keys with kid")
	// errJWKSDuplicateKID は、JWKS に重複した kid が含まれる場合のエラーです（文書ごと不採用）。
	errJWKSDuplicateKID = xerrors.New("jwks: duplicate kid")
	// errJWKSKIDKnownAbsent は、現世代の JWKS で不在が確定済みの kid を要求された場合のエラーです。
	errJWKSKIDKnownAbsent = xerrors.New("jwks: kid known-absent in current key set")
	// errJWKSNoMatchingKID は、取得した JWKS に kid に対応する鍵が無い場合のエラーです。
	errJWKSNoMatchingKID = xerrors.New("jwks: no matching key for kid")
)

// jwksResolver は、JWKS エンドポイントから kid で公開鍵を解決する KeyResolver 実装です。
// 取得は httpclient に委ね、取得した JWK Set を TTL でキャッシュします。期限切れ・未知 kid のときに再取得します（遅延取得）。
type jwksResolver struct {
	client      httpclient.Client
	tracer      observability.LayerTracer
	urlFn       func(context.Context) (string, error)
	cacheTTL    time.Duration
	cooldown    time.Duration
	allowedAlgs []string
	clk         clock.Clock

	// fetchMu は取得を直列化し、同時に高々 1 回だけ HTTP させます（stampede 防止）。
	fetchMu sync.Mutex

	// mu は keys / fetchedAt / lastAttempt / lastErr / negative を保護します（取得の HTTP I/O 中は保持しません）。
	mu          sync.RWMutex
	keys        map[string]crypto.PublicKey
	fetchedAt   time.Time
	lastAttempt time.Time
	lastErr     error

	// negative は「取得済みの現世代で不在が確定した kid」の集合です（不在 kid の再取得連打を抑止）。
	// 現世代（fetchedAt から cacheTTL 以内）でのみ有効で、公開鍵集合が変わる取得成功でクリアします。
	// これにより、回転で新規追加された kid は negative に載らないため次の取得で解決でき、確定的に不在な kid の再問い合わせだけを抑えます。
	negative map[string]struct{}
}

// NewDownstreamProfile は、JWKS/discovery 取得向けの httpclient resilient プロファイルを返します。
// 外部 IdP へ内部相関 ID を漏らさないよう trace は伝搬しません。
// allowPrivateNetwork は private 網宛て接続の可否で、環境に応じた解決は DI が行います（infra は env を知らない）。
// 認証ホットパス向けに全体タイムアウト・試行回数・応答上限を substrate 既定より絞ります。
func NewDownstreamProfile(allowPrivateNetwork bool) httpclient.DownstreamProfile {
	p := httpclient.DefaultProfile()
	p.PropagateTrace = false
	p.AllowPrivateNetwork = allowPrivateNetwork
	p.OverallTimeout = jwksOverallTimeout
	p.MaxAttempts = jwksMaxAttempts
	p.MaxResponseBytes = jwksMaxResponseBytes
	return httpclient.DownstreamProfile{Name: jwksDownstream, Profile: p}
}

// RequiredDownstream は、JWKS 取得に用いる Downstream を返します。
// required_downstreams へ供給することで、起動失敗になることを保証します。
func RequiredDownstream() httpclient.Downstream {
	return jwksDownstream
}

// newJWKSResolver は、JWKS 解決器を生成します（この時点では取得しません＝遅延取得）。
// cacheTTL / cooldown が非正、allowedAlgs が空のときは、それぞれ既定値へフォールバックします。
func newJWKSResolver(
	client httpclient.Client,
	urlFn func(context.Context) (string, error),
	cacheTTL time.Duration,
	allowedAlgs []string,
	cooldown time.Duration,
	clk clock.Clock,
	tf observability.TracerFactory,
) *jwksResolver {
	if tf == nil {
		tf = observability.NewDisabledTracerFactory()
	}
	if cacheTTL <= 0 {
		cacheTTL = defaultJWKSCacheTTL
	}
	if cooldown <= 0 {
		cooldown = jwksRefreshCooldown
	}
	if len(allowedAlgs) == 0 {
		allowedAlgs = defaultAllowedAlgs
	}
	return &jwksResolver{
		client:      client,
		tracer:      tf.Infra(),
		urlFn:       urlFn,
		cacheTTL:    cacheTTL,
		cooldown:    cooldown,
		allowedAlgs: allowedAlgs,
		clk:         clk,
		keys:        map[string]crypto.PublicKey{},
	}
}

// ResolveKey は、kid に対応する署名検証用公開鍵を返します（KeyResolver 実装）。
// キャッシュに無い / 期限切れの場合は JWKS を再取得します。
// 返すエラーは keyFunc → ParseWithClaims 経由で Authenticate の境界へ伝播し、そこで
// ErrJWTAuthenticatorInvalidToken へ一括正規化するため、ここでは原因のみを持つ素の error を返します。
func (r *jwksResolver) ResolveKey(ctx context.Context, kid string) (crypto.PublicKey, error) {
	if key := r.lookup(kid); key != nil {
		return key, nil
	}
	// 現世代で不在が確定済みの kid は再取得せず即座に拒否する（再取得連打の抑止）。
	if r.negativelyCached(kid) {
		return nil, errJWKSKIDKnownAbsent
	}
	fetched, err := r.refresh(ctx)
	if err != nil {
		return nil, err
	}
	if key := r.lookup(kid); key != nil {
		return key, nil
	}
	// 実際に取得した世代でのみ不在を確定する。cooldown で未取得（throttle）のときに記録すると、
	// その後 cooldown が明けても cacheTTL 満了まで再取得がブロックされ、回転追加の kid を取りこぼす。
	if fetched {
		r.recordAbsent(kid)
	}
	return nil, errJWKSNoMatchingKID
}

// negativelyCached は、現世代（鮮度内）で kid が不在確定として記録済みかを返します。
// キャッシュが空 / 期限切れのとき、および kid が negative に未記録のときも false を返します。
// false は「再取得で解決を試みるべき」を意味し kid の存在は保証しません（退役後の再評価・回転追加の取りこぼし防止）。
func (r *jwksResolver) negativelyCached(kid string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.fetchedAt.IsZero() || r.clk.Now().Sub(r.fetchedAt) >= r.cacheTTL {
		return false
	}
	_, ok := r.negative[kid]
	return ok
}

// recordAbsent は、現世代で不在が確定した kid を negative へ記録します。
// 鮮度外（fetchedAt ゼロ / cacheTTL 超過）では記録しません — 次の世代交代で negative がクリアされるまで残り、
// その間に回転追加された kid を誤って拒否し続けるためです。
func (r *jwksResolver) recordAbsent(kid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fetchedAt.IsZero() || r.clk.Now().Sub(r.fetchedAt) >= r.cacheTTL {
		return
	}
	if r.negative == nil {
		r.negative = map[string]struct{}{}
	}
	r.negative[kid] = struct{}{}
}

// lookup は、鮮度内のキャッシュから kid に対応する鍵を返します（無ければ nil）。
func (r *jwksResolver) lookup(kid string) crypto.PublicKey {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.fetchedAt.IsZero() || r.clk.Now().Sub(r.fetchedAt) >= r.cacheTTL {
		return nil
	}
	return r.keys[kid]
}

// refresh は、cooldown を尊重して JWKS を再取得します。HTTP I/O 中は mu を保持しません。
// 第 1 戻り値は実際に HTTP 取得を行ったかを表し、cooldown 中の throttle・ctx キャンセルでは false になります
// （呼び出し元が「取得済み世代での不在確定」と「未取得」を区別し、後者を negative へ誤記録しないため）。
func (r *jwksResolver) refresh(ctx context.Context) (bool, error) {
	r.fetchMu.Lock()
	defer r.fetchMu.Unlock()

	r.mu.RLock()
	throttled := !r.lastAttempt.IsZero() && r.clk.Now().Sub(r.lastAttempt) < r.cooldown
	lastErr := r.lastErr
	r.mu.RUnlock()
	if throttled {
		// cooldown 中は再取得せず直近取得の結果を伝播する。
		// 直近が失敗なら原因を返し（インフラ障害を「無効トークン」と誤認させない）、成功なら nil。
		return false, lastErr
	}

	keys, err := r.fetch(ctx)

	// 呼び出し元 ctx のキャンセルは共有キャッシュ（lastErr/lastAttempt）へ載せない。
	// 1 リクエストの切断が cooldown 中の全リクエストへ「無効トークン」として波及するのを防ぐ。
	if err != nil && xerrors.Is(err, apperror.ErrCanceled) {
		return false, err
	}

	now := r.clk.Now()
	r.mu.Lock()
	r.lastAttempt = now
	r.lastErr = err
	if err == nil {
		// 公開鍵集合が変わった取得成功では、不在確定（negative）を破棄して再評価する（回転追加の取りこぼし防止）。
		// 集合が不変なら negative は現世代の判断として保持する（同一 bogus kid の再取得抑止を維持）。
		if !sameKeySet(r.keys, keys) {
			r.negative = nil
		}
		r.keys = keys
		r.fetchedAt = now
	}
	r.mu.Unlock()

	return true, err
}

// sameKeySet は、2 つの鍵マップが同一の kid 集合を持つかを返します。
func sameKeySet(a, b map[string]crypto.PublicKey) bool {
	if len(a) != len(b) {
		return false
	}
	for kid := range a {
		if _, ok := b[kid]; !ok {
			return false
		}
	}
	return true
}

// fetch は、httpclient substrate 経由で JWKS を取得し、kid→公開鍵のマップへ変換します。
// タイムアウト・キャンセルは ctx と substrate の profile が担います。
func (r *jwksResolver) fetch(ctx context.Context) (map[string]crypto.PublicKey, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	url, err := r.urlFn(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(ctx, httpclient.NewRequest(httpclient.MethodGet(), jwksDownstream, url))
	if err != nil {
		return nil, err
	}
	return parseJWKSKeys(resp.Body, r.allowedAlgs)
}

// parseJWKSKeys は、JWKS(JSON) を kid→公開鍵のマップへパースします。
// 署名用途（use=sig）の KeyID 付き RSA 公開鍵のみ採用し、JWK が alg を宣言している場合は
// allowedAlgs（許可署名アルゴリズム）に含まれるものだけ採用します（RFC 7517 §4.4）。
// 1 件も無ければエラーを返します。
func parseJWKSKeys(data []byte, allowedAlgs []string) (map[string]crypto.PublicKey, error) {
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, xerrors.Join(errJWKSMalformed, err)
	}

	keys := make(map[string]crypto.PublicKey, len(set.Keys))
	for i := range set.Keys {
		k := set.Keys[i]
		// 署名用途以外（use=="enc" 等）は採用しない。use 未指定は RFC 7517 上 optional のため許容する。
		if k.Use != "" && k.Use != "sig" {
			continue
		}
		// JWK が alg を宣言している場合は許可アルゴリズムに限定する（未宣言は許容）。
		if k.Algorithm != "" && !slices.Contains(allowedAlgs, k.Algorithm) {
			continue
		}
		pub, ok := k.Key.(*rsa.PublicKey)
		if k.KeyID == "" || !ok {
			continue
		}
		// 重複 kid は silent な last-wins を避け、文書ごと不採用にする（攻撃・設定事故の兆候として loud に失敗する）。
		if _, dup := keys[k.KeyID]; dup {
			return nil, xerrors.Wrap(errJWKSDuplicateKID, k.KeyID)
		}
		keys[k.KeyID] = pub
	}
	if len(keys) == 0 {
		return nil, errJWKSNoKeys
	}
	return keys, nil
}
