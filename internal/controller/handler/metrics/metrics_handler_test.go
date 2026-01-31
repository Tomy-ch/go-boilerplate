package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestBindHandler_BasicAuth(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	mtc := config.NewMetricsConfig(cfg)
	validator := func(username, password string, _ echo.Context) (bool, error) {
		if username == mtc.UserName() && password == mtc.Password() {
			return true, nil
		}
		return false, nil
	}

	e := echo.New()
	BindHandler(e, validator)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.SetBasicAuth(mtc.UserName(), mtc.Password())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusUnauthorized, rec2.Code)
}
