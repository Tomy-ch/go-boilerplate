package observability

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestGuardedDialControl(t *testing.T) {
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

		t.Run("IPリテラルでないホスト名は素通しする", func(t *testing.T) {
			t.Parallel()
			// ParseIP が nil（未解決ホスト名）なら、実接続先 IP の判定は dial 後に委ねるため素通しする。
			require.NoError(t, guardedDialControl(deny, "tcp", "example.com:80", nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ポート無しアドレスはSplitHostPortエラーを返す", func(t *testing.T) {
			t.Parallel()
			// net.SplitHostPort が失敗するアドレス（ポート区切り無し）はそのままエラーを返す。
			require.Error(t, guardedDialControl(deny, "tcp", "noport", nil))
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
			// RFC 6890 IETF プロトコル割当（192.0.0.0/24）— 一般宛て通信には使われない。
			require.Error(t, guardedDialControl(allow, "tcp", "192.0.0.1:80", nil))
			// RFC 3849 IPv6 ドキュメント用（2001:db8::/32）— テスト/文書専用で実到達不能。
			require.Error(t, guardedDialControl(allow, "tcp", "[2001:db8::1]:443", nil))
		})
	})
}

func TestMustParseCIDR(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正当なCIDRリテラルを*net.IPNetへ解析する", func(t *testing.T) {
			t.Parallel()

			n := mustParseCIDR("10.0.0.0/8")

			require.NotNil(t, n)
			assert.True(t, n.Contains(net.ParseIP("10.1.2.3")))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正なCIDRリテラルはpanicする", func(t *testing.T) {
			t.Parallel()

			assert.Panics(t, func() { mustParseCIDR("not-a-cidr") })
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

func TestConditionalPropagator(t *testing.T) {
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
