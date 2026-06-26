//go:generate mockgen -source=$GOFILE -destination=mock/mock_emit.gen.go -package=mock_$GOPACKAGE

// Package outbox は、トランザクショナル outbox の emit / relay / GC / replay ユースケースを提供します。
package outbox

import (
	"context"
	"encoding/json"
	"maps"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// EmitInput は、ドメインイベントを outbox へ emit する入力です。
type EmitInput struct {
	// AggregateType は集約種別です（観測・調査用）。
	AggregateType string
	// AggregateID は集約 ID です（観測・調査用）。
	AggregateID string
	// EventType はイベント種別 + version です。
	EventType string
	// Payload は呼び出し側が snapshot + version で marshal 済みのイベント本文 JSON です。
	Payload []byte
	// Headers は publish 時に伝搬するヘッダ（traceparent 等）です。nil 可。
	Headers map[string]string
}

// EmitUsecase は、ドメイン変更と同一 tx で outbox 行を INSERT するユースケースです。
type EmitUsecase interface {
	// Emit は、業務 tx 内で呼ばれ、outbox 行を 1 行 INSERT し、採番された message_id を返します。
	// 呼び出し側がドメイン変更と同じ tx.Manager.Do の中で呼ぶことで lost event を排除します。
	Emit(ctx context.Context, in EmitInput) (uuid.UUID, error)
}

type emitUsecase struct {
	store  outboxbndry.Store
	tracer observability.LayerTracer
}

// NewEmit は、EmitUsecase を生成します。
func NewEmit(store outboxbndry.Store, tf observability.TracerFactory) EmitUsecase {
	return &emitUsecase{store: store, tracer: tf.Usecase()}
}

// Emit は、業務 tx 内で outbox 行を 1 行 INSERT し、採番された message_id を返します。
// 現在の ctx の traceparent を headers へ capture し、後続の relay→受信側を起点 trace に繋ぎます。
func (u *emitUsecase) Emit(ctx context.Context, in EmitInput) (uuid.UUID, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	headers := make(map[string]string, len(in.Headers)+1)
	maps.Copy(headers, in.Headers)
	// emit span の trace context を traceparent として載せる（消費側が同一 trace に繋がる）。
	// TraceContext 限定で inject し、インバウンド由来の baggage が外部へ転送されるのを防ぐ。
	observability.InjectTraceContextToCarrier(ctx, headers)

	var headerBytes []byte
	if len(headers) > 0 {
		b, err := json.Marshal(headers)
		if err != nil {
			return uuid.UUID{}, xerrors.Wrap(apperror.ErrInternal, "failed to encode outbox headers: "+err.Error())
		}
		headerBytes = b
	}

	return u.store.Insert(ctx, outboxbndry.EmitParams{
		AggregateType: in.AggregateType,
		AggregateID:   in.AggregateID,
		EventType:     in.EventType,
		Payload:       in.Payload,
		Headers:       headerBytes,
	})
}
