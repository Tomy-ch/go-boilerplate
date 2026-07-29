package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace/noop"
)

// errTestDialBlocked は、注入した dial control が接続を拒否したことを示すテスト用センチネルです。
var errTestDialBlocked = xerrors.New("dial blocked by test control")

func TestContextWithAllowPrivateNetwork(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("trueを設定するとSSRFガードがloopback宛てを許可する", func(t *testing.T) {
			t.Parallel()

			ctx := ContextWithAllowPrivateNetwork(context.Background(), true)

			require.NoError(t, guardedDialControl(ctx, "tcp", "127.0.0.1:8080", nil))
		})

		t.Run("falseを設定するとSSRFガードがloopback宛てを拒否する", func(t *testing.T) {
			t.Parallel()

			ctx := ContextWithAllowPrivateNetwork(context.Background(), false)

			require.ErrorIs(t, guardedDialControl(ctx, "tcp", "127.0.0.1:8080", nil), errSSRFPrivateAddress)
		})

		t.Run("元のctxは変更されず未設定のまま残る", func(t *testing.T) {
			t.Parallel()

			base := context.Background()
			_ = ContextWithAllowPrivateNetwork(base, true)

			assert.False(t, allowPrivateNetworkFromContext(base))
		})
	})
}

func TestContextWithTracePropagation(t *testing.T) {
	t.Parallel()

	prop := conditionalPropagator{inner: propagation.TraceContext{}}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("trueを設定するとtraceparentが注入される", func(t *testing.T) {
			t.Parallel()

			ctx := ContextWithTracePropagation(newSampledContext(), true)
			carrier := propagation.MapCarrier{}

			prop.Inject(ctx, carrier)

			assert.NotEmpty(t, carrier.Get("traceparent"))
		})

		t.Run("falseを設定するとtraceparentの注入が抑止される", func(t *testing.T) {
			t.Parallel()

			ctx := ContextWithTracePropagation(newSampledContext(), false)
			carrier := propagation.MapCarrier{}

			prop.Inject(ctx, carrier)

			assert.Empty(t, carrier.Get("traceparent"))
		})

		t.Run("元のctxは変更されず未設定のまま残る", func(t *testing.T) {
			t.Parallel()

			base := newSampledContext()
			_ = ContextWithTracePropagation(base, false)

			carrier := propagation.MapCarrier{}
			prop.Inject(base, carrier)

			assert.NotEmpty(t, carrier.Get("traceparent"))
		})
	})
}

func TestHTTPClientTransport_RoundTripper(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("span用URL redactionを最外層に持つRoundTripperを返す", func(t *testing.T) {
			t.Parallel()

			transport := newHTTPClientTransport(noop.NewTracerProvider(), NewTextMapPropagator(), permissiveDialControl)

			redacting, ok := transport.RoundTripper().(spanURLRedactingRoundTripper)
			require.True(t, ok)
			assert.NotNil(t, redacting.inner)
		})
	})
}

func Test_allowPrivateNetworkFromContext(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("許可フラグにtrueが設定されていればtrueを返す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, allowPrivateNetworkFromContext(ContextWithAllowPrivateNetwork(context.Background(), true)))
		})

		t.Run("許可フラグにfalseが設定されていればfalseを返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, allowPrivateNetworkFromContext(ContextWithAllowPrivateNetwork(context.Background(), false)))
		})

		t.Run("許可フラグが未設定なら安全側のfalseを返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, allowPrivateNetworkFromContext(context.Background()))
		})

		t.Run("許可フラグがbool以外の型なら安全側のfalseを返す", func(t *testing.T) {
			t.Parallel()

			// 同じキーへ別の型が入っても許可へ倒れない（型アサート失敗時の fail-close）。
			ctx := context.WithValue(context.Background(), allowPrivateNetworkKey{}, "true")

			assert.False(t, allowPrivateNetworkFromContext(ctx))
		})
	})
}

func Test_newHTTPClientTransport(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注入したdial controlを経由して宛先へ接続する", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			var gotAddr string
			control := func(_ context.Context, _, address string, _ syscall.RawConn) error {
				gotAddr = address
				return nil
			}

			transport := newHTTPClientTransport(noop.NewTracerProvider(), NewTextMapPropagator(), control)
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)

			resp, err := (&http.Client{Transport: transport.RoundTripper()}).Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
			// 既定の guardedDialControl ではなく注入した control が dial 判定に使われている。
			assert.Equal(t, srv.Listener.Addr().String(), gotAddr)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("dial controlが拒否した宛先へは接続せずエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			control := func(context.Context, string, string, syscall.RawConn) error { return errTestDialBlocked }

			transport := newHTTPClientTransport(noop.NewTracerProvider(), NewTextMapPropagator(), control)
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)

			resp, err := (&http.Client{Transport: transport.RoundTripper()}).Do(req)
			require.ErrorIs(t, err, errTestDialBlocked)
			assert.Nil(t, resp)
		})
	})
}

func Test_permissiveDialControl(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SSRFガードが拒否する宛先も含めて全て許可する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// guardedDialControl が拒否する宛先（loopback / link-local / 予約帯）でも nil を返す。
			require.NoError(t, permissiveDialControl(ctx, "tcp", "127.0.0.1:8080", nil))
			require.NoError(t, permissiveDialControl(ctx, "tcp", "169.254.169.254:80", nil))
			require.NoError(t, permissiveDialControl(ctx, "tcp", "192.0.2.1:80", nil))
			// パース不能なアドレスでも fail-close しない（テスト用の素通し）。
			require.NoError(t, permissiveDialControl(ctx, "tcp", "noport", nil))
		})
	})
}
