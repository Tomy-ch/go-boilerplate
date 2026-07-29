package jwt

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/infrastructure/httpclient"
	mock_httpclient "go-boilerplate/internal/infrastructure/httpclient/mock"
)

const (
	discoveryIssuer  = "https://issuer.example.test"
	discoveryJWKSURI = "https://issuer.example.test/keys.json"
)

// discoveryDoc は、issuer / jwks_uri を持つ discovery 応答 JSON を組み立てます。
func discoveryDoc(t *testing.T, issuer, jwksURI string) []byte {
	t.Helper()
	raw, err := json.Marshal(openidConfiguration{Issuer: issuer, JWKSURI: jwksURI})
	require.NoError(t, err)
	return raw
}

// newHTTPSDiscovery は、https 前提（allowInsecure=false）の discovery 解決器を生成します。
func newHTTPSDiscovery(client httpclient.Client, clk *fakeClock) *discoveryResolver {
	return newDiscoveryResolver(client, discoveryIssuer, time.Hour, clk, false)
}

func Test_newDiscoveryResolver(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("issuer から discovery URL を導出する", func(t *testing.T) {
			t.Parallel()
			d := newDiscoveryResolver(stubJWKSClient(t, nil), discoveryIssuer, time.Hour, newFakeClock(fixedNow), false)
			assert.Equal(t, discoveryIssuer+"/.well-known/openid-configuration", d.discoveryURL)
			assert.Equal(t, time.Hour, d.ttl)
		})

		t.Run("ttl が 0 以下なら既定 discovery TTL が適用される", func(t *testing.T) {
			t.Parallel()
			d := newDiscoveryResolver(stubJWKSClient(t, nil), discoveryIssuer, 0, newFakeClock(fixedNow), false)
			assert.Equal(t, defaultDiscoveryTTL, d.ttl)
		})
	})
}

func Test_discoveryResolver_jwksURL(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("鮮度内キャッシュがあれば再取得せず jwks_uri を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).
				Return(&httpclient.Response{StatusCode: 200, Body: discoveryDoc(t, discoveryIssuer, discoveryJWKSURI)}, nil).
				Times(1)
			d := newHTTPSDiscovery(client, newFakeClock(fixedNow))

			u1, err := d.jwksURL(context.Background())
			require.NoError(t, err)
			assert.Equal(t, discoveryJWKSURI, u1)

			u2, err := d.jwksURL(context.Background())
			require.NoError(t, err)
			assert.Equal(t, discoveryJWKSURI, u2)
		})

		t.Run("TTL 経過後は discovery を再取得し jwks_uri を更新する", func(t *testing.T) {
			t.Parallel()
			clk := newFakeClock(fixedNow)
			d := newHTTPSDiscovery(jwksClientReturning(t,
				discoveryDoc(t, discoveryIssuer, discoveryIssuer+"/keys1.json"),
				discoveryDoc(t, discoveryIssuer, discoveryIssuer+"/keys2.json"),
			), clk)

			u1, err := d.jwksURL(context.Background())
			require.NoError(t, err)
			assert.Equal(t, discoveryIssuer+"/keys1.json", u1)

			clk.advance(time.Hour + time.Minute)
			u2, err := d.jwksURL(context.Background())
			require.NoError(t, err)
			assert.Equal(t, discoveryIssuer+"/keys2.json", u2)
		})
	})
}

func Test_discoveryResolver_cached(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("鮮度内なら jwks_uri を返す", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(stubJWKSClient(t, nil), newFakeClock(fixedNow))
			d.jwksURI = discoveryJWKSURI
			d.fetchedAt = fixedNow
			assert.Equal(t, discoveryJWKSURI, d.cached())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未取得（fetchedAt ゼロ値）なら空を返す", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(stubJWKSClient(t, nil), newFakeClock(fixedNow))
			assert.Empty(t, d.cached())
		})

		t.Run("TTL 超過なら空を返す", func(t *testing.T) {
			t.Parallel()
			clk := newFakeClock(fixedNow)
			d := newHTTPSDiscovery(stubJWKSClient(t, nil), clk)
			d.jwksURI = discoveryJWKSURI
			d.fetchedAt = fixedNow
			clk.advance(time.Hour + time.Minute)
			assert.Empty(t, d.cached())
		})
	})
}

func Test_discoveryResolver_refresh(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得成功時は jwks_uri / fetchedAt を更新する", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(stubJWKSClient(t, discoveryDoc(t, discoveryIssuer, discoveryJWKSURI)), newFakeClock(fixedNow))

			u, err := d.refresh(context.Background())
			require.NoError(t, err)
			assert.Equal(t, discoveryJWKSURI, u)
			assert.Equal(t, fixedNow, d.fetchedAt)
			assert.Equal(t, discoveryJWKSURI, d.jwksURI)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得失敗時は原因を返し fetchedAt を更新しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			client := mock_httpclient.NewMockClient(ctrl)
			client.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil, errBoom).Times(1)
			d := newHTTPSDiscovery(client, newFakeClock(fixedNow))

			_, err := d.refresh(context.Background())
			require.ErrorIs(t, err, errBoom)
			assert.True(t, d.fetchedAt.IsZero())
		})
	})
}

func Test_discoveryResolver_fetch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("issuer 一致・https・同一オリジンなら jwks_uri を返す", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(stubJWKSClient(t, discoveryDoc(t, discoveryIssuer, discoveryJWKSURI)), newFakeClock(fixedNow))

			u, err := d.fetch(context.Background())
			require.NoError(t, err)
			assert.Equal(t, discoveryJWKSURI, u)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("issuer 不一致は拒否する", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(
				stubJWKSClient(t, discoveryDoc(t, "https://evil.example.test", discoveryJWKSURI)),
				newFakeClock(fixedNow),
			)

			_, err := d.fetch(context.Background())
			require.ErrorIs(t, err, errDiscoveryIssuerMismatch)
		})

		t.Run("jwks_uri が無い場合は拒否する", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(stubJWKSClient(t, discoveryDoc(t, discoveryIssuer, "")), newFakeClock(fixedNow))

			_, err := d.fetch(context.Background())
			require.ErrorIs(t, err, errDiscoveryNoJWKSURI)
		})

		t.Run("jwks_uri が issuer と別オリジンなら拒否する", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(
				stubJWKSClient(t, discoveryDoc(t, discoveryIssuer, "https://cdn.example.test/keys.json")),
				newFakeClock(fixedNow),
			)

			_, err := d.fetch(context.Background())
			require.ErrorIs(t, err, errDiscoveryUntrustedJWKS)
		})

		t.Run("discovery 応答の jwks_uri が http なら拒否する", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(
				stubJWKSClient(t, discoveryDoc(t, discoveryIssuer, "http://issuer.example.test/keys.json")),
				newFakeClock(fixedNow),
			)

			_, err := d.fetch(context.Background())
			require.ErrorIs(t, err, errDiscoveryInsecureURL)
		})

		t.Run("JSON が不正な場合は拒否する", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(stubJWKSClient(t, []byte("{")), newFakeClock(fixedNow))

			_, err := d.fetch(context.Background())
			require.ErrorIs(t, err, errDiscoveryMalformed)
		})
	})
}

func Test_requireSecureURL(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("https は許可する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, requireSecureURL("https://issuer.example.test/x", false))
		})

		t.Run("allowInsecure なら検証しない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, requireSecureURL("http://issuer.example.test/x", true))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("allowInsecure でない http は拒否する", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, requireSecureURL("http://issuer.example.test/x", false), errDiscoveryInsecureURL)
		})
	})
}

func Test_discoveryResolver_requireSameOrigin(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一オリジンは許可する", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(stubJWKSClient(t, nil), newFakeClock(fixedNow))
			require.NoError(t, d.requireSameOrigin(discoveryJWKSURI))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("別ホストは拒否する", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(stubJWKSClient(t, nil), newFakeClock(fixedNow))
			require.ErrorIs(t, d.requireSameOrigin("https://cdn.example.test/keys.json"), errDiscoveryUntrustedJWKS)
		})

		t.Run("同一ホストでもポートが異なれば拒否する", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(stubJWKSClient(t, nil), newFakeClock(fixedNow))
			require.ErrorIs(t, d.requireSameOrigin("https://issuer.example.test:8443/keys.json"), errDiscoveryUntrustedJWKS)
		})

		t.Run("scheme が異なれば拒否する", func(t *testing.T) {
			t.Parallel()
			d := newHTTPSDiscovery(stubJWKSClient(t, nil), newFakeClock(fixedNow))
			require.ErrorIs(t, d.requireSameOrigin("http://issuer.example.test/keys.json"), errDiscoveryUntrustedJWKS)
		})
	})
}
