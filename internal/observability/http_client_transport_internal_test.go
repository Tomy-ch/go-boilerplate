package observability

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// captureRoundTripper は、委譲先へ渡された Request を捕捉するテスト用 RoundTripper です。
type captureRoundTripper struct {
	gotReq *http.Request
}

// capturingSpanExporter は、終了した span を捕捉するテスト用 SpanExporter です。
type capturingSpanExporter struct {
	mu    sync.Mutex
	spans []sdktrace.ReadOnlySpan
}

func Test_guardedDialControl(t *testing.T) {
	t.Parallel()

	allow := ContextWithAllowPrivateNetwork(context.Background(), true)
	deny := ContextWithAllowPrivateNetwork(context.Background(), false)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("グローバルアドレスは許可する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, guardedDialControl(deny, "tcp", "93.184.216.34:443", nil))
		})

		t.Run("private許可フラグありならloopback/privateを許可する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, guardedDialControl(allow, "tcp", "127.0.0.1:8080", nil))
			require.NoError(t, guardedDialControl(allow, "tcp", "10.0.0.5:80", nil))
			require.NoError(t, guardedDialControl(allow, "tcp", "192.168.1.10:80", nil))
			// CGNAT(RFC 6598) も private 扱いで、フラグありなら許可する。
			require.NoError(t, guardedDialControl(allow, "tcp", "100.64.0.1:80", nil))
		})

		t.Run("CGNAT帯の外(100.128.x)は許可する", func(t *testing.T) {
			t.Parallel()
			// 100.128.0.0 は /10 の外＝グローバル扱いで通ること（過剰ブロック防止）。
			require.NoError(t, guardedDialControl(deny, "tcp", "100.128.0.1:80", nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ポート無しアドレスはSplitHostPortエラーを返す", func(t *testing.T) {
			t.Parallel()
			// net.SplitHostPort が失敗するアドレス（ポート区切り無し）はそのままエラーを返す。
			require.Error(t, guardedDialControl(deny, "tcp", "noport", nil))
		})

		t.Run("IPリテラルでないホスト名はfail-closeで拒否する", func(t *testing.T) {
			t.Parallel()
			// パース不能なアドレスは許可せず拒否する（fail-close）。
			require.Error(t, guardedDialControl(allow, "tcp", "example.com:80", nil))
			require.Error(t, guardedDialControl(deny, "tcp", "example.com:80", nil))
		})

		t.Run("zone付きIPv6リンクローカルはフラグに関わらず拒否する", func(t *testing.T) {
			t.Parallel()
			// fe80::1%eth0 は zone 付きでも link-local と判定してブロックする（判定回避の穴を塞ぐ）。
			require.Error(t, guardedDialControl(allow, "tcp", "[fe80::1%eth0]:80", nil))
			require.Error(t, guardedDialControl(deny, "tcp", "[fe80::1%eth0]:80", nil))
		})

		t.Run("IPv4-mappedのIPv6リテラルでも予約帯/CGNATを拒否する", func(t *testing.T) {
			t.Parallel()
			// netip.Prefix.Contains は family 不一致（IPv4 prefix vs 4-in-6）で false を返すため、
			// Unmap による正規化が退行すると ::ffff: 形式で判定を迂回できてしまう。
			require.Error(t, guardedDialControl(allow, "tcp", "[::ffff:240.0.0.1]:80", nil))
			require.Error(t, guardedDialControl(deny, "tcp", "[::ffff:100.64.0.1]:80", nil))
		})

		t.Run("リンクローカル(メタデータ)はフラグに関わらず拒否する", func(t *testing.T) {
			t.Parallel()
			require.Error(t, guardedDialControl(allow, "tcp", "169.254.169.254:80", nil))
			require.Error(t, guardedDialControl(deny, "tcp", "169.254.169.254:80", nil))
		})

		t.Run("private許可フラグなしならloopback/privateを拒否する", func(t *testing.T) {
			t.Parallel()
			require.Error(t, guardedDialControl(deny, "tcp", "127.0.0.1:8080", nil))
			require.Error(t, guardedDialControl(deny, "tcp", "10.0.0.5:80", nil))
			require.Error(t, guardedDialControl(deny, "tcp", "192.168.1.10:80", nil))
		})

		t.Run("private許可フラグなしならCGNAT(100.64.0.0/10)も拒否する", func(t *testing.T) {
			t.Parallel()
			// Go の IsPrivate は CGNAT を含まないため、明示判定で塞ぐ。境界も検証。
			require.Error(t, guardedDialControl(deny, "tcp", "100.64.0.1:80", nil))
			require.Error(t, guardedDialControl(deny, "tcp", "100.127.255.254:80", nil))
		})

		t.Run("予約/将来利用帯はprivate許可フラグありでも拒否する", func(t *testing.T) {
			t.Parallel()
			// RFC 5737 TEST-NET-1（192.0.2.0/24）— ドキュメント/テスト用、正当な宛先にならない。
			require.Error(t, guardedDialControl(allow, "tcp", "192.0.2.1:80", nil))
			// RFC 5737 TEST-NET-2（198.51.100.0/24）— ドキュメント/テスト用、正当な宛先にならない。
			require.Error(t, guardedDialControl(allow, "tcp", "198.51.100.5:80", nil))
			// RFC 5737 TEST-NET-3（203.0.113.0/24）— ドキュメント/テスト用、正当な宛先にならない。
			require.Error(t, guardedDialControl(allow, "tcp", "203.0.113.5:80", nil))
			// RFC 1112/6890 将来予約（240.0.0.0/4）— 現実の宛先として到達不能。
			require.Error(t, guardedDialControl(allow, "tcp", "240.0.0.1:80", nil))
			// RFC 2544 ベンチマーク測定用（198.18.0.0/15）— 本番トラフィックの宛先にならない。
			require.Error(t, guardedDialControl(allow, "tcp", "198.18.0.1:80", nil))
			// RFC 1122/6890「このネットワーク」（0.0.0.0/8）— 0.0.0.0 ちょうど以外も拒否する。
			require.Error(t, guardedDialControl(allow, "tcp", "0.0.0.1:80", nil))
			// 0.0.0.0 ちょうどは IsUnspecified 経路で拒否される（/8 追加前からの既存挙動）。
			require.Error(t, guardedDialControl(allow, "tcp", "0.0.0.0:80", nil))
			// RFC 6890 IETF プロトコル割当（192.0.0.0/24）— 一般宛て通信には使われない。
			require.Error(t, guardedDialControl(allow, "tcp", "192.0.0.1:80", nil))
			// RFC 3849 IPv6 ドキュメント用（2001:db8::/32）— テスト/文書専用で実到達不能。
			require.Error(t, guardedDialControl(allow, "tcp", "[2001:db8::1]:443", nil))
		})
	})
}

func newSampledContext() context.Context {
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
	})
	return trace.ContextWithSpanContext(context.Background(), sc)
}

func Test_conditionalPropagator_Inject(t *testing.T) {
	t.Parallel()
	t.Skip("conditionalPropagator.Inject は Test_conditionalPropagator が有効/無効/未設定の全分岐を検証済み")
}

func Test_newGuardedBaseTransport(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定した dial control を DialContext に持つ transport を返す", func(t *testing.T) {
			t.Parallel()

			tr := newGuardedBaseTransport(permissiveDialControl)

			require.NotNil(t, tr)
			assert.NotNil(t, tr.DialContext)
		})
	})
}

func Test_conditionalPropagator(t *testing.T) {
	t.Parallel()

	prop := conditionalPropagator{inner: propagation.TraceContext{}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フラグ未設定なら通常どおり伝搬する", func(t *testing.T) {
			t.Parallel()

			carrier := propagation.MapCarrier{}
			prop.Inject(newSampledContext(), carrier)
			assert.NotEmpty(t, carrier.Get("traceparent"))
		})

		t.Run("伝搬有効フラグでは伝搬する", func(t *testing.T) {
			t.Parallel()

			ctx := ContextWithTracePropagation(newSampledContext(), true)
			carrier := propagation.MapCarrier{}
			prop.Inject(ctx, carrier)
			assert.NotEmpty(t, carrier.Get("traceparent"))
		})

		t.Run("Extract は内側 propagator へ委譲し trace context を復元する", func(t *testing.T) {
			t.Parallel()

			// 内側と同じ propagator で carrier を作り、Extract が委譲されていることを確認する。
			carrier := propagation.MapCarrier{}
			propagation.TraceContext{}.Inject(newSampledContext(), carrier)

			got := prop.Extract(context.Background(), carrier)

			sc := trace.SpanContextFromContext(got)
			assert.True(t, sc.IsValid())
			want := trace.SpanContextFromContext(newSampledContext())
			assert.Equal(t, want.TraceID().String(), sc.TraceID().String())
			assert.Equal(t, want.SpanID().String(), sc.SpanID().String())
		})

		t.Run("Fields は内側 propagator のキー集合を委譲する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, propagation.TraceContext{}.Fields(), prop.Fields())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("伝搬無効フラグでは外部へ注入しない", func(t *testing.T) {
			t.Parallel()

			ctx := ContextWithTracePropagation(newSampledContext(), false)
			carrier := propagation.MapCarrier{}
			prop.Inject(ctx, carrier)
			assert.Empty(t, carrier.Get("traceparent"))
		})
	})
}

func (c *captureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	c.gotReq = req
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
}

func Test_spanURLRedactingRoundTripper_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クエリをctxへ退避しinnerへ渡すURLからは除去する", func(t *testing.T) {
			t.Parallel()

			captured := &captureRoundTripper{}
			rt := spanURLRedactingRoundTripper{inner: captured}
			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "https://example.com/search?token=secret&postalCode=1000001", nil)
			require.NoError(t, err)

			_, err = rt.RoundTrip(req)

			require.NoError(t, err)
			// inner（otelhttp）へ渡る URL はクエリ無し＝span の url.full にクエリが乗らない。
			assert.Empty(t, captured.gotReq.URL.RawQuery)
			// 実クエリは ctx に退避され、後段の復元に使える。
			assert.Equal(t, "token=secret&postalCode=1000001", captured.gotReq.Context().Value(spanQueryRedactionKey{}))
			// caller の Request は破壊しない。
			assert.Equal(t, "token=secret&postalCode=1000001", req.URL.RawQuery)
		})

		t.Run("クエリが無いRequestはそのままinnerへ渡す", func(t *testing.T) {
			t.Parallel()

			captured := &captureRoundTripper{}
			rt := spanURLRedactingRoundTripper{inner: captured}
			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "https://example.com/search", nil)
			require.NoError(t, err)

			_, err = rt.RoundTrip(req)

			require.NoError(t, err)
			assert.Empty(t, captured.gotReq.URL.RawQuery)
			assert.Nil(t, captured.gotReq.Context().Value(spanQueryRedactionKey{}))
		})
	})
}

func Test_queryRestoringRoundTripper_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("退避済みクエリを実送信前にURLへ復元する", func(t *testing.T) {
			t.Parallel()

			captured := &captureRoundTripper{}
			rt := queryRestoringRoundTripper{base: captured}
			ctx := context.WithValue(context.Background(), spanQueryRedactionKey{}, "token=secret&postalCode=1000001")
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com/search", nil)
			require.NoError(t, err)

			_, err = rt.RoundTrip(req)

			require.NoError(t, err)
			assert.Equal(t, "token=secret&postalCode=1000001", captured.gotReq.URL.RawQuery)
		})

		t.Run("退避クエリが無ければURLを変更しない", func(t *testing.T) {
			t.Parallel()

			captured := &captureRoundTripper{}
			rt := queryRestoringRoundTripper{base: captured}
			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "https://example.com/search", nil)
			require.NoError(t, err)

			_, err = rt.RoundTrip(req)

			require.NoError(t, err)
			assert.Empty(t, captured.gotReq.URL.RawQuery)
		})
	})
}

func (e *capturingSpanExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *capturingSpanExporter) Shutdown(context.Context) error { return nil }

func (e *capturingSpanExporter) clientSpanURLFull(t *testing.T) string {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, s := range e.spans {
		if s.SpanKind() != trace.SpanKindClient {
			continue
		}
		for _, attr := range s.Attributes() {
			if attr.Key == attribute.Key("url.full") {
				return attr.Value.AsString()
			}
		}
	}
	t.Fatal("client span with url.full attribute not found")
	return ""
}

func Test_newHTTPClientTransport_redactsQueryFromSpanButPreservesRequest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("span_url_fullはクエリ無し_実リクエストはクエリを保持する", func(t *testing.T) {
			t.Parallel()

			var (
				mu         sync.Mutex
				gotRawQ    string
				serverPath string
			)
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				mu.Lock()
				gotRawQ = r.URL.RawQuery
				serverPath = r.URL.Path
				mu.Unlock()
			}))
			defer srv.Close()

			exporter := &capturingSpanExporter{}
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
			defer func() { _ = tp.Shutdown(context.Background()) }()

			transport := newHTTPClientTransport(tp, NewTextMapPropagator(), permissiveDialControl)
			httpClient := &http.Client{Transport: transport.RoundTripper()}

			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, srv.URL+"/search?token=secret&postalCode=1000001", nil)
			require.NoError(t, err)

			resp, err := httpClient.Do(req)
			require.NoError(t, err)
			_, _ = io.Copy(io.Discard, resp.Body)
			require.NoError(t, resp.Body.Close()) // otelhttp は body close で span を終了する
			require.NoError(t, tp.ForceFlush(context.Background()))

			// 実リクエストにはクエリがそのまま届く（復元が効いている）。
			mu.Lock()
			assert.Equal(t, "token=secret&postalCode=1000001", gotRawQ)
			assert.Equal(t, "/search", serverPath)
			mu.Unlock()

			// span の url.full にはクエリが乗らない（機微情報の漏洩を防ぐ）。
			urlFull := exporter.clientSpanURLFull(t)
			assert.NotContains(t, urlFull, "token")
			assert.NotContains(t, urlFull, "secret")
			assert.NotContains(t, urlFull, "postalCode")
			assert.Equal(t, srv.URL+"/search", urlFull)
		})
	})
}
