//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import (
	"context"

	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// AccessRevoker は、feature が subject の destination への権利を取り下げたときに呼ぶ失効の seam です。
// この service の中で起きた失効は即時に伝わり（ADR-0074）、IdP 側の失効はここを通りません。
type AccessRevoker interface {
	// Revoke は、subject へ destination に対して発行済みの ticket をすべて無効にしたうえで、
	// 開いている接続を閉じるよう各 instance へ伝えます。該当する ticket や接続が無くてもエラーになりません。
	// 無効化が先で、通知に失敗しても無効化は成立しています（順序の根拠は README / ADR-0074）。
	Revoke(ctx context.Context, subject string, destination rt.StreamID) error
}

type accessRevoker struct {
	tickets  rt.StreamTicketStore
	notifier rt.RevocationNotifier
	tracer   observability.LayerTracer
}

// NewAccessRevoker は、AccessRevoker を生成します。
func NewAccessRevoker(
	tickets rt.StreamTicketStore,
	notifier rt.RevocationNotifier,
	tf observability.TracerFactory,
) AccessRevoker {
	return &accessRevoker{tickets: tickets, notifier: notifier, tracer: tf.Usecase()}
}

func (r *accessRevoker) Revoke(ctx context.Context, subject string, destination rt.StreamID) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	if err := r.tickets.Invalidate(ctx, subject, destination); err != nil {
		return err
	}

	return r.notifier.NotifyRevoked(ctx, subject, destination)
}
