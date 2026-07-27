package hook

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/server"
	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"

	"github.com/labstack/echo/v4"
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

	e := server.NewAppServer(srvCfg)

	RegisterHTTPServerHooks(e, mockReg, mockLogger, appCfg, secCfg, srvCfg, osCfg, &extension.AppliedServerExtends{})
	assert.NotNil(t, startFn)
	assert.NotNil(t, shutdownFn)
}

func Test_newStartServerFunc(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("HTTPサーバーが起動後にListenerが設定されること", func(t *testing.T) {
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

			lc := &net.ListenConfig{}
			ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
			require.NoError(t, err)
			tcpAddr, ok := ln.Addr().(*net.TCPAddr)
			require.True(t, ok)
			port := tcpAddr.Port
			require.NoError(t, ln.Close())

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			secCfg := config.NewSecurityConfig(cfg)
			srvCfg := config.NewServerConfig(cfg)
			osCfg := config.NewOperatingSystemConfig(cfg)
			srvCfg.SetServerPort(t, port)

			e := server.NewAppServer(srvCfg)

			fn := newStartServerFunc(e, mockLogger, appCfg, secCfg, srvCfg, osCfg)
			err = fn(context.Background())
			require.NoError(t, err)
			assert.NotNil(t, e.Listener)

			t.Cleanup(func() {
				_ = e.Shutdown(context.Background())
			})
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

			e := server.NewAppServer(srvCfg)
			fn := newStartServerFunc(e, mockLogger, appCfg, secCfg, srvCfg, osCfg)

			err = fn(context.Background())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to listen on port")
		})

		t.Run("Startがhttp.ErrServerClosed以外で失敗するとErrorログを出す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockLogger := mock_logging.NewMockLogger(ctrl)
			namedMock := mock_logging.NewMockLogger(ctrl)

			mockLogger.EXPECT().Named("server.Start").Return(namedMock).AnyTimes()
			namedMock.EXPECT().CallerSkip(serverCallerSkip).Return(namedMock).AnyTimes()
			namedMock.EXPECT().
				Info(gomock.Any(), "http started", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				AnyTimes()
			errLogged := make(chan struct{})
			namedMock.EXPECT().
				Error(gomock.Any(), "failed to start http server", gomock.Any()).
				Do(func(context.Context, string, ...*logging.Field) { close(errLogged) }).
				Times(1)

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			secCfg := config.NewSecurityConfig(cfg)
			srvCfg := config.NewServerConfig(cfg)
			osCfg := config.NewOperatingSystemConfig(cfg)
			srvCfg.SetServerPort(t, 0) // OS 割り当ての空きポート

			e := server.NewAppServer(srvCfg)
			fn := newStartServerFunc(e, mockLogger, appCfg, secCfg, srvCfg, osCfg)

			require.NoError(t, fn(context.Background()))
			// Shutdown ではなく Listener を直接 Close し、Serve を http.ErrServerClosed 以外で終了させる。
			require.NoError(t, e.Listener.Close())

			select {
			case <-errLogged:
			case <-time.After(2 * time.Second):
				t.Fatal("Start 失敗時の Error ログが呼ばれなかった")
			}
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

			e := server.NewAppServer(srvCfg)
			fn := newStopServerFunc(e, mockLogger, osCfg)

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

			e := server.NewAppServer(srvCfg)

			// ハンドラを処理中にして接続を active に保ち、Shutdown を idle 完了させない
			entered := make(chan struct{})
			release := make(chan struct{})
			e.GET("/block", func(c echo.Context) error {
				close(entered)
				<-release
				return c.NoContent(http.StatusOK)
			})

			lc := &net.ListenConfig{}
			ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
			require.NoError(t, err)
			e.Listener = ln
			go func() { _ = e.Start("") }()
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
			fn := newStopServerFunc(e, mockLogger, osCfg)
			require.Error(t, fn(ctx))
		})
	})
}

func Test_lifecycleEventFields(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
