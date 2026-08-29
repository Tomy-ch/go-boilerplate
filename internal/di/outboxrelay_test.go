package di

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	config "go-boilerplate/internal/config"
	outboxengine "go-boilerplate/internal/controller/outbox"
	"go-boilerplate/internal/di/module"
	"go-boilerplate/internal/infrastructure/publisher"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	outboxuc "go-boilerplate/internal/usecase/outbox"
)

func TestNewOutboxRelayCore(t *testing.T) {
	t.Parallel()

	require.NoError(t, fx.ValidateApp(NewOutboxRelayCore(outboxbndry.ChannelHTTP), fx.WithLogger(NewFxEventLogger)))
}

func TestNewOutboxRelayApp(t *testing.T) {
	config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

	t.Run("正常系", func(t *testing.T) {
		t.Run("有効な ENDPOINT_OUTBOX なら relay app が起動可能", func(t *testing.T) {
			// 送信先の検証対象は HTTP 経路なので、判別子を明示して env の既定に依存させない。
			t.Setenv("OUTBOX_PUBLISHER", "http")
			t.Setenv("ENDPOINT_OUTBOX", "http://localhost:9999")

			app := NewOutboxRelayApp(30*time.Second, outboxbndry.ChannelHTTP)

			// fx.New はコンストラクタ（NewEndpoint 等）を実行しエラーを app.Err() に格納する。
			require.NoError(t, app.Err())
		})

		t.Run("realtime チャネルは EventLog へ append する publisher で構築できる", func(t *testing.T) {
			t.Setenv("REALTIME_TOPIC", "arn:aws:sns:us-east-1:000000000000:realtime-fanout-test")
			t.Setenv("ENDPOINT_REALTIME_PUBSUB", "http://localhost:4100")

			app := NewOutboxRelayApp(30*time.Second, outboxbndry.ChannelRealtime)

			require.NoError(t, app.Err())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("担当する publisher module が無いチャネルは起動時に弾かれる", func(t *testing.T) {
			t.Setenv("OUTBOX_PUBLISHER", "http")
			t.Setenv("ENDPOINT_OUTBOX", "http://localhost:9999")

			// fx.ValidateApp は invoke の本体を実行しないため、ガードの検証は実際の構築で行う。
			app := NewOutboxRelayApp(30*time.Second, outboxbndry.Channel("unknown"))

			require.ErrorIs(t, app.Err(), publisher.ErrChannelUnsupported)
		})

		t.Run("realtime チャネルは REALTIME_TOPIC が空だと起動時に弾かれる", func(t *testing.T) {
			t.Setenv("REALTIME_TOPIC", "")

			app := NewOutboxRelayApp(30*time.Second, outboxbndry.ChannelRealtime)

			require.ErrorIs(t, app.Err(), module.ErrRealtimeTopicNotConfigured)
		})

		t.Run("ENDPOINT_OUTBOX が空なら起動時に弾かれる", func(t *testing.T) {
			t.Setenv("OUTBOX_PUBLISHER", "http")
			t.Setenv("ENDPOINT_OUTBOX", "")

			app := NewOutboxRelayApp(30*time.Second, outboxbndry.ChannelHTTP)

			require.Error(t, app.Err())
		})
	})
}

//nolint:paralleltest // EnsureRepoRootAndEnv が t.Setenv/t.Chdir を使用するため並列化不可
func TestRunOutboxReplay(t *testing.T) {
	config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

	t.Run("正常系", func(t *testing.T) {
		t.Run("dead行が無くても0件・エラーなしで起動から停止まで完了する", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			count, err := RunOutboxReplay(ctx, nil)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, count, int64(0))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("キャンセル済みコンテキストではapp.Start失敗を0件で返す", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			count, err := RunOutboxReplay(ctx, nil)
			require.Error(t, err)
			assert.Zero(t, count)
		})

		t.Run("fxグラフの構築に失敗するとpanicせず0件と構築エラーを返す", func(t *testing.T) {
			t.Setenv("APP_SHUTDOWN_TIMEOUT", "not-a-duration")

			// nil 参照の退行が起きても panic をこのテスト内で捕捉し、同一パッケージの後続テストを巻き込まない。
			var (
				count int64
				err   error
			)
			require.NotPanics(t, func() {
				count, err = RunOutboxReplay(context.Background(), nil)
			})

			require.ErrorIs(t, err, config.ErrFailedToParseConfig)
			assert.Zero(t, count)
		})
	})
}

func Test_outboxRelayCommonOptions(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("replay ワンショットが必要とする依存だけで結線が成立する", func(t *testing.T) {
			t.Parallel()

			var (
				replay outboxuc.ReplayUsecase
				appCfg *config.ApplicationConfig
			)

			opts := append(outboxRelayCommonOptions(),
				fx.Populate(&replay, &appCfg), fx.WithLogger(NewFxEventLogger))

			require.NoError(t, fx.ValidateApp(opts...))
		})

		t.Run("relay engine は共通モジュールに含まれずOutboxRelayModuleが要る", func(t *testing.T) {
			t.Parallel()

			var engine *outboxengine.Engine

			withoutRelay := append(outboxRelayCommonOptions(), fx.Populate(&engine), fx.NopLogger)
			require.Error(t, fx.ValidateApp(withoutRelay...))

			withRelay := append(outboxRelayCommonOptions(),
				module.OutboxRelayModule(outboxbndry.ChannelHTTP), fx.Populate(&engine), fx.NopLogger)
			require.NoError(t, fx.ValidateApp(withRelay...))
		})
	})
}
