package jwt

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"

	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/pkg/xerrors"
)

const (
	// defaultDiscoveryTTL は、OIDC discovery 文書を再取得するまでの既定の保持期間です。
	// jwks_uri の変更（IdP 構成変更）は稀のため、鍵キャッシュ TTL より長く取ります。
	defaultDiscoveryTTL = 24 * time.Hour
	// discoveryPath は、issuer からの OIDC discovery 文書の相対パスです（OpenID Connect Discovery §4）。
	discoveryPath = "/.well-known/openid-configuration"
)

var (
	// errDiscoveryIssuerMismatch は、discovery 応答の issuer が config の issuer と一致しない場合のエラーです。
	errDiscoveryIssuerMismatch = xerrors.New("discovery: issuer mismatch")
	// errDiscoveryInsecureURL は、discovery URL / jwks_uri が https でない場合のエラーです。
	errDiscoveryInsecureURL = xerrors.New("discovery: non-https url")
	// errDiscoveryUntrustedJWKS は、jwks_uri が issuer と同一オリジンでない場合のエラーです。
	errDiscoveryUntrustedJWKS = xerrors.New("discovery: jwks_uri outside issuer origin")
	// errDiscoveryNoJWKSURI は、discovery 応答に jwks_uri が無い場合のエラーです。
	errDiscoveryNoJWKSURI = xerrors.New("discovery: missing jwks_uri")
	// errDiscoveryMalformed は、discovery 応答(JSON) の解釈に失敗した場合のエラーです。
	errDiscoveryMalformed = xerrors.New("discovery: malformed document")
)

// openidConfiguration は、OIDC discovery 応答のうち本実装が用いる最小フィールドです。
type openidConfiguration struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"` //nolint:tagliatelle // OIDC Discovery の応答フィールドは snake_case が仕様
}

// discoveryResolver は、issuer から OIDC discovery 経由で jwks_uri を解決します。
// 取得結果を discovery TTL でキャッシュし、issuer 厳密一致・https・同一オリジンを検証します（信頼境界外の応答として）。
type discoveryResolver struct {
	client        httpclient.Client
	issuer        string
	discoveryURL  string
	ttl           time.Duration
	clk           clock.Clock
	allowInsecure bool

	// fetchMu は discovery 取得を直列化し、同時に高々 1 回だけ HTTP させます（stampede 防止）。
	fetchMu sync.Mutex

	// mu は jwksURI / fetchedAt を保護します（取得の HTTP I/O 中は保持しません）。
	mu        sync.RWMutex
	jwksURI   string
	fetchedAt time.Time
}

// newDiscoveryResolver は、issuer ベースの discovery 解決器を生成します（遅延取得）。
func newDiscoveryResolver(client httpclient.Client, issuer string, ttl time.Duration, clk clock.Clock, allowInsecure bool) *discoveryResolver {
	if ttl <= 0 {
		ttl = defaultDiscoveryTTL
	}
	return &discoveryResolver{
		client:        client,
		issuer:        issuer,
		discoveryURL:  strings.TrimRight(issuer, "/") + discoveryPath,
		ttl:           ttl,
		clk:           clk,
		allowInsecure: allowInsecure,
	}
}

// jwksURL は、キャッシュ済みの jwks_uri を返します。鮮度切れなら discovery を再取得・検証します。
func (d *discoveryResolver) jwksURL(ctx context.Context) (string, error) {
	if u := d.cached(); u != "" {
		return u, nil
	}
	return d.refresh(ctx)
}

// cached は、鮮度内のキャッシュから jwks_uri を返します（無ければ空文字）。
func (d *discoveryResolver) cached() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.fetchedAt.IsZero() || d.clk.Now().Sub(d.fetchedAt) >= d.ttl {
		return ""
	}
	return d.jwksURI
}

// refresh は、discovery を取得・検証し jwks_uri をキャッシュします。HTTP I/O 中は mu を保持しません。
func (d *discoveryResolver) refresh(ctx context.Context) (string, error) {
	d.fetchMu.Lock()
	defer d.fetchMu.Unlock()

	// 待機中に他 goroutine が更新済みの可能性があるため、取得前に再確認する（二重取得の回避）。
	if u := d.cached(); u != "" {
		return u, nil
	}

	jwksURI, err := d.fetch(ctx)
	if err != nil {
		return "", err
	}

	now := d.clk.Now()
	d.mu.Lock()
	d.jwksURI = jwksURI
	d.fetchedAt = now
	d.mu.Unlock()
	return jwksURI, nil
}

// fetch は、discovery 文書を取得し、issuer 厳密一致 / https / 同一オリジンを検証して jwks_uri を返します。
func (d *discoveryResolver) fetch(ctx context.Context) (string, error) {
	if err := requireSecureURL(d.discoveryURL, d.allowInsecure); err != nil {
		return "", err
	}

	resp, err := d.client.Do(ctx, httpclient.NewRequest(httpclient.MethodGet(), jwksDownstream, d.discoveryURL))
	if err != nil {
		return "", err
	}

	var doc openidConfiguration
	if err := json.Unmarshal(resp.Body, &doc); err != nil {
		return "", xerrors.Join(errDiscoveryMalformed, err)
	}

	// issuer 厳密一致（OpenID Connect Discovery §4.3 MUST）。応答侵害時に別 issuer を信頼させない。
	if doc.Issuer != d.issuer {
		return "", xerrors.Wrap(errDiscoveryIssuerMismatch, doc.Issuer)
	}
	if strings.TrimSpace(doc.JWKSURI) == "" {
		return "", errDiscoveryNoJWKSURI
	}
	if err := requireSecureURL(doc.JWKSURI, d.allowInsecure); err != nil {
		return "", err
	}
	// jwks_uri は応答由来のため、issuer と同一オリジンに限定して任意 URL への誘導を防ぐ。
	if err := d.requireSameOrigin(doc.JWKSURI); err != nil {
		return "", err
	}
	return doc.JWKSURI, nil
}

// requireSecureURL は、URL が https であることを要求します（allowInsecure が true の場合は検証しません）。
func requireSecureURL(rawURL string, allowInsecure bool) error {
	if allowInsecure {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return xerrors.Join(errDiscoveryInsecureURL, err)
	}
	if u.Scheme != "https" {
		return xerrors.Wrap(errDiscoveryInsecureURL, rawURL)
	}
	return nil
}

// requireSameOrigin は、jwks_uri が issuer と同一オリジン（scheme + host）であることを要求します。
func (d *discoveryResolver) requireSameOrigin(rawJWKS string) error {
	iss, err := url.Parse(d.issuer)
	if err != nil {
		return xerrors.Join(errDiscoveryUntrustedJWKS, err)
	}
	j, err := url.Parse(rawJWKS)
	if err != nil {
		return xerrors.Join(errDiscoveryUntrustedJWKS, err)
	}
	if iss.Scheme != j.Scheme || iss.Host != j.Host {
		return xerrors.Wrap(errDiscoveryUntrustedJWKS, rawJWKS)
	}
	return nil
}
