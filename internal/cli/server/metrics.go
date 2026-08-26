package server

import (
	"context"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // G108: 非本番環境でのプロファイリング用にインポート
	"strconv"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
)

// metricsServer は、メトリクス/pprof（DefaultServeMux）を公開する補助 HTTP サーバーを生成します。
func metricsServer(mtcCfg *config.MetricsConfig) *http.Server {
	return &http.Server{
		Addr:              mtcCfg.Host() + ":" + strconv.Itoa(mtcCfg.Port()),
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// NewMetricsServer は、メトリクスサーバーの開始および終了関数を生成します。
func NewMetricsServer(mtcCfg *config.MetricsConfig, logger logging.Logger) (func(), func(ctx context.Context)) {
	metricsSrv := metricsServer(mtcCfg)

	start := func() {
		go func() {
			logListenError(logger, metricsSrv.ListenAndServe())
		}()
	}
	end := func(ctx context.Context) {
		// 停止失敗でアプリ本体を巻き込まないよう、panic せずログに留める（start 側の goroutine と同方針）。
		if err := metricsSrv.Shutdown(ctx); err != nil {
			logger.Named("metrics.Shutdown").Error(ctx, "metrics server shutdown error", logging.Error(logging.ErrorKey, err))
		}
	}
	return start, end
}

// logListenError は、補助サーバーの ListenAndServe 戻り値を判定し、正常停止(ErrServerClosed)以外の
// エラーのみをログ記録します。bind 失敗等でアプリ本体を巻き込まないよう、panic せずログに留めます。
func logListenError(logger logging.Logger, err error) {
	if err != nil && !xerrors.Is(err, http.ErrServerClosed) {
		logger.Named("metrics.ListenAndServe").Error(context.Background(), "metrics server error", logging.Error(logging.ErrorKey, err))
	}
}
