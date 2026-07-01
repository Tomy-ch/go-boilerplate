package module

import (
	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	outboxengine "go-boilerplate/internal/controller/outbox"
	relayhook "go-boilerplate/internal/di/outboxrelay/hook"
	"go-boilerplate/internal/observability"
	outboxuc "go-boilerplate/internal/usecase/outbox"
)

// OutboxRelayModule は、outbox relay engine とそのライフサイクルフックを提供するfx.Moduleです。
// relay 専用プロセス（cmd outbox-relay）でのみ使用します。
// outboxPublisherModule は非標準の httpclient profile（MaxAttempts=1 等）を value group へ寄与するため、
// relay 以外のプロセスへ漏れないよう共有 InfrastructureModule ではなくここに閉じ込めます。
func OutboxRelayModule() fx.Option {
	return fx.Module("outbox-relay",
		outboxPublisherModule(),
		fx.Provide(
			// 具象 OutboxMetrics を usecase 境界の Metrics interface として供給する。
			fx.Annotate(
				observability.NewOutboxMetrics,
				fx.As(new(outboxuc.Metrics)),
			),
			outboxuc.NewRelay,
			provideRelaySettings,
			outboxengine.NewEngine,
		),
		fx.Invoke(relayhook.RegisterRelayHooks),
	)
}

// provideRelaySettings は、OutboxConfig から relay engine の設定を生成します。
// BatchSize が 0 以下だとスピンループを招くため DefaultBatchSize へ clamp します。
func provideRelaySettings(cfg *config.OutboxConfig) outboxengine.Settings {
	batchSize := outboxuc.DefaultBatchSize
	if cfg.BatchSize() > 0 {
		//nolint:gosec // BatchSize は設定由来の小さな正整数であり int32 に収まる
		batchSize = int32(cfg.BatchSize())
	}
	return outboxengine.Settings{
		BatchSize:    batchSize,
		PollInterval: cfg.PollInterval(),
		ErrorBackoff: cfg.ErrorBackoff(),
	}
}
