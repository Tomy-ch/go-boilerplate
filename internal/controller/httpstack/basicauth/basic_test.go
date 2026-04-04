package basicauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestNewBasicAuthValidator(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	mtc := config.NewMetricsConfig(cfg)
	validator := NewBasicAuthValidator(mtc)

	t.Run("valid credentials", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		ctx := context.Background()

		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		ok, err := validator(mtc.UserName(), mtc.Password(), c)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		ctx := context.Background()

		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		ok, err := validator("bad", "bad", c)
		require.False(t, ok)
		require.ErrorIs(t, err, apperror.ErrUnauthenticated)
	})
}
