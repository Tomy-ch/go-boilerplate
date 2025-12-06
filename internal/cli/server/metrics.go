package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // G108: 非本番環境でのプロファイリング用にインポート
	"strconv"
	"time"

	"boilerplate-go/internal/config"
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
		Addr:              mtcCfg.Host() + ":" + strconv.Itoa(mtcCfg.Port()),
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// NewMetricsServer は、メトリクスサーバーの開始および終了関数を生成します。
func NewMetricsServer(mtcCfg *config.MetricsConfig) (func(), func(ctx context.Context)) {
	metricsSrv := MetricsServer(mtcCfg)

	start := func() {
		go func() {
			if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				panic(fmt.Errorf("metrics server error: %w", err))
			}
		}()
	}
	end := func(ctx context.Context) {
		if err := metricsSrv.Shutdown(ctx); err != nil {
			panic(fmt.Errorf("metrics server shutdown error: %w", err))
		}
	}
	return start, end
}
