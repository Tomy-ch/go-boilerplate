//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package outbox は、トランザクショナル outbox の emit / relay / GC / replay ユースケースを提供します。
package outbox

import (
	"context"
	"encoding/json"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	"go-boilerplate/pkg/httpheader"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// ErrInvalidOrdering は、順序キーと位置の指定が対になっていないことを示すエラーです。
var ErrInvalidOrdering = xerrors.Wrap(apperror.ErrInvalidArgument, "invalid outbox ordering")

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
	// Headers は publish 時に外部エンドポイントへ伝搬するヘッダ（traceparent 等）です。nil 可。
	// 既知の機微ヘッダ名（Authorization / Cookie 等）は送出前に落としますが、それは defense-in-depth で
	// あって主契約ではありません。機微情報は入れないこと。
	Headers map[string]string
	// Channel は配送レーンです。既定値を持たないため、必ず指定します。
	Channel outboxbndry.Channel
	// OrderingKey は順序保証の単位（ストリーム）です。順序を持たない配送では空にします。
	OrderingKey string
	// OrderingSequence は OrderingKey 内の位置（1 起算）です。OrderingKey が空なら 0 にします。
	OrderingSequence int64
}

// EmitUsecase は、ドメイン変更と同一 tx で outbox へ 1 件記録するユースケースです。
type EmitUsecase interface {
	// Emit は、業務 tx 内で呼ばれ、outbox へちょうど 1 件記録し、採番された message_id を返します。
	// 呼び出し側がドメイン変更と同じ tx.Manager.Do の中で呼ぶことで lost event を排除します。
	// 保存される Headers には、呼び出し時点の trace context が自動で追加されます（traceparent、および存在すれば tracestate）。
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

func (u *emitUsecase) Emit(ctx context.Context, in EmitInput) (uuid.UUID, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := validateEmit(in); err != nil {
		return uuid.UUID{}, err
	}

	headers := make(map[string]string, len(in.Headers)+1)
	for k, v := range in.Headers {
		// 呼び出し側契約（EmitInput.Headers の doc）に加えた defense-in-depth として、
		// egress の起点であるここで、誤って混入した既知の機微ヘッダを保守的に落とします。
		if httpheader.IsSensitive(k) {
			continue
		}
		headers[k] = v
	}
	// emit span の trace context を traceparent として載せる（消費側が同一 trace に繋がる）。
	observability.InjectTraceContextToCarrier(ctx, headers)

	var headerBytes []byte
	if len(headers) > 0 {
		b, err := json.Marshal(headers)
		if err != nil {
			return uuid.UUID{}, xerrors.Join(apperror.ErrInternal, xerrors.Wrap(err, "failed to encode outbox headers"))
		}
		headerBytes = b
	}

	return u.store.Insert(ctx, outboxbndry.EmitParams{
		AggregateType:    in.AggregateType,
		AggregateID:      in.AggregateID,
		EventType:        in.EventType,
		Payload:          in.Payload,
		Headers:          headerBytes,
		Channel:          in.Channel,
		OrderingKey:      in.OrderingKey,
		OrderingSequence: in.OrderingSequence,
	})
}

// validateEmit は、行を作る前に配送チャネルと順序指定を検査します。
// 未知のチャネルの行はどの relay も claim せず、対になっていない順序指定は DB の制約で弾かれるため、
// いずれも呼び出し元へ即座に返します。
func validateEmit(in EmitInput) error {
	if _, err := outboxbndry.ParseChannel(in.Channel.String()); err != nil {
		return err
	}

	switch {
	case in.OrderingKey == "" && in.OrderingSequence != 0:
		return xerrors.Wrap(ErrInvalidOrdering, "ordering sequence requires an ordering key")
	case in.OrderingKey != "" && in.OrderingSequence <= 0:
		return xerrors.Wrap(ErrInvalidOrdering, "ordering key requires a positive ordering sequence")
	default:
		return nil
	}
}
