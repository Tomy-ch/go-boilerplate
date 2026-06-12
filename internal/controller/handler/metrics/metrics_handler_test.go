package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/basicauth"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestBindHandler_BasicAuth(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	mtc := config.NewMetricsConfig(cfg)

	exec := func(t *testing.T, withAuth bool, user, pass string) int {
		t.Helper()
		e := echo.New()
		BindHandler(e, basicauth.NewBasicAuthValidator(mtc))

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
		if withAuth {
			req.SetBasicAuth(user, pass)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec.Code
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正しい認証情報で200を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, http.StatusOK, exec(t, true, mtc.UserName(), mtc.Password()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報なしで401を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, http.StatusUnauthorized, exec(t, false, "", ""))
		})

		t.Run("不正な認証情報で401を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, http.StatusUnauthorized, exec(t, true, "wrong-user", "wrong-pass"))
		})
	})
}
