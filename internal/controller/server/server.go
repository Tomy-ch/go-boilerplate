package server

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/middleware/logging"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func New(
	cfg *config.Config,
	logger *zap.Logger,
) *echo.Echo {
	e := echo.New()
	e.Use(logging.LoggingMiddleware(logger, cfg))

	return e
}
