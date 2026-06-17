package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	mw := Middleware("test-service")
	require.NotNil(t, mw)
}

func TestMiddleware_Integration(t *testing.T) {
	t.Parallel()

	e := echo.New()
	e.Use(Middleware("test-service"))

	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.NotNil(t, rec)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
