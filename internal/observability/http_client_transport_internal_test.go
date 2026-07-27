package observability

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"syscall"
	"testing"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// errTestRoundTrip は、委譲先 RoundTripper のエラー伝播を検証するためのセンチネルです。
var errTestRoundTrip = xerrors.New("round trip failed")

// captureRoundTripper は、委譲先へ渡された Request を捕捉するテスト用 RoundTripper です。
// err を設定するとそのエラーを返し、委譲先のエラーがそのまま伝播することを検証できます。
type captureRoundTripper struct {
	gotReq *http.Request
	err    error
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

		t.Run("NAT64 Well-Known Prefixに埋め込んだpublic IPv4は許可する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, guardedDialControl(deny, "tcp", "[64:ff9b::5db8:d822]:80", nil)) // 93.184.216.34
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

		t.Run("NAT64 Well-Known Prefixに埋め込んだ内部宛てIPは埋め込みIPv4で判定して拒否する", func(t *testing.T) {
			t.Parallel()
			// 埋め込み IPv4 を剥がす正規化が退行すると IPv4 ガードを迂回できる。
			require.Error(t, guardedDialControl(deny, "tcp", "[64:ff9b::7f00:1]:80", nil))     // 127.0.0.1
			require.Error(t, guardedDialControl(deny, "tcp", "[64:ff9b::a00:5]:80", nil))      // 10.0.0.5
			require.Error(t, guardedDialControl(allow, "tcp", "[64:ff9b::a9fe:a9fe]:80", nil)) // 169.254.169.254
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

//nolint:paralleltest // 子テストが t.Setenv を使うため関数全体を並列化不可
func Test_newGuardedBaseTransport(t *testing.T) {
	//nolint:paralleltest // 子テストが t.Setenv を使うため並列化不可
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 兄弟テストが t.Setenv を使うため並列化不可
		t.Run("指定した dial control を DialContext に持つ transport を返す", func(t *testing.T) {
			tr := newGuardedBaseTransport(permissiveDialControl)

			require.NotNil(t, tr)
			assert.NotNil(t, tr.DialContext)
		})

		//nolint:paralleltest // 兄弟テストが t.Setenv を使うため並列化不可
		t.Run("env 由来 proxy を継承せず Proxy を無効化する", func(t *testing.T) {
			tr := newGuardedBaseTransport(permissiveDialControl)

			// 不変条件: proxy 経由では dial 先が proxy になり宛先 IP 検査が素通りするため、Proxy は常に nil。
			assert.Nil(t, tr.Proxy)
		})

		//nolint:paralleltest // t.Setenv 使用のため並列化不可
		t.Run("proxy 設定下でも dial 先が宛先IPでガードが効く", func(t *testing.T) {
			// proxy を設定しても base.Proxy=nil のため transport は proxy を使わず宛先へ直結し、
			// guardedDialControl（dial 先 IP 検査）が宛先に効き続ける（SSRF ガードの無効化を防ぐ）。
			t.Setenv("HTTP_PROXY", "http://10.0.0.1:3128")
			t.Setenv("HTTPS_PROXY", "http://10.0.0.1:3128")

			var gotAddr string
			control := func(_ context.Context, _, address string, _ syscall.RawConn) error {
				gotAddr = address
				return errTestRoundTrip // 実接続はせず dial 直前で止める
			}

			base := newGuardedBaseTransport(control)
			// 回帰の実効ロックはこれ: Proxy=nil であれば http.Client は ProxyFromEnvironment を参照せず、
			// dial 先が常に宛先になり宛先 IP ガードが効く。SSRF ガード無効化の退行はここで確実に落ちる。
			require.Nil(t, base.Proxy)

			client := &http.Client{Transport: base}
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://93.184.216.34:80/x", nil)
			require.NoError(t, err)

			_, err = client.Do(req)
			require.Error(t, err) // dial control が止めるためエラーになる

			// e2e アサート: dial 先が proxy(10.0.0.1) ではなく宛先(93.184.216.34) であること。ただし net/http の
			// proxy 判定は httpproxy 環境変数を一度だけキャッシュするため、他テストの環境変数読取り順序に依存しうる。
			// 実効ロックは上の require.Nil(base.Proxy) 側にあり、本アサートは補助的な end-to-end 確認にとどめる。
			assert.Equal(t, "93.184.216.34:80", gotAddr)
		})
	})
}

func Test_conditionalPropagator_Inject(t *testing.T) {
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

func Test_conditionalPropagator_Extract(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("内側 propagator へ委譲し trace context を復元する", func(t *testing.T) {
			t.Parallel()

			// 内側と同じ propagator で carrier を作り、Extract が委譲されていることを確認する。
			carrier := propagation.MapCarrier{}
			propagation.TraceContext{}.Inject(newSampledContext(), carrier)

			got := conditionalPropagator{inner: propagation.TraceContext{}}.Extract(context.Background(), carrier)

			sc := trace.SpanContextFromContext(got)
			assert.True(t, sc.IsValid())
			want := trace.SpanContextFromContext(newSampledContext())
			assert.Equal(t, want.TraceID().String(), sc.TraceID().String())
			assert.Equal(t, want.SpanID().String(), sc.SpanID().String())
		})
	})
}

func Test_conditionalPropagator_Fields(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("内側 propagator のキー集合を委譲する", func(t *testing.T) {
			t.Parallel()

			prop := conditionalPropagator{inner: propagation.TraceContext{}}

			assert.Equal(t, propagation.TraceContext{}.Fields(), prop.Fields())
		})
	})
}

func (c *captureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	c.gotReq = req
	if c.err != nil {
		return nil, c.err
	}
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
}

func Test_spanURLRedactingRoundTripper_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クエリとフラグメントをctxへ退避しinnerへ渡すURLからは除去する", func(t *testing.T) {
			t.Parallel()

			captured := &captureRoundTripper{}
			rt := spanURLRedactingRoundTripper{inner: captured}
			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "https://example.com/search?token=secret&postalCode=1000001#sess=abc", nil)
			require.NoError(t, err)

			_, err = rt.RoundTrip(req)

			require.NoError(t, err)
			// inner（otelhttp）へ渡る URL はクエリ・フラグメント無し＝span の url.full にどちらも乗らない。
			assert.Empty(t, captured.gotReq.URL.RawQuery)
			assert.Empty(t, captured.gotReq.URL.Fragment)
			// 実クエリ・フラグメントは ctx に退避され、後段の復元に使える。
			parts, ok := captured.gotReq.Context().Value(spanQueryRedactionKey{}).(redactedURLParts)
			require.True(t, ok)
			assert.Equal(t, "token=secret&postalCode=1000001", parts.rawQuery)
			assert.Equal(t, "sess=abc", parts.fragment)
			// caller の Request は破壊しない。
			assert.Equal(t, "token=secret&postalCode=1000001", req.URL.RawQuery)
			assert.Equal(t, "sess=abc", req.URL.Fragment)
		})

		t.Run("クエリもフラグメントも無いRequestはそのままinnerへ渡す", func(t *testing.T) {
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

		t.Run("URLがnilならそのままinnerへ渡しpanicしない", func(t *testing.T) {
			t.Parallel()

			captured := &captureRoundTripper{}
			rt := spanURLRedactingRoundTripper{inner: captured}
			req := &http.Request{Method: http.MethodGet, Header: make(http.Header)}

			_, err := rt.RoundTrip(req.WithContext(context.Background()))

			require.NoError(t, err)
			assert.Nil(t, captured.gotReq.URL)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("innerのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			wantErr := errTestRoundTrip
			captured := &captureRoundTripper{err: wantErr}
			rt := spanURLRedactingRoundTripper{inner: captured}
			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "https://example.com/search?token=secret", nil)
			require.NoError(t, err)

			resp, err := rt.RoundTrip(req)

			require.ErrorIs(t, err, wantErr)
			assert.Nil(t, resp)
		})
	})
}

func Test_urlSecretRestoringRoundTripper_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("退避済みクエリとフラグメントを実送信前にURLへ復元する", func(t *testing.T) {
			t.Parallel()

			captured := &captureRoundTripper{}
			rt := urlSecretRestoringRoundTripper{base: captured}
			parts := redactedURLParts{rawQuery: "token=secret&postalCode=1000001", fragment: "sess=abc"}
			ctx := context.WithValue(context.Background(), spanQueryRedactionKey{}, parts)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com/search", nil)
			require.NoError(t, err)

			_, err = rt.RoundTrip(req)

			require.NoError(t, err)
			assert.Equal(t, "token=secret&postalCode=1000001", captured.gotReq.URL.RawQuery)
			assert.Equal(t, "sess=abc", captured.gotReq.URL.Fragment)
		})

		t.Run("退避値が無ければURLを変更しない", func(t *testing.T) {
			t.Parallel()

			captured := &captureRoundTripper{}
			rt := urlSecretRestoringRoundTripper{base: captured}
			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "https://example.com/search", nil)
			require.NoError(t, err)

			_, err = rt.RoundTrip(req)

			require.NoError(t, err)
			assert.Empty(t, captured.gotReq.URL.RawQuery)
		})

		t.Run("URLがnilなら復元せずそのままbaseへ渡しpanicしない", func(t *testing.T) {
			t.Parallel()

			captured := &captureRoundTripper{}
			rt := urlSecretRestoringRoundTripper{base: captured}
			ctx := context.WithValue(context.Background(), spanQueryRedactionKey{}, redactedURLParts{rawQuery: "token=secret"})
			req := (&http.Request{Method: http.MethodGet, Header: make(http.Header)}).WithContext(ctx)

			_, err := rt.RoundTrip(req)

			require.NoError(t, err)
			assert.Nil(t, captured.gotReq.URL)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("baseのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			wantErr := errTestRoundTrip
			captured := &captureRoundTripper{err: wantErr}
			rt := urlSecretRestoringRoundTripper{base: captured}
			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "https://example.com/search", nil)
			require.NoError(t, err)

			resp, err := rt.RoundTrip(req)

			require.ErrorIs(t, err, wantErr)
			assert.Nil(t, resp)
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

func (e *capturingSpanExporter) clientSpan(t *testing.T) sdktrace.ReadOnlySpan {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, s := range e.spans {
		if s.SpanKind() == trace.SpanKindClient {
			return s
		}
	}
	t.Fatal("client span not found")
	return nil
}

func (e *capturingSpanExporter) clientSpanURLFull(t *testing.T) string {
	t.Helper()
	for _, attr := range e.clientSpan(t).Attributes() {
		if attr.Key == attribute.Key("url.full") {
			return attr.Value.AsString()
		}
	}
	t.Fatal("client span has no url.full attribute")
	return ""
}

// clientSpanText は、client span から漏洩検査対象となる文字列（status 説明 + span 名 + 全属性値）を集めます。
func (e *capturingSpanExporter) clientSpanText(t *testing.T) string {
	t.Helper()
	s := e.clientSpan(t)
	var sb strings.Builder
	sb.WriteString(s.Name())
	sb.WriteString(" ")
	sb.WriteString(s.Status().Description)
	for _, attr := range s.Attributes() {
		sb.WriteString(" ")
		sb.WriteString(attr.Value.String())
	}
	return sb.String()
}

func Test_newHTTPClientTransport_redactsQueryFromSpanButPreservesRequest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("span_url_fullはクエリ_フラグメント無し_実リクエストはクエリを保持する", func(t *testing.T) {
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
				context.Background(), http.MethodGet, srv.URL+"/search?token=secret&postalCode=1000001#sess=abc", nil)
			require.NoError(t, err)

			resp, err := httpClient.Do(req)
			require.NoError(t, err)
			_, _ = io.Copy(io.Discard, resp.Body)
			require.NoError(t, resp.Body.Close()) // otelhttp は body close で span を終了する
			require.NoError(t, tp.ForceFlush(context.Background()))

			// 実リクエストにはクエリがそのまま届く（復元が効いている。フラグメントは HTTP では送出されない）。
			mu.Lock()
			assert.Equal(t, "token=secret&postalCode=1000001", gotRawQ)
			assert.Equal(t, "/search", serverPath)
			mu.Unlock()

			// span の url.full にはクエリもフラグメントも乗らない（機微情報の漏洩を防ぐ）。
			urlFull := exporter.clientSpanURLFull(t)
			assert.NotContains(t, urlFull, "token")
			assert.NotContains(t, urlFull, "secret")
			assert.NotContains(t, urlFull, "postalCode")
			assert.NotContains(t, urlFull, "sess")
			assert.Equal(t, srv.URL+"/search", urlFull)
		})

		t.Run("接続失敗時もエラーspanにクエリが乗らない", func(t *testing.T) {
			t.Parallel()

			// 即座に閉じるリスナーへ接続させ、transport 層のエラー（*net.OpError）を発生させる。
			// otelhttp は err.Error() を span status へ記録するため、そこにクエリが漏れないことを検証する。
			ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
			require.NoError(t, err)
			addr := ln.Addr().String()
			require.NoError(t, ln.Close())

			exporter := &capturingSpanExporter{}
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
			defer func() { _ = tp.Shutdown(context.Background()) }()

			transport := newHTTPClientTransport(tp, NewTextMapPropagator(), permissiveDialControl)
			httpClient := &http.Client{Transport: transport.RoundTripper()}

			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "http://"+addr+"/search?token=secret&postalCode=1000001", nil)
			require.NoError(t, err)

			resp, err := httpClient.Do(req)
			require.Error(t, err)
			assert.Nil(t, resp)
			require.NoError(t, tp.ForceFlush(context.Background()))

			leakText := exporter.clientSpanText(t)
			assert.NotContains(t, leakText, "token")
			assert.NotContains(t, leakText, "secret")
			assert.NotContains(t, leakText, "postalCode")
		})
	})
}
