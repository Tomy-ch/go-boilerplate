package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// httpGet は、context 付きで GET リクエストを実行するテストヘルパです（noctx 回避）。
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
		})

		t.Run("readyz は ready が true なら 200 を返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(healthMux(func() bool { return true }))
			defer srv.Close()

			res := httpGet(t, srv.URL+"/readyz")
			defer func() { _ = res.Body.Close() }()
			assert.Equal(t, http.StatusOK, res.StatusCode)
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
		})
	})
}
