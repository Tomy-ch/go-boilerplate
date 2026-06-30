package worker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readBody は、レスポンスボディを文字列で読み出すテストヘルパです。
func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return string(body)
}

// httpGet は、context 付きで GET リクエストを実行するテストヘルパです。
func httpGet(t *testing.T, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return res
}

func Test_healthMux(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("healthz は常に 200 を返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(healthMux(func() bool { return false }))
			defer srv.Close()

			res := httpGet(t, srv.URL+"/healthz")
			defer func() { _ = res.Body.Close() }()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "ok", readBody(t, res))
		})

		t.Run("readyz は ready が true なら 200 を返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(healthMux(func() bool { return true }))
			defer srv.Close()

			res := httpGet(t, srv.URL+"/readyz")
			defer func() { _ = res.Body.Close() }()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "ready", readBody(t, res))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("readyz は ready が false なら 503 を返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(healthMux(func() bool { return false }))
			defer srv.Close()

			res := httpGet(t, srv.URL+"/readyz")
			defer func() { _ = res.Body.Close() }()
			assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
			assert.Equal(t, "not ready", readBody(t, res))
		})
	})
}

func TestNewHealthServer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("起動から停止までのライフサイクルがpanicせず完了する", func(t *testing.T) {
			t.Parallel()

			// 固定ポートの bind 衝突を避けるため、エフェメラルポート(:0)で待ち受けさせる。
			start, stop := NewHealthServer(":0", func() bool { return true }, logging.NewTestLogger(t))
			require.NotNil(t, start)
			require.NotNil(t, stop)

			assert.NotPanics(t, func() {
				start()
				stop(context.Background())
			})
		})

		t.Run("起動していないサーバーへの停止でもpanicしない", func(t *testing.T) {
			t.Parallel()

			_, stop := NewHealthServer(":0", func() bool { return true }, logging.NewTestLogger(t))

			// 起動していないサーバーへの Shutdown はエラーにならず、panic しないこと。
			assert.NotPanics(t, func() { stop(context.Background()) })
		})
	})
}
