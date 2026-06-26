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
// relay 専用プロセス（cmd outbox-relay）でのみ使用します。RelayUsecase と OutboxMetrics は
// relay でのみ必要なため、共有の UsecaseModule ではなくここで提供します。
func OutboxRelayModule() fx.Option {
	return fx.Module("outbox-relay",
		fx.Provide(
			observability.NewOutboxMetrics,
			outboxuc.NewRelay,
			provideRelaySettings,
			outboxengine.NewEngine,
		),
		fx.Invoke(relayhook.RegisterRelayHooks),
	)
}

// provideRelaySettings は、OutboxConfig から relay engine の設定を生成します。
func provideRelaySettings(cfg *config.OutboxConfig) outboxengine.Settings {
	return outboxengine.Settings{
		//nolint:gosec // BatchSize は設定由来の小さな正整数であり int32 に収まる
		BatchSize:    int32(cfg.BatchSize()),
		PollInterval: cfg.PollInterval(),
		ErrorBackoff: cfg.ErrorBackoff(),
	}
}
