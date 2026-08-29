package module

import (
	"time"

	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	outboxengine "go-boilerplate/internal/controller/outbox"
	relayhook "go-boilerplate/internal/di/outboxrelay/hook"
	"go-boilerplate/internal/infrastructure/publisher"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	publisherbndry "go-boilerplate/internal/usecase/boundary/publisher"
	"go-boilerplate/internal/usecase/boundary/tx"
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

// relayUsecaseIn は、relay usecase の依存を DI から集約します。
// 依存を 1 つずつ引数に並べると構築子の引数が増え続けるため、集約は DI 側に置きます。
type relayUsecaseIn struct {
	fx.In

	Txm       tx.Manager
	Store     outboxbndry.Store
	Publisher publisherbndry.Publisher
	Metrics   outboxuc.Metrics
	Clock     clock.Clock
	Logging   logging.Logger
	Tracer    observability.TracerFactory
	Channel   outboxbndry.Channel
}

// OutboxRelayModule は、1 つの配送チャネルを担う outbox relay engine とそのライフサイクルフックを
// 提供するfx.Moduleです。relay 専用プロセス（cmd outbox-relay）でのみ使用します。
// チャネル隔離と publisher profile の閉じ込めは internal/di/README.md の Optional を参照。
func OutboxRelayModule(channel outboxbndry.Channel) fx.Option {
	return fx.Module("outbox-relay",
		publisherModuleFor(channel),
		fx.Supply(channel),
		fx.Provide(
			fx.Annotate(
				observability.NewOutboxMetrics,
				fx.As(new(outboxuc.Metrics)),
			),
			provideRelayUsecase,
			provideRelaySettings,
			outboxengine.NewEngine,
		),
		fx.Invoke(relayhook.RegisterRelayHooks),
	)
}

// publisherModuleFor は、channel を配送できる publisher module を返します。HTTP channel は判別子
// （OUTBOX_PUBLISHER）で HTTP / SQS 互換ブローカーを選ぶ従来の module、realtime channel は EventLog へ
// append してから wakeup を publish する module です。対応する case が無い channel は、担当者の居ない
// relay を黙って起動させないため構築エラーにします（fail-closed）。
func publisherModuleFor(channel outboxbndry.Channel) fx.Option {
	switch channel {
	case outboxbndry.ChannelHTTP:
		return fx.Options(outboxPublisherModule(), fx.Invoke(publisher.VerifyChannel))
	case outboxbndry.ChannelRealtime:
		return realtimePublisherModule()
	default:
		return fx.Error(xerrors.Wrap(publisher.ErrChannelUnsupported, "no publisher module serves delivery channel "+channel.String()))
	}
}

// provideRelayUsecase は、集約した依存から channel を担う RelayUsecase を生成します。
func provideRelayUsecase(in relayUsecaseIn) outboxuc.RelayUsecase {
	return outboxuc.NewRelay(outboxuc.RelayDeps{
		Txm:       in.Txm,
		Store:     in.Store,
		Publisher: in.Publisher,
		Metrics:   in.Metrics,
		Clock:     in.Clock,
		Logging:   in.Logging,
		Tracer:    in.Tracer,
	}, in.Channel)
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
