package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"boilerplate-go/internal/controller/httpstack"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestMountRoutes(t *testing.T) {
	t.Parallel()

	e := echo.New()
	appliedExtends := &httpstack.AppliedServerExtends{}

	const (
		route1         = "/route1"
		responseRoute1 = "route1 response"

		route2         = "/route2"
		responseRoute2 = "route2 response"
	)

	registrar1 := func(e *echo.Echo) {
		e.GET(route1, func(c echo.Context) error {
			return c.String(http.StatusOK, responseRoute1)
		})
	}

	registrar2 := func(e *echo.Echo) {
		e.GET(route2, func(c echo.Context) error {
			return c.String(http.StatusOK, responseRoute2)
		})
	}

	in := RouteMountIn{
		Registrars: []RouteMount{registrar1, registrar2},
	}

	MountRoutes(e, in, appliedExtends)

	t.Run("route1に登録したレスポンスが取得できる", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, route1, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, responseRoute1, rec.Body.String())
	})

	t.Run("route2に登録したレスポンスが取得できる", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, route2, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, responseRoute2, rec.Body.String())
	})
}
