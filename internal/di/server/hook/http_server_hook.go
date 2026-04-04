package hook

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
)

const serverCallerSkip = 3

// RegisterHTTPServerHooks は、HTTPサーバーを起動します。
func RegisterHTTPServerHooks(
	e *echo.Echo,
	reg lifecycle.Registrar,
	log logging.Logger,
	appCfg *config.ApplicationConfig,
	secCfg *config.SecurityConfig,
	srvCfg *config.ServerConfig,
	osCfg *config.OperationSystemConfig,
	// 下記はサーバー機能の拡張が適用されたことを示すトークン
	_ *extension.AppliedServerExtends,
) {
	reg.RegisterStart(newStartServerFunc(e, srvCfg, log, secCfg, appCfg, osCfg))
	reg.RegisterStop(newStopServerFunc(e, log, osCfg))
}

// newStartServerFunc は、HTTPサーバーを起動する関数を生成します。
func newStartServerFunc(
	e *echo.Echo,
	srvCfg *config.ServerConfig,
	log logging.Logger,
	secCfg *config.SecurityConfig,
	appCfg *config.ApplicationConfig,
	osCfg *config.OperationSystemConfig,
) func(context.Context) error {
	return func(ctx context.Context) error {
		addr := srvCfg.Port()

		lc := &net.ListenConfig{}
		ln, err := lc.Listen(ctx, "tcp", ":"+strconv.Itoa(addr))
		if err != nil {
			return fmt.Errorf("failed to listen on port %d: %w", addr, err)
		}
		e.Listener = ln

		go func() {
			if err := e.Start(""); err != nil && !xerrors.Is(err, http.ErrServerClosed) {
				log.Named("server.Start").Error("failed to start http server", logging.Error("e.Start", err))
			}
		}()
		log.Named("server.Start").CallerSkip(serverCallerSkip).Info(
			"http started",
			logging.String(logging.EventTypeKey, logging.EventTypeStart),
			logging.Time(logging.EventAtKey, time.Now()),
			logging.String(logging.EventTzKey, osCfg.TimeZone()),
			logging.String("port", strconv.Itoa(addr)),
			logging.Strings("allowed_origins", secCfg.AllowedOrigins()),
			logging.String("cidr", secCfg.CIDR().IP.String()),
			logging.String("mode", appCfg.Mode()),
		)
		return nil
	}
}

// newStopServerFunc は、HTTPサーバーを停止する関数を生成します。
func newStopServerFunc(
	e *echo.Echo, log logging.Logger, osCfg *config.OperationSystemConfig,
) func(context.Context) error {
	return func(ctx context.Context) error {
		log.Named("server.Stop").CallerSkip(serverCallerSkip).Info(
			"http stopping",
			logging.String(logging.EventTypeKey, logging.EventTypeEnd),
			logging.Time(logging.EventAtKey, time.Now()),
			logging.String(logging.EventTzKey, osCfg.TimeZone()),
		)
		return e.Shutdown(ctx)
	}
}
