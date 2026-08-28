package module

import (
	"time"

	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	outboxengine "go-boilerplate/internal/controller/outbox"
	relayhook "go-boilerplate/internal/di/outboxrelay/hook"
	"go-boilerplate/internal/infrastructure/publisher"
	"go-boilerplate/internal/observability"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	outboxuc "go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"
)

const (
	// defaultRelayPollInterval は、PollInterval が 0 以下のときに寄せる既定待機時間です（envspec の POLL_INTERVAL 既定と一致）。
	defaultRelayPollInterval = 1 * time.Second
	// defaultRelayErrorBackoff は、ErrorBackoff が 0 以下のときに寄せる既定待機時間です（envspec の ERROR_BACKOFF 既定と一致）。
	defaultRelayErrorBackoff = 5 * time.Second
)

// OutboxRelayModule は、1 つの配送チャネルを担う outbox relay engine とそのライフサイクルフックを
// 提供するfx.Moduleです。relay 専用プロセス（cmd outbox-relay）でのみ使用します。
// チャネル隔離と publisher profile の閉じ込めは internal/di/README.md の Optional を参照。
func OutboxRelayModule(channel outboxbndry.Channel) fx.Option {
	return fx.Module("outbox-relay",
		outboxPublisherModule(),
		fx.Supply(channel),
		fx.Invoke(publisher.VerifyChannel),
		fx.Provide(
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

// provideRelaySettings は、OutboxConfig から relay engine の設定を生成します
// （clamp の理由は internal/controller/outbox/README.md の Public API を参照）。
// BatchSize が engine の int32 に収まらないときは、既定値へ丸めて誤設定を隠すのではなく起動を失敗させます。
func provideRelaySettings(cfg *config.OutboxConfig) (outboxengine.Settings, error) {
	batchSize := outboxuc.DefaultBatchSize
	if cfg.BatchSize() > 0 {
		v, err := safecast.IntToInt32(cfg.BatchSize())
		if err != nil {
			return outboxengine.Settings{}, xerrors.Wrap(err, "outbox relay batch size is out of range")
		}
		batchSize = v
	}

	pollInterval := defaultRelayPollInterval
	if cfg.PollInterval() > 0 {
		pollInterval = cfg.PollInterval()
	}

	errorBackoff := defaultRelayErrorBackoff
	if cfg.ErrorBackoff() > 0 {
		errorBackoff = cfg.ErrorBackoff()
	}

	return outboxengine.Settings{
		BatchSize:    batchSize,
		PollInterval: pollInterval,
		ErrorBackoff: errorBackoff,
	}, nil
}
