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
)

// serverCallerSkip は、fx ライフサイクル経由のクロージャ呼び出しで挟まるフレームを飛ばしログの caller を実呼び出し位置へ合わせる段数です。
const serverCallerSkip = 3

// RegisterHTTPServerHooks は、HTTPサーバーの起動・停止フックをライフサイクルに登録します。
func RegisterHTTPServerHooks(
	srv *http.Server,
	reg lifecycle.Registrar,
	log logging.Logger,
	appCfg *config.ApplicationConfig,
	secCfg *config.SecurityConfig,
	srvCfg *config.ServerConfig,
	osCfg *config.OperatingSystemConfig,
	// 下記はサーバー機能の拡張が適用されたことを示すトークン
	_ *extension.AppliedServerExtends,
) {
	reg.RegisterStart(newStartServerFunc(srv, log, appCfg, secCfg, srvCfg, osCfg))
	reg.RegisterStop(newStopServerFunc(srv, log, osCfg))
}

// newStartServerFunc は、HTTPサーバーを起動する関数を生成します。
func newStartServerFunc(
	srv *http.Server,
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
			return xerrors.Wrap(err, fmt.Sprintf("failed to listen on port %d", port))
		}
		go serveHTTP(ctx, srv, ln, log)

		fields := append(lifecycleEventFields(logging.EventTypeStart, osCfg.TimeZone()),
			logging.String("port", strconv.Itoa(port)),
			logging.Strings("allowed_origins", secCfg.AllowedOrigins()),
			logging.String("cidr", secCfg.CIDR().IP.String()),
			logging.String("mode", appCfg.Mode()),
		)
		log.Named("server.Start").CallerSkip(serverCallerSkip).Info(ctx, "http started", fields...)
		return nil
	}
}

// serveHTTP は、ln 上で HTTP サーバーを稼働させます。
// 正常停止（[http.ErrServerClosed]）以外で終了した場合はエラーログを残します。
func serveHTTP(ctx context.Context, srv *http.Server, ln net.Listener, log logging.Logger) {
	if err := srv.Serve(ln); err != nil && !xerrors.Is(err, http.ErrServerClosed) {
		log.Named("server.Start").Error(ctx, "failed to start http server", logging.Error(logging.ErrorKey, err))
	}
}

// newStopServerFunc は、HTTPサーバーを停止する関数を生成します。
func newStopServerFunc(
	srv *http.Server, log logging.Logger, osCfg *config.OperatingSystemConfig,
) func(context.Context) error {
	return func(ctx context.Context) error {
		l := log.Named("server.Stop").CallerSkip(serverCallerSkip)
		l.Info(ctx, "http stopping", lifecycleEventFields(logging.EventTypeEnd, osCfg.TimeZone())...)
		if err := srv.Shutdown(ctx); err != nil {
			l.Error(ctx, "failed to shutdown http server", logging.Error(logging.ErrorKey, err))
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
