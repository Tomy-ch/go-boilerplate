package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// metricsBlockPathSeq は、DefaultServeMux への二重登録 panic を避けるため一意な登録パスを払い出します。
var metricsBlockPathSeq atomic.Int64

func TestMetricsServer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Addrとハンドラと全タイムアウトが宣言通りに設定される", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			mtcCfg := config.NewMetricsConfig(cfg)

			srv := metricsServer(mtcCfg)
			require.NotNil(t, srv)
			// Addr が host:port 形式で組み立てられること。
			assert.Equal(t, mtcCfg.Host()+":"+strconv.Itoa(mtcCfg.Port()), srv.Addr)
			// 補助サーバーは DefaultServeMux（pprof）を公開する。
			assert.Equal(t, http.DefaultServeMux, srv.Handler)
			// タイムアウトが固定値で設定されること（4 値全て）。
			assert.Equal(t, readHeaderTimeout, srv.ReadHeaderTimeout)
			assert.Equal(t, readTimeout, srv.ReadTimeout)
			assert.Equal(t, writeTimeout, srv.WriteTimeout)
			assert.Equal(t, idleTimeout, srv.IdleTimeout)
		})
	})
}

func TestLogListenError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilは何もログしない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			// Named/Error への呼び出しが起きれば gomock が想定外呼び出しとして失敗させる。
			logger := mock_logging.NewMockLogger(ctrl)

			logListenError(logger, nil)
		})

		t.Run("ErrServerClosedは正常停止としてログしない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			logger := mock_logging.NewMockLogger(ctrl)

			logListenError(logger, http.ErrServerClosed)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("それ以外のエラーはErrorで1回ログする", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			logger := mock_logging.NewMockLogger(ctrl)
			logger.EXPECT().Named("metrics.ListenAndServe").Return(logger)
			logger.EXPECT().Error(gomock.Any(), "metrics server error", gomock.Any())

			logListenError(logger, xerrors.New("bind failed"))
		})
	})
}

func TestNewMetricsServer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("起動から停止までのライフサイクルがpanicせず完了する", func(t *testing.T) {
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

		t.Run("起動していないサーバーへの停止でもpanicしない", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			mtcCfg := config.NewMetricsConfig(cfg)

			_, end := NewMetricsServer(mtcCfg, logging.NewTestLogger(t))

			// 起動していないサーバーへの Shutdown はエラーにならず、panic しないこと
			// （goroutine 内 panic 廃止の方針に沿う）。
			assert.NotPanics(t, func() { end(context.Background()) })
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("進行中リクエストがあるとShutdownの失敗をエラーとしてログする", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			entered := make(chan struct{}, 1)
			release := make(chan struct{})

			// 補助サーバーは DefaultServeMux を公開するため、専用パスに進行中ハンドラを登録する。
			blockPath := fmt.Sprintf("/__metrics_shutdown_block__%d", metricsBlockPathSeq.Add(1))
			http.HandleFunc(blockPath, blockingHandler(entered, release))

			mtcCfg := config.NewMetricsConfig(config.MockConfigForTest(t))
			port := reservePort(t, mtcCfg.Host())
			mtcCfg.SetMetricsPort(t, port)

			start, end := NewMetricsServer(mtcCfg, logger)
			start()

			driveInFlight(t, "http://"+mtcCfg.Host()+":"+strconv.Itoa(port)+blockPath, entered, release)

			// 非アイドル接続が残る状態へ、期限切れの ctx で Shutdown を当てる。
			end(canceledContext())
			close(release)

			assert.Positive(t, observed.FilterMessage("metrics server shutdown error").Len())
		})
	})
}

// blockingHandler は、最初の呼び出しで entered を通知し release まで待機するハンドラを返します。
func blockingHandler(entered chan<- struct{}, release <-chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		w.WriteHeader(http.StatusOK)
	}
}

// reservePort は、host と同じ family で bind 可能なポートを予約し確定ポートを返します。
func reservePort(t *testing.T, host string) int {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", host+":0")
	require.NoError(t, err)
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	require.NoError(t, ln.Close())
	return port
}

// driveInFlight は、url へ接続してハンドラを進行中（非アイドル接続）にし、進入を待ちます。
func driveInFlight(t *testing.T, url string, entered <-chan struct{}, release chan struct{}) {
	t.Helper()
	go sendUntilConnected(url)
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("進行中ハンドラへ到達しなかった")
	}
}

// sendUntilConnected は、bind 完了まで接続を試み、接続できたら応答を読み切ります。
func sendUntilConnected(url string) {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return
		}
		res, err := http.DefaultClient.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// canceledContext は、生成直後にキャンセル済みの context を返します。
func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
