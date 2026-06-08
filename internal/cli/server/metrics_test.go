package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func TestLogListenError(t *testing.T) {
	t.Parallel()

	t.Run("正常系_nilは何もログしない", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		// Named/Error への呼び出しが起きれば gomock が想定外呼び出しとして失敗させる。
		logger := mock_logging.NewMockLogger(ctrl)

		logListenError(logger, nil)
	})

	t.Run("正常系_ErrServerClosedは正常停止としてログしない", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		logger := mock_logging.NewMockLogger(ctrl)

		logListenError(logger, http.ErrServerClosed)
	})

	t.Run("異常系_それ以外のエラーはErrorで1回ログする", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		logger := mock_logging.NewMockLogger(ctrl)
		logger.EXPECT().Named("metrics.ListenAndServe").Return(logger)
		logger.EXPECT().Error("metrics server error", gomock.Any())

		logListenError(logger, errors.New("bind failed"))
	})
}

func TestNewMetricsServer(t *testing.T) {
	t.Parallel()

	t.Run("正常系_起動から停止までのライフサイクルがpanicせず完了する", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		mtcCfg := config.NewMetricsConfig(cfg)
		// 固定ポートの bind 衝突を避けるため、エフェメラルポート(:0)で待ち受けさせる。
		mtcCfg.SetMetricsPort(t, 0)

		start, end := NewMetricsServer(mtcCfg, logging.NewTestLogger(t))
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

		_, end := NewMetricsServer(mtcCfg, logging.NewTestLogger(t))

		// 起動していないサーバーへの Shutdown はエラーにならず、panic しないこと
		// （goroutine 内 panic 廃止の方針に沿う）。
		assert.NotPanics(t, func() { end(context.Background()) })
	})
}
