package observability

import (
	"net/http"

	"go.opentelemetry.io/otel/trace/noop"
)

// maxOutboundRedirects は、追従を許す最大リダイレクト回数です。
const maxOutboundRedirects = 10

// OutboundHTTPClient は、SSRF ガード付き transport を持つ外部 SDK 向けの HTTP クライアントです。
// 自前の substrate（infrastructure/httpclient）を通せない SDK へ渡すための型で、素の *http.Client と
// 取り違えないよう名前を持たせています。
type OutboundHTTPClient struct {
	*http.Client
}

// policyStampingRoundTripper は、通過する全リクエストへ固定の SSRF ガード方針を適用する RoundTripper です。
type policyStampingRoundTripper struct {
	inner               http.RoundTripper
	allowPrivateNetwork bool
}

// NewOutboundHTTPClient は、SSRF ガード付き transport を持つ OutboundHTTPClient を返します。
//
// SDK は独自のリトライとタイムアウトを持つため、ここで与えるのは transport だけで、breaker や
// budget は載せません。
//
// allowPrivateNetwork はこの client を通る全リクエストへ一律に適用します。SDK は呼び出し側の ctx を
// そのままリクエストへ渡すため、呼び出しごとに ctx へ積む形にすると、積み忘れた経路が黙って既定値で
// 通ってしまいます。link-local（クラウドメタデータ等）の拒否はフラグに依らず常に効きます。
//
// 同じ t から方針の異なる client を作らないでください。判定は dial の直前に行う一方、transport は
// コネクションプールを持つため、許可側が張った接続を拒否側が再利用すると判定を経ずに通ります。
func NewOutboundHTTPClient(t *HTTPClientTransport, allowPrivateNetwork bool) *OutboundHTTPClient {
	return &OutboundHTTPClient{
		Client: &http.Client{
			Transport: policyStampingRoundTripper{
				inner:               t.RoundTripper(),
				allowPrivateNetwork: allowPrivateNetwork,
			},
			CheckRedirect: limitedRedirectForOutbound,
		},
	}
}

// RoundTrip は、方針を ctx へ積んだ clone を内側へ委譲します。
func (rt policyStampingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// http.RoundTripper は呼び出し元の Request を変更してはならないため clone する。
	return rt.inner.RoundTrip(
		req.Clone(ContextWithAllowPrivateNetwork(req.Context(), rt.allowPrivateNetwork)),
	)
}

// limitedRedirectForOutbound は、メソッドと本文を保つ 307 / 308 だけ追従し、他は最終レスポンス
// （3xx）をそのまま返します。
//
// 全面禁止にしないのは、AWS SDK 既定のクライアントが同じ範囲で追従するためです。HTTPClient を
// 明示した時点で SDK 側の追従は効かなくなるので、ここで同じ範囲を持たないと、S3 が 307 を返す
// 場面（作成直後のバケット等）で操作が失敗するようになります。
// 追従先も同じ transport を通るため、link-local / private 宛ての判定は追従後の接続にも効きます。
func limitedRedirectForOutbound(req *http.Request, via []*http.Request) error {
	if req.Response == nil || len(via) >= maxOutboundRedirects {
		return http.ErrUseLastResponse
	}
	switch req.Response.StatusCode {
	case http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return nil
	default:
		return http.ErrUseLastResponse
	}
}

// NewDisabledOutboundHTTPClient は、スパンを一切送出しない OutboundHTTPClient を返します。
// DI グラフを組まない CLI が infra 実装を直接組み立てる場合に用います。SSRF ガードは計装の有無に
// 依らず効くため、この経路でも link-local は拒否されます。
func NewDisabledOutboundHTTPClient(allowPrivateNetwork bool) *OutboundHTTPClient {
	return NewOutboundHTTPClient(
		NewHTTPClientTransport(noop.NewTracerProvider(), NewTextMapPropagator()),
		allowPrivateNetwork,
	)
}
