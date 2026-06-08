package server

import (
	"context"
	"errors"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // G108: 非本番環境でのプロファイリング用にインポート
	"strconv"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
)

// MetricsServer は、メトリクスサーバーを生成します。
func MetricsServer(mtcCfg *config.MetricsConfig) *http.Server {
	return &http.Server{
		// pprof / metrics は DefaultServeMux に登録される前提で補助サーバーとして公開します。
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
	metricsSrv := MetricsServer(mtcCfg)

	start := func() {
		go func() {
			logListenError(logger, metricsSrv.ListenAndServe())
		}()
	}
	end := func(ctx context.Context) {
		// 停止失敗でアプリ本体を巻き込まないよう、panic せずログに留める（start 側の goroutine と同方針）。
		if err := metricsSrv.Shutdown(ctx); err != nil {
			logger.Named("metrics.Shutdown").Error("metrics server shutdown error", logging.Error("shutdown", err))
		}
	}
	return start, end
}

// logListenError は、補助サーバーの ListenAndServe 戻り値を判定し、正常停止(ErrServerClosed)以外の
// エラーのみをログ記録します。bind 失敗等でアプリ本体を巻き込まないよう、panic せずログに留めます。
func logListenError(logger logging.Logger, err error) {
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Named("metrics.ListenAndServe").Error("metrics server error", logging.Error("listen", err))
	}
}
