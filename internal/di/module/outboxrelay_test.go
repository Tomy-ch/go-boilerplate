package module

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/mock/gomock"

	outboxengine "go-boilerplate/internal/controller/outbox"
	"go-boilerplate/internal/infrastructure/publisher"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	mock_outbox "go-boilerplate/internal/usecase/boundary/outbox/mock"
	publisherbd "go-boilerplate/internal/usecase/boundary/publisher"
	mock_publisher "go-boilerplate/internal/usecase/boundary/publisher/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	outboxuc "go-boilerplate/internal/usecase/outbox"
)

func TestOutboxRelayModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// relay engine は usecase（RelayUsecase）・clock.Sleeper・OutboxConfig 等に依存する。
	// poll ループの振る舞いは controller 層のテストに任せ、ここでは engine と
	// そのライフサイクルフックが依存と欠落なく結線されることを確認する。
	opts := append(commonDeps(), InfrastructureModule(), UsecaseModule(), OutboxRelayModule(outboxbndry.ChannelHTTP))
	validateGraph(t, opts...)
}

func TestOutboxRelayModule(t *testing.T) {
	t.Parallel()

	relayDeps := func() []fx.Option {
		return append(commonDeps(), InfrastructureModule(), UsecaseModule())
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("relay engine と設定と RelayUsecase を提供する", func(t *testing.T) {
			t.Parallel()

			var (
				engine   *outboxengine.Engine
				settings outboxengine.Settings
				relay    outboxuc.RelayUsecase
			)

			validateGraph(t, append(relayDeps(), OutboxRelayModule(outboxbndry.ChannelHTTP),
				fx.Populate(&engine, &settings, &relay))...)
		})

		t.Run("relay 専用の outbox publisher も同梱して配線する", func(t *testing.T) {
			t.Parallel()

			// publisher は共有 InfrastructureModule には含まれず、本モジュールが持ち込む。
			var publisher publisherbd.Publisher

			validateGraph(
				t,
				append(relayDeps(), OutboxRelayModule(outboxbndry.ChannelHTTP), fx.Populate(&publisher))...)
		})

		t.Run("realtime channel では EventLog へ append する publisher を配線する", func(t *testing.T) {
			t.Parallel()

			var publisher publisherbd.Publisher

			validateGraph(
				t,
				append(relayDeps(), OutboxRelayModule(outboxbndry.ChannelRealtime), fx.Populate(&publisher))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("担当する publisher module が無い channel は構築エラー（fail-closed）", func(t *testing.T) {
			t.Parallel()

			err := fx.ValidateApp(
				append(relayDeps(), OutboxRelayModule(outboxbndry.Channel("unknown")), fx.NopLogger)...)
			require.ErrorIs(t, err, publisher.ErrChannelUnsupported)
		})

		t.Run("未配線では relay engine が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var engine *outboxengine.Engine

			opts := append(relayDeps(), fx.Populate(&engine), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}

func Test_provideRelayUsecase(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("供給されたチャネルを担う relay usecase を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)

			// 集約した依存が取り違えなく渡ることを、チャネルが claim 系の呼び出しへ届くことで確かめる。
			store.EXPECT().
				OldestPendingCreatedAt(gomock.Any(), outboxbndry.ChannelRealtime).
				Return(time.Time{}, false, nil)

			uc := provideRelayUsecase(relayUsecaseIn{
				Txm:       mock_tx.NewMockManager(ctrl),
				Store:     store,
				Publisher: mock_publisher.NewMockPublisher(ctrl),
				Metrics:   observability.NewNoopOutboxMetrics(t),
				Clock:     clocktestkit.NewStepClock(time.Time{}, 0),
				Logging:   logging.NewTestLogger(t),
				Tracer:    observability.NewNoopTracerFactory(t),
				Channel:   outboxbndry.ChannelRealtime,
			})

			require.NotNil(t, uc)
			require.NoError(t, uc.RecordLag(context.Background()))
		})
	})
}

func Test_publisherModuleFor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("http は OUTBOX_PUBLISHER で選ぶ publisher module を返す", func(t *testing.T) {
			t.Parallel()

			var p publisherbd.Publisher

			validateGraph(
				t,
				append(commonDeps(), InfrastructureModule(), UsecaseModule(), publisherModuleFor(outboxbndry.ChannelHTTP), fx.Populate(&p))...)
		})

		t.Run("realtime は EventLog へ append する publisher module を返す", func(t *testing.T) {
			t.Parallel()

			var p publisherbd.Publisher

			validateGraph(t, append(commonDeps(), publisherModuleFor(outboxbndry.ChannelRealtime), fx.Populate(&p))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知の channel は ErrChannelUnsupported の構築エラーになる", func(t *testing.T) {
			t.Parallel()

			err := fx.ValidateApp(publisherModuleFor(outboxbndry.Channel("unknown")), fx.NopLogger)
			require.ErrorIs(t, err, publisher.ErrChannelUnsupported)
		})
	})
}
