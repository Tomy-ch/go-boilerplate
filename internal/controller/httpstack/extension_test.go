package httpstack

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestApplyPreMiddlewares_SetsHeader(t *testing.T) {
	t.Parallel()

	e := echo.New()
	// pre middleware sets a header
	pre := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("X-Pre", "ok")
			return next(c)
		}
	}

	ApplyPreMiddlewares(e, zap.NewNop(), []echo.MiddlewareFunc{pre})

	e.GET("/", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, "ok", rec.Header().Get("X-Pre"))
}

func TestApplyUseMiddlewares_Order(t *testing.T) {
	t.Parallel()

	e := echo.New()

	mwA := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Add("X-Order", "A")
			return next(c)
		}
	}
	mwB := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Add("X-Order", "B")
			return next(c)
		}
	}

	// provide out-of-order priorities to ensure sorting happens
	mws := []UseMiddleware{
		{Priority: 2, Middleware: mwB},
		{Priority: 1, Middleware: mwA},
	}

	ApplyUseMiddlewares(e, zap.NewNop(), mws)

	e.GET("/", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	vals := rec.Header()["X-Order"]
	// expect sorted by Priority ascending: A then B
	require.Equal(t, []string{"A", "B"}, vals)
}

func TestApplyConfigurators_RegistersRoute(t *testing.T) {
	t.Parallel()

	e := echo.New()

	cfg := func(e *echo.Echo) {
		e.GET("/cfg", func(c echo.Context) error {
			c.Response().Header().Set("X-Cfg", "yes")
			return c.NoContent(http.StatusNoContent)
		})
	}

	ApplyConfigurators(e, zap.NewNop(), []SrvCfg{cfg})

	req := httptest.NewRequest(http.MethodGet, "/cfg", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "yes", rec.Header().Get("X-Cfg"))
}

func TestApplyExtends_Integration(t *testing.T) {
	t.Parallel()

	e := echo.New()

	pre := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("X-Pre", "ok")
			return next(c)
		}
	}
	mw := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Add("X-Order", "1")
			return next(c)
		}
	}
	cfg := func(e *echo.Echo) {
		e.GET("/ext", func(c echo.Context) error {
			return c.String(http.StatusOK, "done")
		})
	}

	extends := ServerExtends{
		PreList: []echo.MiddlewareFunc{pre},
		UseList: []UseMiddleware{{Priority: 0, Middleware: mw}},
		CfgList: []SrvCfg{cfg},
	}

	ApplyExtends(e, zap.NewNop(), extends)

	req := httptest.NewRequest(http.MethodGet, "/ext", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ok", rec.Header().Get("X-Pre"))
	require.Equal(t, "1", rec.Header().Get("X-Order"))
}

func TestApplyFunctions_HandleEmptySlices_NoPanic(t *testing.T) {
	t.Parallel()

	e := echo.New()
	// ensure calling with nil/empty slices does not panic
	ApplyPreMiddlewares(e, zap.NewNop(), nil)
	ApplyUseMiddlewares(e, zap.NewNop(), nil)
	ApplyConfigurators(e, zap.NewNop(), nil)

	// still able to register and serve a route
	e.GET("/ok", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
