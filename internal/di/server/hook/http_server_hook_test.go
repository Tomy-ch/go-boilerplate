package hook

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/server"
	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	"go-boilerplate/internal/di/server/extension"
	mock_logging "go-boilerplate/internal/logging/mock"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRegisterHTTPServerHooks(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	var startFn func(context.Context) error
	var shutdownFn func(context.Context) error
	dummy := func(context.Context) error { return nil }

	mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
	mockReg.EXPECT().RegisterStart(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
		fn, ok := args[0].(func(context.Context) error)
		require.True(t, ok)
		startFn = fn
	}).Times(1)

	mockLogger := mock_logging.NewMockLogger(ctrl)
	mockReg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
		fn, ok := args[0].(func(context.Context) error)
		require.True(t, ok)
		shutdownFn = fn
	}).Times(1)

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	secCfg := config.NewSecurityConfig(cfg)
	srvCfg := config.NewServerConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)

	_, srv := newTestHTTPServer(t, srvCfg)

	RegisterHTTPServerHooks(srv, mockReg, mockLogger, appCfg, secCfg, srvCfg, osCfg, &extension.AppliedServerExtends{})
	assert.NotNil(t, startFn)
	assert.NotNil(t, shutdownFn)
}

func Test_newStartServerFunc(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("HTTPサーバーが起動し設定ポートで待ち受けること", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockLogger := mock_logging.NewMockLogger(ctrl)
			namedMock := mock_logging.NewMockLogger(ctrl)

			mockLogger.EXPECT().Named("server.Start").Return(namedMock).AnyTimes()
			namedMock.EXPECT().CallerSkip(serverCallerSkip).Return(namedMock).AnyTimes()
			namedMock.EXPECT().
				Info(gomock.Any(), "http started", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Times(1)
			namedMock.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			port := freePort(t)

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			secCfg := config.NewSecurityConfig(cfg)
			srvCfg := config.NewServerConfig(cfg)
			osCfg := config.NewOperatingSystemConfig(cfg)
			srvCfg.SetServerPort(t, port)

			e, srv := newTestHTTPServer(t, srvCfg)
			e.GET("/ping", func(c *echo.Context) error { return c.NoContent(http.StatusNoContent) })

			fn := newStartServerFunc(srv, mockLogger, appCfg, secCfg, srvCfg, osCfg)
			require.NoError(t, fn(context.Background()))

			t.Cleanup(func() {
				_ = srv.Shutdown(context.Background())
			})

			// TCP の接続確立はリッスンだけで成立するため、実際にリクエストを処理できることまで確かめる。
			url := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/ping"
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
			require.NoError(t, err)
			resp, err := (&http.Client{Timeout: time.Second}).Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ポートのリッスンに失敗した場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockLogger := mock_logging.NewMockLogger(ctrl)

			lc := &net.ListenConfig{}
			ln, err := lc.Listen(context.Background(), "tcp", ":0")
			require.NoError(t, err)
			tcpAddr, ok := ln.Addr().(*net.TCPAddr)
			require.True(t, ok)
			port := tcpAddr.Port
			t.Cleanup(func() {
				_ = ln.Close()
			})

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			secCfg := config.NewSecurityConfig(cfg)
			srvCfg := config.NewServerConfig(cfg)
			osCfg := config.NewOperatingSystemConfig(cfg)
			srvCfg.SetServerPort(t, port)

			_, srv := newTestHTTPServer(t, srvCfg)
			fn := newStartServerFunc(srv, mockLogger, appCfg, secCfg, srvCfg, osCfg)

			err = fn(context.Background())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to listen on port")
		})
	})
}

func Test_serveHTTP(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Shutdownによる停止ではErrorログを出さない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockLogger := mock_logging.NewMockLogger(ctrl)

			cfg := config.MockConfigForTest(t)
			srvCfg := config.NewServerConfig(cfg)
			_, srv := newTestHTTPServer(t, srvCfg)

			lc := &net.ListenConfig{}
			ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
			require.NoError(t, err)

			done := make(chan struct{})
			go func() {
				serveHTTP(context.Background(), srv, ln, mockLogger)
				close(done)
			}()

			require.NoError(t, srv.Shutdown(context.Background()))
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("serveHTTP が終了しなかった")
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("http.ErrServerClosed以外で終了するとErrorログを出す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockLogger := mock_logging.NewMockLogger(ctrl)
			namedMock := mock_logging.NewMockLogger(ctrl)

			mockLogger.EXPECT().Named("server.Start").Return(namedMock).Times(1)
			namedMock.EXPECT().
				Error(gomock.Any(), "failed to start http server", gomock.Any()).
				Times(1)

			cfg := config.MockConfigForTest(t)
			srvCfg := config.NewServerConfig(cfg)
			_, srv := newTestHTTPServer(t, srvCfg)

			lc := &net.ListenConfig{}
			ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
			require.NoError(t, err)
			// 閉じた Listener を渡し、正常停止以外の終了を再現する。
			require.NoError(t, ln.Close())

			serveHTTP(context.Background(), srv, ln, mockLogger)
		})
	})
}

func Test_newStopServerFunc(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("処理中接続が無ければ即座にShutdownしInfoログを出す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockLogger := mock_logging.NewMockLogger(ctrl)
			mockLogger.EXPECT().Named("server.Stop").Return(mockLogger).AnyTimes()
			mockLogger.EXPECT().CallerSkip(serverCallerSkip).Return(mockLogger).AnyTimes()
			mockLogger.EXPECT().Info(gomock.Any(), "http stopping", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

			cfg := config.MockConfigForTest(t)
			srvCfg := config.NewServerConfig(cfg)
			osCfg := config.NewOperatingSystemConfig(cfg)

			_, srv := newTestHTTPServer(t, srvCfg)
			fn := newStopServerFunc(srv, mockLogger, osCfg)

			require.NoError(t, fn(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("処理中接続が残りShutdownがタイムアウトするとErrorログを出しエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockLogger := mock_logging.NewMockLogger(ctrl)
			mockLogger.EXPECT().Named("server.Stop").Return(mockLogger).AnyTimes()
			mockLogger.EXPECT().CallerSkip(serverCallerSkip).Return(mockLogger).AnyTimes()
			mockLogger.EXPECT().Info(gomock.Any(), "http stopping", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
			mockLogger.EXPECT().Error(gomock.Any(), "failed to shutdown http server", gomock.Any()).Times(1)

			cfg := config.MockConfigForTest(t)
			srvCfg := config.NewServerConfig(cfg)
			osCfg := config.NewOperatingSystemConfig(cfg)

			e, srv := newTestHTTPServer(t, srvCfg)

			// ハンドラを処理中にして接続を active に保ち、Shutdown を idle 完了させない
			entered := make(chan struct{})
			release := make(chan struct{})
			e.GET("/block", func(c *echo.Context) error {
				close(entered)
				<-release
				return c.NoContent(http.StatusOK)
			})

			lc := &net.ListenConfig{}
			ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
			require.NoError(t, err)
			go func() { _ = srv.Serve(ln) }()
			t.Cleanup(func() { close(release) })

			go func() {
				req, rerr := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+ln.Addr().String()+"/block", nil)
				if rerr != nil {
					return
				}
				resp, gerr := http.DefaultClient.Do(req)
				if gerr == nil {
					_ = resp.Body.Close()
				}
			}()
			<-entered // リクエストが処理中＝接続が active になったことを保証

			// 既に期限の切れた context で Shutdown → 処理中接続が残り context error を返す
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			fn := newStopServerFunc(srv, mockLogger, osCfg)
			require.Error(t, fn(ctx))
		})
	})
}

func Test_lifecycleEventFields(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

// newTestHTTPServer は、テスト用の HTTP サーバーを Echo とともに構築します。
func newTestHTTPServer(t *testing.T, srvCfg *config.ServerConfig) (*echo.Echo, *http.Server) {
	t.Helper()
	e := server.NewAppServer()
	return e, server.NewHTTPServer(e, srvCfg)
}

// freePort は、OS が割り当てた空きポート番号を返します。
func freePort(t *testing.T) int {
	t.Helper()
	lc := &net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	require.True(t, ok)
	require.NoError(t, ln.Close())
	return tcpAddr.Port
}
