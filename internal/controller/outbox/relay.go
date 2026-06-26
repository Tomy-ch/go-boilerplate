// Package outbox は、outbox テーブルを周期 poll して未 publish メッセージを送る relay engine を提供します。
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
	// BatchSize は、1 回の poll で claim する pending 行数です。
	BatchSize int32
	// PollInterval は、pending を捌き切った（空振り or 部分消化）後に次 poll まで待機する時間です。
	PollInterval time.Duration
	// ErrorBackoff は、RelayBatch がエラーを返した後に待機する時間です。
	ErrorBackoff time.Duration
}

// Engine は、pending 行を周期 poll して publish する常駐 engine です。
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
//   - 満杯（claim 件数 == BatchSize）かつ publish 進捗あり（published > 0）の間は、まだ pending が
//     残る可能性が高いため待機せず連続して捌きます。
//   - 空振り・部分消化・「満杯だが全件 publish 失敗」なら PollInterval、エラー時は ErrorBackoff 待機します。
//     全件 publish 失敗の満杯バッチを待機ゼロで再 claim すると、下流停止時にホットループして即時 dead 化
//     するため、進捗が無い満杯バッチは必ず待機へ落とします。
//   - 待機は clock.Sleeper 経由で行い、ctx 完了で即座に抜けます（決定的テストのため注入）。
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
			log.Error("outbox relay batch failed", logging.Error(logging.JobErrorKey, err))
			if e.waitDone(ctx, e.set.ErrorBackoff) {
				return nil
			}
			continue
		}

		// lag 記録はバッチ成功時のみ行う。エラー時は同一原因（DB 障害等）で二重にエラーログが出るのを避ける。
		e.observeLag(ctx, log)

		if res.Claimed >= int(e.set.BatchSize) && res.Published > 0 {
			// 満杯かつ進捗あり。まだ pending が残る可能性が高いため待機せず次 poll を即実行する。
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
		log.Error("failed to record outbox lag", logging.Error(logging.JobErrorKey, err))
	}
}
