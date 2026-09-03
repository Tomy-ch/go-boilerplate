package di

import (
	"context"
	"net/http"
	"testing"
	"time"

	config "go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	gomock "go.uber.org/mock/gomock"
)

func TestNewApplicationCore(t *testing.T) {
	// BootsWithMockedDB サブテストが EnsureRepoRootAndEnv 経由で t.Setenv/t.Chdir を使うため、
	// 親では t.Parallel() を付けない（cwd を変える serial サブテストと並列サブテストの競合を避ける）。
	t.Run("正常系", func(t *testing.T) {
		t.Run("依存グラフの結線が検証を通る", func(t *testing.T) {
			t.Parallel()

			// 本番と同じ fx.WithLogger(NewFxEventLogger) を渡し、ロガー構成子の依存解決も併せて検証する。
			opts := append(applicationCoreOptions(), fx.WithLogger(NewFxEventLogger))
			require.NoError(t, fx.ValidateApp(opts...))
		})

		//nolint:paralleltest // EnsureRepoRootAndEnv が t.Setenv/t.Chdir を使用するため並列化不可
		t.Run("モック DB で全コンストラクタとライフサイクルが起動・停止する", func(t *testing.T) {
			// 実 DB とポート衝突を避けつつ、全コンストラクタの実行とライフサイクル(OnStart/OnStop)を検証する。
			// DB ドライバを IF レベルでモックに差し替えて実 Ping を回避し、サーバポートは 0（エフェメラル）にする。
			config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

			ctrl := gomock.NewController(t)
			mockDB := mock_driver.NewMockDatabaseDriver(ctrl)

			var closeCalled bool
			mockDB.EXPECT().Close().DoAndReturn(func() error { closeCalled = true; return nil }).AnyTimes()
			mockDB.EXPECT().Stats().Return(&pgxpool.Stat{}).AnyTimes()
			mockDB.EXPECT().Ping(gomock.Any()).Return(nil).AnyTimes()

			app := NewApplicationCore(
				fx.Replace(fx.Annotate(mockDB, fx.As(new(driver.DatabaseDriver)))),
				realtimeSubstrate(t, ctrl),
				fx.Decorate(func(s *config.ServerConfig) *config.ServerConfig {
					s.SetServerPort(t, 0)
					return s
				}),
				fx.NopLogger,
			)

			start, stop := NewApplicationServer(app)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			require.NoError(t, start(ctx))

			stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer stopCancel()
			require.NoError(t, stop(stopCtx))

			// Close が呼ばれた＝モックがグラフに組み込まれ、実 DB(NewDB の Ping)が使われていないことの証左。
			assert.True(t, closeCalled, "db close hook がモックドライバの Close を呼ぶこと")
		})
	})
}

// realtimeSubstrate は、Realtime Delivery の外部依存を IF レベルで差し替えます。
// serve lifecycle（起動時の probe → 受信先の用意 → 常駐処理 → drain → 片付け）は本物のまま、
// DynamoDB と SNS / SQS へ出ていく往復だけが止まります。
//
// 差し替えた型の構築子とその上流（DynamoDB クライアント・fan-out の組み立て）は走りません。
// fx の decorator は元の構築子を呼ばないためで、REALTIME_TOPIC 未設定のような
// 「app.Start より前に落ちる」分岐はここでは露見せず、app-di-startup-check が受け持ちます。
func realtimeSubstrate(t *testing.T, ctrl *gomock.Controller) fx.Option {
	t.Helper()

	eventLog := mock_realtime.NewMockEventLogStore(ctrl)
	// 起動時の probe が listen より前に走ることを、呼ばれた事実で固定します。
	eventLog.EXPECT().Latest(gomock.Any(), gomock.Any()).Return(rt.DeliveryEvent{}, false, nil).MinTimes(1)

	tickets := mock_realtime.NewMockStreamTicketStore(ctrl)

	leases := mock_realtime.NewMockInstanceLeaseStore(ctrl)
	leases.EXPECT().Heartbeat(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	// 片付けで lease を閉じるところまでが serve lifecycle の終わりです。
	leases.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	sub := mock_realtime.NewMockInstanceSubscription(ctrl)
	// 起動と片付けの参加者が登録から落ちても graph は解決するので、回数で固定します。
	sub.EXPECT().Provision(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	sub.EXPECT().Teardown(gomock.Any()).Return(nil).Times(1)
	// 空を即座に返すと consumer の loop が待たずに回り続けるので、停止まで待たせる。
	sub.EXPECT().Receive(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ int) ([]rt.Notification, error) {
			<-ctx.Done()

			return nil, ctx.Err()
		}).AnyTimes()

	notifier := mock_realtime.NewMockRevocationNotifier(ctrl)

	return fx.Replace(
		fx.Annotate(eventLog, fx.As(new(rt.EventLogStore))),
		fx.Annotate(tickets, fx.As(new(rt.StreamTicketStore))),
		fx.Annotate(leases, fx.As(new(rt.InstanceLeaseStore))),
		fx.Annotate(sub, fx.As(new(rt.InstanceSubscription))),
		fx.Annotate(notifier, fx.As(new(rt.RevocationNotifier))),
	)
}

func Test_serveRealtimeOptions(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Realtime の runtime を 1 つだけ返す", func(t *testing.T) {
			t.Parallel()

			// sample の realtime adapter を消すとマーカー行ごと消えて空になります。
			// ここが 1 であることが、いま runtime が serve に載っている唯一の根拠です。
			assert.Len(t, serveRealtimeOptions(), 1)
		})
	})
}

func Test_serveBaseOptions(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("アプリケーションコアは base と Realtime の連結である", func(t *testing.T) {
			t.Parallel()

			// base だけで fx を組めるのは sample 撤去後だけ（撤去前は feature が realtime adapter を要る）なので、
			// 撤去後の graph が起動することは sample-removal-check が受け持ちます。ここで固定するのは
			// 「2 つの差が serveRealtimeOptions だけである」という分割の形です。
			assert.NotEmpty(t, serveBaseOptions())
			assert.Len(t, applicationCoreOptions(), len(serveBaseOptions())+len(serveRealtimeOptions()))
		})
	})
}

func TestNewApplicationServer(t *testing.T) {
	t.Parallel()

	// ライフサイクルフックの発火有無で、ラッパーが実際に app.Start / app.Stop を駆動したことを検証する。
	// 空アプリだと start/stop が常に nil を返し、駆動していなくても通ってしまうため。
	newHookedApp := func(started, stopped *bool) *fx.App {
		return fx.New(
			fx.Invoke(func(lc fx.Lifecycle) {
				lc.Append(fx.Hook{
					OnStart: func(context.Context) error { *started = true; return nil },
					OnStop:  func(context.Context) error { *stopped = true; return nil },
				})
			}),
			fx.NopLogger,
		)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("返した start/stop が app のライフサイクルを起動・停止する", func(t *testing.T) {
			t.Parallel()

			var started, stopped bool
			start, stop := NewApplicationServer(newHookedApp(&started, &stopped))
			require.NotNil(t, start)
			require.NotNil(t, stop)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			require.NoError(t, start(ctx))
			assert.True(t, started, "start ラッパーが app.Start を呼びライフサイクルが起動すること")

			stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer stopCancel()
			require.NoError(t, stop(stopCtx))
			assert.True(t, stopped, "stop ラッパーが app.Stop を呼びライフサイクルが停止すること")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストの start は app.Start のエラーをそのまま返す", func(t *testing.T) {
			t.Parallel()

			var started, stopped bool
			start, _ := NewApplicationServer(newHookedApp(&started, &stopped))

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			require.ErrorIs(t, start(ctx), context.Canceled)
			assert.False(t, started, "起動前に中断されフックへ到達しないこと")
		})
	})
}

func Test_applicationCoreOptions(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("server プロファイル固有のHTTPスタックを結線する", func(t *testing.T) {
			t.Parallel()

			var (
				e   *echo.Echo
				srv *http.Server
			)

			opts := append(applicationCoreOptions(), fx.Populate(&e, &srv), fx.NopLogger)
			require.NoError(t, fx.ValidateApp(opts...))
		})
	})
}
