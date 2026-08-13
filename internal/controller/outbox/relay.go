// Package outbox は、outbox を周期 poll して未 publish メッセージを送る relay engine を提供します。
// engine 自体は loop と待機制御だけを担い、claim → publish → mark の業務は usecase に委譲します。
package outbox

import (
	"context"
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	outboxuc "go-boilerplate/internal/usecase/outbox"
)

const relayLoggerName = "outbox-relay"

// Settings は、relay engine のチューニング値です。
type Settings struct {
	// BatchSize は、1 回の poll で claim する pending エントリ数です。
	BatchSize int32
	// PollInterval は、pending を捌き切った（空振り or 部分消化）後に次 poll まで待機する時間です。
	PollInterval time.Duration
	// ErrorBackoff は、RelayBatch がエラーを返した後に待機する時間です。
	ErrorBackoff time.Duration
}

// Engine は、pending のエントリを周期 poll して publish する常駐 engine です。
type Engine struct {
	uc      outboxuc.RelayUsecase
	sleeper clock.Sleeper
	logging logging.Logger
	tracer  observability.LayerTracer
	set     Settings
}

// NewEngine は、relay engine を生成します。
func NewEngine(
	uc outboxuc.RelayUsecase,
	sleeper clock.Sleeper,
	log logging.Logger,
	tf observability.TracerFactory,
	set Settings,
) *Engine {
	return &Engine{
		uc:      uc,
		sleeper: sleeper,
		logging: log,
		tracer:  tf.Controller(),
		set:     set,
	}
}

// Run は、poll ループ本体です。ctx 完了まで常駐し、完了時に nil を返します。
// 待機判断（連続消化 / PollInterval / ErrorBackoff）の表と根拠は README の Loop semantics を参照。
// 待機は clock.Sleeper 経由で行い、ctx 完了で即座に抜けます。
func (e *Engine) Run(ctx context.Context) error {
	ctx, endSpan := e.tracer.Start(ctx)
	defer endSpan()

	log := e.logging.Named(relayLoggerName)
	for {
		if ctx.Err() != nil {
			return nil
		}

		res, err := e.uc.RelayBatch(ctx, e.set.BatchSize)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Error(ctx, "outbox relay batch failed", logging.Error(logging.JobErrorKey, err))
			if e.waitDone(ctx, e.set.ErrorBackoff) {
				return nil
			}
			continue
		}

		e.observeLag(ctx, log)

		if res.Claimed >= int(e.set.BatchSize) && res.Published > 0 {
			continue
		}
		if e.waitDone(ctx, e.set.PollInterval) {
			return nil
		}
	}
}

// waitDone は、d 待機します。ctx 完了で待機が中断された場合に true を返します。
func (e *Engine) waitDone(ctx context.Context, d time.Duration) bool {
	return e.sleeper.Sleep(ctx, d) != nil
}

// observeLag は、outbox lag(SLI) をベストエフォートで記録します。
// ctx 完了時は記録をスキップし、記録失敗はループを止めずログのみ行います。
func (e *Engine) observeLag(ctx context.Context, log logging.Logger) {
	if ctx.Err() != nil {
		return
	}
	if err := e.uc.RecordLag(ctx); err != nil {
		log.Error(ctx, "failed to record outbox lag", logging.Error(logging.JobErrorKey, err))
	}
}
