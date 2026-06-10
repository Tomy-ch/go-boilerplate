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

	tests := []struct {
		name     string
		user     string
		pass     string
		withAuth bool
		want     int
	}{
		{name: "正常系_正しい認証情報で200", user: mtc.UserName(), pass: mtc.Password(), withAuth: true, want: http.StatusOK},
		{name: "異常系_認証情報なしで401", withAuth: false, want: http.StatusUnauthorized},
		{name: "異常系_不正な認証情報で401", user: "wrong-user", pass: "wrong-pass", withAuth: true, want: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			BindHandler(e, basicauth.NewBasicAuthValidator(mtc))

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
			if tt.withAuth {
				req.SetBasicAuth(tt.user, tt.pass)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code)
		})
	}
}
