package worker

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_healthMux(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("healthz は常に 200 を返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(healthMux(func() bool { return false }))
			defer srv.Close()

			res, err := http.Get(srv.URL + "/healthz")
			require.NoError(t, err)
			defer func() { _ = res.Body.Close() }()
			assert.Equal(t, http.StatusOK, res.StatusCode)
		})

		t.Run("readyz は ready が true なら 200 を返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(healthMux(func() bool { return true }))
			defer srv.Close()

			res, err := http.Get(srv.URL + "/readyz")
			require.NoError(t, err)
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

			res, err := http.Get(srv.URL + "/readyz")
			require.NoError(t, err)
			defer func() { _ = res.Body.Close() }()
			assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
		})
	})
}
