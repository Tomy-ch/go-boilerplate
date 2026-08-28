//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package outbox

import (
	"context"
	"encoding/json"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	"go-boilerplate/internal/usecase/boundary/publisher"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/backoff"
	"go-boilerplate/pkg/retry"
	"go-boilerplate/pkg/xerrors"
)

const (
	// DefaultBatchSize は、1 回の relay で claim する pending エントリ数の既定値です。
	DefaultBatchSize int32 = 100

	// retryInitialInterval は、1 回目の publish 失敗後に空ける待機時間です。
	retryInitialInterval = 1 * time.Second
	// retryMaxInterval は、再試行間隔の上限です（ADR-0058）。
	retryMaxInterval = 60 * time.Second
	// retryMultiplier は、失敗を重ねるごとの間隔の倍率です。
	retryMultiplier = 2.0

	relayLoggerName = "outbox-relay"
)

// retryBackoff は、retryDelay が使う指数バックオフのパラメータです。
var retryBackoff = backoff.Exponential{
	Initial:    retryInitialInterval,
	Max:        retryMaxInterval,
	Multiplier: retryMultiplier,
}

// Metrics は、relay engine が記録する outbox 固有の o11y シンクです。
// channel はチャネル名で、いずれの記録もチャネル単位で識別されます。
type Metrics interface {
	// SetLagSeconds は、最古 pending エントリの経過秒数（SLI=outbox lag）を記録します。
	SetLagSeconds(ctx context.Context, channel string, seconds int64)
	// IncDead は、dead 化したメッセージ数を計上します。
	IncDead(ctx context.Context, channel string)
	// SetBlockedStreams は、先頭エントリが dead で進行が止まっているストリーム数を記録します。
	SetBlockedStreams(ctx context.Context, channel string, count int64)
}

// RelayResult は、1 回の RelayBatch の結果です。
type RelayResult struct {
	// Claimed は、claim した pending エントリ数です（0 なら pending 無し）。
	Claimed int
	// Published は、claim したエントリのうち publish に成功した件数です。
	Published int
}

// RelayUsecase は、pending のエントリを claim して publish するユースケースです。relay engine が周期的に呼びます。
type RelayUsecase interface {
	// RelayBatch は、最大 batchSize 件の pending エントリを 1 tx で claim → publish → mark し、結果を返します。
	// 個々の publish 失敗は tx を巻き戻さず（failed/dead をマークして）次 poll で再送します。
	// DB アクセス自体の失敗のみ tx を巻き戻すエラーとして返します。
	RelayBatch(ctx context.Context, batchSize int32) (RelayResult, error)
	// RecordLag は、最古 pending エントリの経過時間（outbox lag）を SLI メトリクスへ記録します。
	// pending のエントリが無ければ 0 を記録します。
	RecordLag(ctx context.Context) error
	// RecordBlockedStreams は、先頭エントリが dead で進行が止まっているストリーム数をメトリクスへ記録します。
	RecordBlockedStreams(ctx context.Context) error
}

type relayUsecase struct {
	txm       tx.Manager
	store     outboxbndry.Store
	publisher publisher.Publisher
	metrics   Metrics
	clock     clock.Clock
	logging   logging.Logger
	tracer    observability.LayerTracer
	channel   outboxbndry.Channel
}

// NewRelay は、1 つの配送チャネルを担う RelayUsecase を生成します
// （チャネル隔離は docs/design/outbox.md の Design invariants を参照）。
func NewRelay(
	txm tx.Manager,
	store outboxbndry.Store,
	pub publisher.Publisher,
	metrics Metrics,
	clk clock.Clock,
	log logging.Logger,
	tf observability.TracerFactory,
	channel outboxbndry.Channel,
) RelayUsecase {
	return &relayUsecase{
		txm:       txm,
		store:     store,
		publisher: pub,
		metrics:   metrics,
		clock:     clk,
		logging:   log,
		tracer:    tf.Usecase(),
		channel:   channel,
	}
}

func (u *relayUsecase) RecordLag(ctx context.Context) error {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	createdAt, ok, err := u.store.OldestPendingCreatedAt(ctx, u.channel)
	if err != nil {
		return err
	}

	var lag time.Duration
	if ok {
		if lag = u.clock.Now().Sub(createdAt); lag < 0 {
			lag = 0
		}
	}
	u.metrics.SetLagSeconds(ctx, u.channel.String(), int64(lag.Seconds()))
	return nil
}

func (u *relayUsecase) RecordBlockedStreams(ctx context.Context) error {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	count, err := u.store.CountBlockedStreams(ctx, u.channel)
	if err != nil {
		return err
	}
	u.metrics.SetBlockedStreams(ctx, u.channel.String(), count)
	return nil
}

func (u *relayUsecase) RelayBatch(ctx context.Context, batchSize int32) (RelayResult, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (RelayResult, error) {
		msgs, err := u.store.ClaimPending(ctx, u.channel, batchSize)
		if err != nil {
			return RelayResult{}, err
		}
		res := RelayResult{Claimed: len(msgs)}
		for i := range msgs {
			published, derr := u.deliver(ctx, msgs[i])
			if derr != nil {
				return RelayResult{}, derr
			}
			if published {
				res.Published++
			}
		}
		return res, nil
	})
}

// deliver は、1 件を publish し、結果に応じて published / failed / dead をマークします（tx 方針は RelayBatch を参照）。
// publish 成功なら published=true を、DB マーク自体の失敗時のみ error を返します。
func (u *relayUsecase) deliver(ctx context.Context, m outboxbndry.PendingMessage) (bool, error) {
	perr := u.publisher.Publish(ctx, publisher.Message{
		MessageID: m.MessageID,
		EventType: m.EventType,
		Payload:   m.Payload,
		Headers:   decodeHeaders(m.Headers),
	})
	if perr == nil {
		return true, u.store.MarkPublished(ctx, m.ID)
	}

	// 失敗理由は dead でも残すため、dead 化の前に必ず記録する。
	permanent := isPermanent(perr)
	nextAttemptAt := u.clock.Now()
	if !permanent {
		nextAttemptAt = nextAttemptAt.Add(retryDelay(m.Attempts))
	}
	if ferr := u.store.MarkFailed(ctx, m.ID, perr.Error(), nextAttemptAt); ferr != nil {
		return false, ferr
	}
	if !permanent {
		return false, nil
	}

	if derr := u.store.MarkDead(ctx, m.ID); derr != nil {
		return false, derr
	}
	u.metrics.IncDead(ctx, u.channel.String())
	u.logging.Named(relayLoggerName).Warn(
		ctx,
		"outbox message marked dead by permanent publish failure",
		logging.String(logging.MessageIDKey, m.MessageID.String()),
		logging.String(logging.EventTypeKey, m.EventType),
		logging.Error(logging.JobErrorKey, perr),
	)
	return false, nil
}

// isPermanent は、publish の失敗が再試行で結果の変わらない永久失敗かを返します。
// どちらの分類も持たないエラーは一時失敗として扱います（既定の理由は ADR-0058）。
func isPermanent(err error) bool {
	return xerrors.Is(err, apperror.ErrPermanent)
}

// retryDelay は、attempts 回失敗した行を次に claim してよくなるまでの間隔を、指数バックオフへ
// full jitter を重ねて返します（同時に失敗した行が同じ poll へ殺到しないため）。
func retryDelay(attempts int32) time.Duration {
	return retry.Full(retryBackoff.Duration(int(attempts)))
}

// decodeHeaders は、保存済みヘッダ JSON を map へ復元します。壊れていても publish は継続するため nil を返します。
func decodeHeaders(b []byte) map[string]string {
	if len(b) == 0 {
		return nil
	}
	var h map[string]string
	if err := json.Unmarshal(b, &h); err != nil {
		return nil
	}
	return h
}
