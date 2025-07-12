package handler

import (
	errUtil "boilerplate-go/pkg/errutil"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo, xerrors errUtil.XErrors) {
	e.GET("/health", func(c echo.Context) error {
		return xerrors.New("health check failed")
		// return c.String(http.StatusOK, "ok")
	})
}
