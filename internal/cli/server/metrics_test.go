package server

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsServer(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	mtcCfg := config.NewMetricsConfig(cfg)

	srv := MetricsServer(mtcCfg)
	require.NotNil(t, srv)
	// Addr が host:port 形式で組み立てられること。
	assert.Equal(t, mtcCfg.Host()+":"+strconv.Itoa(mtcCfg.Port()), srv.Addr)
	// 補助サーバーは DefaultServeMux（pprof）を公開する。
	assert.Equal(t, http.DefaultServeMux, srv.Handler)
	// タイムアウトが固定値で設定されること。
	assert.Equal(t, readHeaderTimeout, srv.ReadHeaderTimeout)
	assert.Equal(t, writeTimeout, srv.WriteTimeout)
}

func TestNewMetricsServer(t *testing.T) {
	t.Parallel()

	t.Run("正常系_起動から停止までのライフサイクルがpanicせず完了する", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		mtcCfg := config.NewMetricsConfig(cfg)
		// 固定ポートの bind 衝突を避けるため、エフェメラルポート(:0)で待ち受けさせる。
		mtcCfg.SetMetricsPort(t, 0)

		start, end := NewMetricsServer(mtcCfg)
		require.NotNil(t, start)
		require.NotNil(t, end)

		assert.NotPanics(t, func() {
			start()
			end(context.Background())
		})
	})

	t.Run("正常系_起動していないサーバーへの停止でもpanicしない", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		mtcCfg := config.NewMetricsConfig(cfg)

		_, end := NewMetricsServer(mtcCfg)

		// 起動していないサーバーへの Shutdown はエラーにならず、panic しないこと
		// （goroutine 内 panic 廃止の方針に沿う）。
		assert.NotPanics(t, func() { end(context.Background()) })
	})
}
