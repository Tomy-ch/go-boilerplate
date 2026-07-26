//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package outbox

import (
	"context"
	"encoding/json"
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	"go-boilerplate/internal/usecase/boundary/publisher"
	"go-boilerplate/internal/usecase/boundary/tx"
)

const (
	// DefaultBatchSize は、1 回の relay で claim する pending 行数の既定値です。
	DefaultBatchSize int32 = 100
	// DefaultMaxAttempts は、dead 化までの publish 試行回数の既定値です。
	DefaultMaxAttempts int32 = 10

	relayLoggerName = "outbox-relay"
)

// Metrics は、relay engine が記録する outbox 固有の o11y シンクです。
type Metrics interface {
	// SetLagSeconds は、最古 pending 行の経過秒数（SLI=outbox lag）を記録します。
	SetLagSeconds(ctx context.Context, seconds int64)
	// IncDead は、dead 化したメッセージ数を計上します。
	IncDead(ctx context.Context)
}

// RelayResult は、1 回の RelayBatch の結果です。
type RelayResult struct {
	// Claimed は、claim した pending 行数です（0 なら pending 無し）。
	Claimed int
	// Published は、claim 行のうち publish に成功した行数です。
	Published int
}

// RelayUsecase は、pending 行を claim して publish するユースケースです。relay engine が周期的に呼びます。
type RelayUsecase interface {
	// RelayBatch は、最大 batchSize 件の pending 行を 1 tx で claim → publish → mark し、結果を返します。
	// 個々の publish 失敗は tx を巻き戻さず（failed/dead をマークして）次 poll で再送します。
	// DB アクセス自体の失敗のみ tx を巻き戻すエラーとして返します。
	RelayBatch(ctx context.Context, batchSize int32) (RelayResult, error)
	// RecordLag は、最古 pending 行の経過時間（outbox lag）を SLI メトリクスへ記録します。
	// pending 行が無ければ 0 を記録します。
	RecordLag(ctx context.Context) error
}

type relayUsecase struct {
	txm         tx.Manager
	store       outboxbndry.Store
	publisher   publisher.Publisher
	metrics     Metrics
	clock       clock.Clock
	logging     logging.Logger
	tracer      observability.LayerTracer
	maxAttempts int32
}

// NewRelay は、RelayUsecase を生成します。
func NewRelay(
	txm tx.Manager,
	store outboxbndry.Store,
	pub publisher.Publisher,
	metrics Metrics,
	clk clock.Clock,
	log logging.Logger,
	tf observability.TracerFactory,
) RelayUsecase {
	return &relayUsecase{
		txm:         txm,
		store:       store,
		publisher:   pub,
		metrics:     metrics,
		clock:       clk,
		logging:     log,
		tracer:      tf.Usecase(),
		maxAttempts: DefaultMaxAttempts,
	}
}

func (u *relayUsecase) RecordLag(ctx context.Context) error {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	createdAt, ok, err := u.store.OldestPendingCreatedAt(ctx)
	if err != nil {
		return err
	}

	var lag time.Duration
	if ok {
		if lag = u.clock.Now().Sub(createdAt); lag < 0 {
			lag = 0
		}
	}
	u.metrics.SetLagSeconds(ctx, int64(lag.Seconds()))
	return nil
}

func (u *relayUsecase) RelayBatch(ctx context.Context, batchSize int32) (RelayResult, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	// claim〜mark を同一 tx 内で完結させることで、多インスタンス間での二重 publish を防ぐ。
	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (RelayResult, error) {
		msgs, err := u.store.ClaimPending(ctx, batchSize)
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

// deliver は、1 件を publish し、結果に応じて published / failed / dead をマークします。
// publish 成功なら published=true を返します。publish 失敗は tx を巻き戻さず（次 poll の再送に委ねる）
// published=false・error=nil を返し、DB マークの失敗のみエラーを返します。
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

	attempts, ferr := u.store.MarkFailed(ctx, m.ID, perr.Error())
	if ferr != nil {
		return false, ferr
	}
	if attempts >= u.maxAttempts {
		if derr := u.store.MarkDead(ctx, m.ID); derr != nil {
			return false, derr
		}
		u.metrics.IncDead(ctx)
		u.logging.Named(relayLoggerName).Warn(
			ctx,
			"outbox message marked dead after reaching max attempts",
			logging.String(logging.MessageIDKey, m.MessageID.String()),
			logging.String(logging.EventTypeKey, m.EventType),
			logging.Error(logging.JobErrorKey, perr),
		)
	}
	return false, nil
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
