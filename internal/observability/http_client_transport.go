package observability

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"syscall"

	"go-boilerplate/pkg/xerrors"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// cgnatPrefix は、CGNAT 共有アドレス空間（RFC 6598, 100.64.0.0/10）です。
// Go の netip.Addr.IsPrivate は RFC1918 / ULA(fc00::/7) のみで CGNAT を含まないため、
// private 不許可時のブロック対象として明示的に判定します（クラウド内部用途の SSRF 面を塞ぐ）。
var cgnatPrefix = netip.MustParsePrefix("100.64.0.0/10")

// nat64WellKnownPrefix は、NAT64 の Well-Known Prefix（RFC 6052）です。埋め込み IPv4 を剥がして
// 判定しないと 64:ff9b::7f00:1（=127.0.0.1）等で IPv4 ガードを迂回されるため明示判定します。
var nat64WellKnownPrefix = netip.MustParsePrefix("64:ff9b::/96")

// reservedPrefixes は、bogon/予約帯として常時拒否する CIDR 一覧です。
// これらは正当な宛先になり得ないため、allowPrivateNetwork フラグに関わらずブロックします。
var reservedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),       // RFC 1122/6890 「このネットワーク」（IsUnspecified は 0.0.0.0 ちょうどしか捕捉しない）
	netip.MustParsePrefix("240.0.0.0/4"),     // RFC 1112/6890 将来予約（Future Use）
	netip.MustParsePrefix("192.0.0.0/24"),    // RFC 6890 IETF プロトコル割当（IETF Protocol Assignments）
	netip.MustParsePrefix("192.0.2.0/24"),    // RFC 5737 TEST-NET-1（ドキュメント/テスト用）
	netip.MustParsePrefix("198.51.100.0/24"), // RFC 5737 TEST-NET-2（ドキュメント/テスト用）
	netip.MustParsePrefix("203.0.113.0/24"),  // RFC 5737 TEST-NET-3（ドキュメント/テスト用）
	netip.MustParsePrefix("198.18.0.0/15"),   // RFC 2544 ベンチマーク測定用（Benchmarking）
	netip.MustParsePrefix("2001:db8::/32"),   // RFC 3849 IPv6 ドキュメント用（Documentation）
}

var (
	// errSSRFUnparsableAddress は、接続先アドレスをパースできず fail-close で拒否した場合のエラーです。
	errSSRFUnparsableAddress = xerrors.New("ssrf guard: blocked unparsable address")
	// errSSRFBlockedAddress は、link-local / unspecified の接続先を拒否した場合のエラーです。
	errSSRFBlockedAddress = xerrors.New("ssrf guard: blocked address")
	// errSSRFReservedAddress は、bogon/予約帯の接続先を拒否した場合のエラーです。
	errSSRFReservedAddress = xerrors.New("ssrf guard: blocked reserved address")
	// errSSRFPrivateAddress は、loopback / private / CGNAT の接続先を、許可フラグが無いまま拒否した場合のエラーです。
	errSSRFPrivateAddress = xerrors.New("ssrf guard: blocked private/loopback address")
)

// dialControl は、接続直前に呼ばれる net.Dialer の ControlContext 関数の型です。
type dialControl = func(ctx context.Context, network, address string, c syscall.RawConn) error

// HTTPClientTransport は、otelhttp 計装済みの outbound transport を包む不透明な型です。
type HTTPClientTransport struct {
	rt http.RoundTripper
}

// tracePropagationKey は、outbound 呼び出しでトレース伝搬を行うかのフラグを保持する ctx キーです。
type tracePropagationKey struct{}

// allowPrivateNetworkKey は、private/loopback 宛て接続を許可するかのフラグを保持する ctx キーです。
type allowPrivateNetworkKey struct{}

// conditionalPropagator は、ctx のフラグが false のとき Inject を抑止する propagator です。
type conditionalPropagator struct {
	inner propagation.TextMapPropagator
}

// spanQueryRedactionKey は、span 記録用に URL から一時退避した機密構成要素を運ぶ ctx キーです。
type spanQueryRedactionKey struct{}

// redactedURLParts は、span 記録のために URL から一時退避する機密になり得る構成要素です。
// フラグメントは実 HTTP リクエストには送出されませんが、url.full には現れるため退避対象に含めます。
type redactedURLParts struct {
	rawQuery    string
	fragment    string
	rawFragment string
}

// spanURLRedactingRoundTripper は、otelhttp が span の url.full へ記録する URL からクエリ・フラグメントを
// ctx へ退避して取り除きます（チェーン全体の順序は newHTTPClientTransport を参照）。
// クエリ・フラグメントは機密になり得るため既定で全除去します（httpclient のエラーメッセージ redaction＝redactURL と同方針）。
// userinfo は otelhttp が url.full 算出時に別途除去するため、ここでは扱いません。
type spanURLRedactingRoundTripper struct {
	inner http.RoundTripper
}

// urlSecretRestoringRoundTripper は、span 記録のために除去した機密構成要素を実送信直前に URL へ復元する base transport です。
// otelhttp は復元前に url.full を記録済みのため、ここでの復元は span へ影響しません。
type urlSecretRestoringRoundTripper struct {
	base http.RoundTripper
}

// NewHTTPClientTransport は、SSRF ガード付き base transport を otelhttp で計装した outbound transport を生成します。
//
// HTTP span 生成は自動化し、traceparent/baggage の outgoing inject は ContextWithTracePropagation の
// フラグに従って外部 downstream への伝搬を抑止できます。private/loopback 宛て接続は
// ContextWithAllowPrivateNetwork のフラグで制御し、link-local（クラウドメタデータ等）は常に拒否します。
func NewHTTPClientTransport(tp trace.TracerProvider, propagator propagation.TextMapPropagator) *HTTPClientTransport {
	return newHTTPClientTransport(tp, propagator, guardedDialControl)
}

// newHTTPClientTransport は、dial control を差し替え可能にした内部コンストラクタです。
//
// transport チェーンは外側から redact(退避) → otelhttp(span生成) → restore(復元) → guardedBase の順です。
// otelhttp は span の url.full を req.URL.String() から算出しクエリ・フラグメント込みで記録するため、otelhttp へ
// 渡す前にこれらを退避・除去し、実送信直前に復元することで span からのみ落とします（実リクエストは無改変）。
//
// 不変条件: otelhttp の base には URL をエラーメッセージへ整形しない素の RoundTripper（*http.Transport）を渡します。
// エラー時 otelhttp は span へ err.Error() を記録するため、base が *url.Error（URL 込み）を返す層（例: http.Client
// でのラップ）だと復元後のクエリがエラー span へ漏れます。この境界を跨ぐ RoundTripper を挟む改修は避けてください。
func newHTTPClientTransport(
	tp trace.TracerProvider, propagator propagation.TextMapPropagator, control dialControl,
) *HTTPClientTransport {
	rt := otelhttp.NewTransport(
		urlSecretRestoringRoundTripper{base: newGuardedBaseTransport(control)},
		otelhttp.WithTracerProvider(tp),
		otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
		otelhttp.WithPropagators(conditionalPropagator{inner: propagator}),
	)
	return &HTTPClientTransport{rt: spanURLRedactingRoundTripper{inner: rt}}
}

// RoundTrip は、クエリ・フラグメントを ctx へ退避し URL から除去した clone を otelhttp へ渡します。
func (rt spanURLRedactingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL == nil || (req.URL.RawQuery == "" && req.URL.Fragment == "" && req.URL.RawFragment == "") {
		return rt.inner.RoundTrip(req)
	}
	// http.RoundTripper は呼び出し元の Request を変更してはならないため clone する。
	parts := redactedURLParts{
		rawQuery:    req.URL.RawQuery,
		fragment:    req.URL.Fragment,
		rawFragment: req.URL.RawFragment,
	}
	cloned := req.Clone(context.WithValue(req.Context(), spanQueryRedactionKey{}, parts))
	cloned.URL.RawQuery = ""
	cloned.URL.Fragment = ""
	cloned.URL.RawFragment = ""
	return rt.inner.RoundTrip(cloned)
}

// RoundTrip は、退避済みの機密構成要素があれば URL へ復元してから base へ委譲します。
func (rt urlSecretRestoringRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if parts, ok := req.Context().Value(spanQueryRedactionKey{}).(redactedURLParts); ok && req.URL != nil {
		req.URL.RawQuery = parts.rawQuery
		req.URL.Fragment = parts.fragment
		req.URL.RawFragment = parts.rawFragment
	}
	return rt.base.RoundTrip(req)
}

// RoundTripper は、ラップしている http.RoundTripper を返します。
func (t *HTTPClientTransport) RoundTripper() http.RoundTripper {
	return t.rt
}

// ContextWithTracePropagation は、この outbound 呼び出しで traceparent/baggage を注入するかを ctx に設定します。
func ContextWithTracePropagation(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, tracePropagationKey{}, enabled)
}

// ContextWithAllowPrivateNetwork は、この outbound 呼び出しで private/loopback 宛て接続を許可するかを ctx に設定します。
func ContextWithAllowPrivateNetwork(ctx context.Context, allowed bool) context.Context {
	return context.WithValue(ctx, allowPrivateNetworkKey{}, allowed)
}

// Inject は、伝搬フラグが明示的に false の場合のみ注入を抑止し、それ以外は内側へ委譲します。
func (p conditionalPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	if enabled, ok := ctx.Value(tracePropagationKey{}).(bool); ok && !enabled {
		return
	}
	p.inner.Inject(ctx, carrier)
}

// Extract は、内側の propagator へ委譲します。
func (p conditionalPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return p.inner.Extract(ctx, carrier)
}

// Fields は、内側の propagator へ委譲します。
func (p conditionalPropagator) Fields() []string {
	return p.inner.Fields()
}

// newGuardedBaseTransport は、指定の dial control を備えた base transport を返します。
func newGuardedBaseTransport(control dialControl) *http.Transport {
	base := &http.Transport{}
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		base = t.Clone()
	}
	dialer := &net.Dialer{ControlContext: control}
	base.DialContext = dialer.DialContext
	// 不変条件: proxy 経由では dial 先が宛先ではなく proxy になり、guardedDialControl の宛先 IP 検査が
	// 素通りする（SSRF ガードの無効化）。DefaultTransport から継承した環境変数由来の Proxy を無効化し、
	// 宛先へ直結してガードを常に宛先 IP に効かせる（ADR-0022 (egress-ssrf-guard) の最終宛先 IP 検査と整合）。
	// 運用注意: 直接 egress を遮断し forward proxy 必須にした環境では outbound HTTP が全断する
	// （HTTP_PROXY 注入では復活せず、ネットワーク層での吸収が必要）。
	base.Proxy = nil
	return base
}

// guardedDialControl は、名前解決後の実接続先 IP を検査する SSRF ガードです（DNS rebinding も防ぎます）。
// パース不能なアドレスは fail-close で拒否し（zone 付き IPv6 リテラルによる link-local 判定回避を塞ぐ）、
// link-local / unspecified / bogon予約帯は常に拒否、loopback / private / CGNAT は ctx フラグで許可されない限り拒否します。
func guardedDialControl(ctx context.Context, _, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return xerrors.Wrap(err, "ssrf guard: blocked malformed address "+address)
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return xerrors.Wrap(errSSRFUnparsableAddress, host)
	}
	// zone を落として family を正規化し、Prefix 判定を IPv4-mapped/zone 付きでも取りこぼさないようにする。
	addr = addr.Unmap().WithZone("")
	// 埋め込み IPv4 を取り出し、以降の loopback/private/予約帯判定を実宛先に効かせる。
	if nat64WellKnownPrefix.Contains(addr) {
		b := addr.As16()
		addr = netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]})
	}
	if addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsUnspecified() {
		return xerrors.Wrap(errSSRFBlockedAddress, host)
	}
	// bogon/予約帯は正当な宛先にならないため allowPrivateNetwork フラグに関わらず常時拒否する。
	for _, p := range reservedPrefixes {
		if p.Contains(addr) {
			return xerrors.Wrap(errSSRFReservedAddress, host)
		}
	}
	if !allowPrivateNetworkFromContext(ctx) && (addr.IsLoopback() || addr.IsPrivate() || cgnatPrefix.Contains(addr)) {
		return xerrors.Wrap(errSSRFPrivateAddress, host)
	}
	return nil
}

// permissiveDialControl は、すべての接続を許可する dial control です（テスト用）。
func permissiveDialControl(context.Context, string, string, syscall.RawConn) error {
	return nil
}

// allowPrivateNetworkFromContext は、ctx の private 許可フラグを返します（未設定は安全側の false）。
func allowPrivateNetworkFromContext(ctx context.Context) bool {
	allowed, ok := ctx.Value(allowPrivateNetworkKey{}).(bool)
	return ok && allowed
}
