package module

import (
	"time"

	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	outboxengine "go-boilerplate/internal/controller/outbox"
	relayhook "go-boilerplate/internal/di/outboxrelay/hook"
	"go-boilerplate/internal/observability"
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

// OutboxRelayModule は、outbox relay engine とそのライフサイクルフックを提供するfx.Moduleです。
// relay 専用プロセス（cmd outbox-relay）でのみ使用します。
// outboxPublisherModule は非標準の httpclient profile（MaxAttempts=1 等）を value group へ寄与するため、
// relay 以外のプロセスへ漏れないよう共有 InfrastructureModule ではなくここに閉じ込めます。
func OutboxRelayModule() fx.Option {
	return fx.Module("outbox-relay",
		outboxPublisherModule(),
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

// provideRelaySettings は、OutboxConfig から relay engine の設定を生成します。
// BatchSize / PollInterval / ErrorBackoff は 0 以下だとホットループ（スピン / Sleep 即 return）を
// 招くため、それぞれ既定値へ clamp します。
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
