//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package outbox

import (
	"context"

	"go-boilerplate/internal/observability"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	"go-boilerplate/pkg/uuid"
)

// ReplayUsecase は、dead 化した outbox 行を pending へ戻し再 publish 対象に復帰させるユースケースです。
type ReplayUsecase interface {
	// ReplayDead は、dead 行を pending へ戻し、戻した件数を返します。
	// messageID が nil なら全 dead 行、指定時は当該 message_id のみを対象とします。
	ReplayDead(ctx context.Context, messageID *uuid.UUID) (int64, error)
}

type replayUsecase struct {
	store  outboxbndry.Store
	tracer observability.LayerTracer
}

// NewReplay は、ReplayUsecase を生成します。
func NewReplay(store outboxbndry.Store, tf observability.TracerFactory) ReplayUsecase {
	return &replayUsecase{store: store, tracer: tf.Usecase()}
}

func (u *replayUsecase) ReplayDead(ctx context.Context, messageID *uuid.UUID) (int64, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	return u.store.ReplayDead(ctx, messageID)
}
