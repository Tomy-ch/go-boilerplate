package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

// capturingRoundTripper は、委譲されたリクエストを記録するだけの RoundTripper です。
type capturingRoundTripper struct {
	got *http.Request
}

func (rt *capturingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.got = req
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

func TestNewOutboundHTTPClient(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ガード付き transport を持つクライアントを返す", func(t *testing.T) {
			t.Parallel()

			got := NewOutboundHTTPClient(NewHTTPClientTransport(noop.NewTracerProvider(), NewTextMapPropagator()), true)

			require.NotNil(t, got)
			require.NotNil(t, got.Client)
			assert.IsType(t, policyStampingRoundTripper{}, got.Transport)
		})

		t.Run("リダイレクトを追従しない", func(t *testing.T) {
			t.Parallel()

			got := NewOutboundHTTPClient(NewHTTPClientTransport(noop.NewTracerProvider(), NewTextMapPropagator()), false)

			require.NotNil(t, got.CheckRedirect)
			assert.Equal(t, http.ErrUseLastResponse, got.CheckRedirect(nil, nil))
		})
	})
}

func TestNewDisabledOutboundHTTPClient(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("計装なしでもガード付き transport を持つ", func(t *testing.T) {
			t.Parallel()

			got := NewDisabledOutboundHTTPClient(true)

			require.NotNil(t, got)
			assert.IsType(t, policyStampingRoundTripper{}, got.Transport)
		})

		t.Run("private 網宛てを許可する設定ではローカルの HTTP サーバーへ到達できる", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			resp, err := NewDisabledOutboundHTTPClient(true).Do(req)

			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		t.Run("private 網宛てを許可しない設定ではローカルの HTTP サーバーへ到達できない", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			_, err = NewDisabledOutboundHTTPClient(false).Do(req)

			require.Error(t, err)
		})
	})
}

func Test_policyStampingRoundTripper_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("方針を ctx へ積んで内側へ委譲する", func(t *testing.T) {
			t.Parallel()

			for _, allowed := range []bool{true, false} {
				inner := &capturingRoundTripper{}
				rt := policyStampingRoundTripper{inner: inner, allowPrivateNetwork: allowed}
				req, err := http.NewRequestWithContext(
					context.Background(), http.MethodGet, "http://example.com", nil)
				require.NoError(t, err)

				_, err = rt.RoundTrip(req)

				require.NoError(t, err)
				got, ok := inner.got.Context().Value(allowPrivateNetworkKey{}).(bool)
				require.True(t, ok, "allowPrivateNetwork が ctx に積まれていない")
				assert.Equal(t, allowed, got)
			}
		})

		t.Run("呼び出し元の Request を変更しない", func(t *testing.T) {
			t.Parallel()

			inner := &capturingRoundTripper{}
			rt := policyStampingRoundTripper{inner: inner, allowPrivateNetwork: true}
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
			require.NoError(t, err)

			_, err = rt.RoundTrip(req)

			require.NoError(t, err)
			assert.Nil(t, req.Context().Value(allowPrivateNetworkKey{}))
		})
	})
}

func Test_noFollowRedirectForOutbound(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("常に ErrUseLastResponse を返して追従を止める", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, http.ErrUseLastResponse, noFollowRedirectForOutbound(nil, nil))
		})
	})
}
