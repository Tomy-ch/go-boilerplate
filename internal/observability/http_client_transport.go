package observability

import (
	"context"
	"net"
	"net/http"
	"syscall"

	"go-boilerplate/pkg/xerrors"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// cgnatNet は、CGNAT 共有アドレス空間（RFC 6598, 100.64.0.0/10）です（L1）。
// Go の net.IP.IsPrivate は RFC1918 / ULA(fc00::/7) のみで CGNAT を含まないため、
// private 不許可時のブロック対象として明示的に判定します（クラウド内部用途の SSRF 面を塞ぐ）。
var cgnatNet = mustParseCIDR("100.64.0.0/10")

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

// NewHTTPClientTransport は、SSRF ガード付き base transport を otelhttp で計装した outbound transport を生成します。
//
// HTTP span 生成は自動化し、traceparent/baggage の outgoing inject は ContextWithTracePropagation の
// フラグに従って外部 downstream への伝搬を抑止できます。private/loopback 宛て接続は
// ContextWithAllowPrivateNetwork のフラグで制御し、link-local（クラウドメタデータ等）は常に拒否します。
func NewHTTPClientTransport(tp trace.TracerProvider, propagator propagation.TextMapPropagator) *HTTPClientTransport {
	return newHTTPClientTransport(tp, propagator, guardedDialControl)
}

// newHTTPClientTransport は、dial control を差し替え可能にした内部コンストラクタです。
func newHTTPClientTransport(
	tp trace.TracerProvider, propagator propagation.TextMapPropagator, control dialControl,
) *HTTPClientTransport {
	rt := otelhttp.NewTransport(
		newGuardedBaseTransport(control),
		otelhttp.WithTracerProvider(tp),
		otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
		otelhttp.WithPropagators(conditionalPropagator{inner: propagator}),
	)
	return &HTTPClientTransport{rt: rt}
}

// RoundTripper は、内部の RoundTripper を返します（substrate 内部でのみ利用します）。
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
	return base
}

// guardedDialControl は、名前解決後の実接続先 IP を検査する SSRF ガードです（DNS rebinding も防ぎます）。
// link-local / unspecified は常に拒否し、loopback / private / CGNAT は ctx フラグで許可されない限り拒否します。
func guardedDialControl(ctx context.Context, _, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return xerrors.New("ssrf guard: blocked address " + host)
	}
	if !allowPrivateNetworkFromContext(ctx) && (ip.IsLoopback() || ip.IsPrivate() || cgnatNet.Contains(ip)) {
		return xerrors.New("ssrf guard: blocked private/loopback address " + host)
	}
	return nil
}

// mustParseCIDR は、定数 CIDR 文字列を *net.IPNet へ解析します。不正リテラルは起動時 panic です。
func mustParseCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
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
