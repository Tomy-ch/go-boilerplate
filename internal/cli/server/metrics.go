package server

import (
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
// 非本番環境でのみ有効です。
func MetricsServer(cfg *config.Config) *http.Server {
	appCfg := config.NewApplicationConfig(cfg)

	if !appCfg.IsAppProductionMode() {
		mtcCfg := config.NewMetricsConfig(cfg)
		return &http.Server{
			Addr:              mtcCfg.MetricsHost() + ":" + strconv.Itoa(mtcCfg.MetricsPort()),
			Handler:           http.DefaultServeMux,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		}
	}
	return nil
}
