package module

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	outboxengine "go-boilerplate/internal/controller/outbox"
	publisherbd "go-boilerplate/internal/usecase/boundary/publisher"
	outboxuc "go-boilerplate/internal/usecase/outbox"
)

func TestOutboxRelayModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// relay engine は usecase（RelayUsecase）・clock.Sleeper・OutboxConfig 等に依存する。
	// poll ループの振る舞いは controller 層のテストに任せ、ここでは engine と
	// そのライフサイクルフックが依存と欠落なく結線されることを確認する。
	opts := append(commonDeps(), InfrastructureModule(), UsecaseModule(), OutboxRelayModule())
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

			validateGraph(t, append(relayDeps(), OutboxRelayModule(),
				fx.Populate(&engine, &settings, &relay))...)
		})

		t.Run("relay 専用の outbox publisher も同梱して配線する", func(t *testing.T) {
			t.Parallel()

			// publisher は共有 InfrastructureModule には含まれず、本モジュールが持ち込む。
			var publisher publisherbd.Publisher

			validateGraph(t, append(relayDeps(), OutboxRelayModule(), fx.Populate(&publisher))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では relay engine が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var engine *outboxengine.Engine

			opts := append(relayDeps(), fx.Populate(&engine), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
