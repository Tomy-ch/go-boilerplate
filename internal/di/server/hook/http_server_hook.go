// Package hook は、起動と停止のライフサイクルフックを提供します。
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

// serverCallerSkip は、fx ライフサイクル経由のクロージャ呼び出しで挟まるフレームを飛ばしログの caller を実呼び出し位置へ合わせる段数です。
const serverCallerSkip = 3

// RegisterHTTPServerHooks は、HTTPサーバーの起動・停止フックをライフサイクルに登録します。
func RegisterHTTPServerHooks(
	e *echo.Echo,
	reg lifecycle.Registrar,
	log logging.Logger,
	appCfg *config.ApplicationConfig,
	secCfg *config.SecurityConfig,
	srvCfg *config.ServerConfig,
	osCfg *config.OperatingSystemConfig,
	// 下記はサーバー機能の拡張が適用されたことを示すトークン
	_ *extension.AppliedServerExtends,
) {
	reg.RegisterStart(newStartServerFunc(e, log, appCfg, secCfg, srvCfg, osCfg))
	reg.RegisterStop(newStopServerFunc(e, log, osCfg))
}

// newStartServerFunc は、HTTPサーバーを起動する関数を生成します。
func newStartServerFunc(
	e *echo.Echo,
	log logging.Logger,
	appCfg *config.ApplicationConfig,
	secCfg *config.SecurityConfig,
	srvCfg *config.ServerConfig,
	osCfg *config.OperatingSystemConfig,
) func(context.Context) error {
	return func(ctx context.Context) error {
		port := srvCfg.Port()

		lc := &net.ListenConfig{}
		ln, err := lc.Listen(ctx, "tcp", ":"+strconv.Itoa(port))
		if err != nil {
			return fmt.Errorf("failed to listen on port %d: %w", port, err)
		}
		e.Listener = ln

		go func() {
			if err := e.Start(""); err != nil && !xerrors.Is(err, http.ErrServerClosed) {
				log.Named("server.Start").Error("failed to start http server", logging.Error(logging.ErrorKey, err))
			}
		}()
		fields := append(lifecycleEventFields(logging.EventTypeStart, osCfg.TimeZone()),
			logging.String("port", strconv.Itoa(port)),
			logging.Strings("allowed_origins", secCfg.AllowedOrigins()),
			logging.String("cidr", secCfg.CIDR().IP.String()),
			logging.String("mode", appCfg.Mode()),
		)
		log.Named("server.Start").CallerSkip(serverCallerSkip).Info("http started", fields...)
		return nil
	}
}

// newStopServerFunc は、HTTPサーバーを停止する関数を生成します。
func newStopServerFunc(
	e *echo.Echo, log logging.Logger, osCfg *config.OperatingSystemConfig,
) func(context.Context) error {
	return func(ctx context.Context) error {
		l := log.Named("server.Stop").CallerSkip(serverCallerSkip)
		l.Info("http stopping", lifecycleEventFields(logging.EventTypeEnd, osCfg.TimeZone())...)
		if err := e.Shutdown(ctx); err != nil {
			l.Error("failed to shutdown http server", logging.Error(logging.ErrorKey, err))
			return err
		}
		return nil
	}
}

// lifecycleEventFields は、起動／停止ログ共通のイベントフィールド（種別・発生時刻・TZ）を返します。
func lifecycleEventFields(eventType, tz string) []*logging.Field {
	return []*logging.Field{
		logging.String(logging.EventTypeKey, eventType),
		logging.Time(logging.EventAtKey, time.Now()),
		logging.String(logging.EventTzKey, tz),
	}
}
