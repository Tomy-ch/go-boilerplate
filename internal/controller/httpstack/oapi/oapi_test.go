package oapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/controller/ctxhelper"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	spec := &openapi3.T{}
	mw := Middleware(spec, nil, nil)

	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	handler := mw(func(_ echo.Context) error {
		return nil
	})

	_ = handler(c)

	got, ok := ctxhelper.GetEchoContext(c.Request().Context())
	require.True(t, ok)
	require.Equal(t, c, got)
}
