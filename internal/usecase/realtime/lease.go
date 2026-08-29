//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import (
	"context"
	"time"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

// instance lease の固定値（ADR-0073）。deployment で変える理由が無いので config ではなく code に置きます。
const (
	// LeaseHeartbeatInterval は、serve instance が生存を記録し直す間隔です。
	LeaseHeartbeatInterval = 30 * time.Second
	// LeaseExpiry は、記録が途絶えた instance を死んだと見なすまでの期間です。
	LeaseExpiry = 2 * time.Minute
	// LeaseCleanupMargin は、期限切れからさらに待ってから resource を回収する安全余裕です（回収は orphan cleanup の job）。
	LeaseCleanupMargin = 5 * time.Minute
)

// LeaseKeeper は、serve instance の生存記録を書き、正常終了時に取り消す seam です。
// 記録は crash した instance の resource を後から回収するためだけにあり、lock でも leader election でもありません。
type LeaseKeeper interface {
	// Beat は、id の instance が今生きていることを記録します。期限は今から LeaseExpiry 後です。
	Beat(ctx context.Context, id rt.InstanceID) error
	// Release は、id の記録を取り消します。resource を自分で片付けた instance が呼び、無くてもエラーになりません。
	Release(ctx context.Context, id rt.InstanceID) error
}

type leaseKeeper struct {
	store  rt.InstanceLeaseStore
	clock  clock.Clock
	tracer observability.LayerTracer
}

// NewLeaseKeeper は、LeaseKeeper を生成します。
func NewLeaseKeeper(store rt.InstanceLeaseStore, clk clock.Clock, tf observability.TracerFactory) LeaseKeeper {
	return &leaseKeeper{store: store, clock: clk, tracer: tf.Usecase()}
}

func (k *leaseKeeper) Beat(ctx context.Context, id rt.InstanceID) error {
	ctx, endSpan := k.tracer.Start(ctx)
	defer endSpan()

	now := k.clock.Now()

	return k.store.Heartbeat(ctx, rt.InstanceLease{InstanceID: id, HeartbeatAt: now, ExpiresAt: now.Add(LeaseExpiry)})
}

func (k *leaseKeeper) Release(ctx context.Context, id rt.InstanceID) error {
	ctx, endSpan := k.tracer.Start(ctx)
	defer endSpan()

	return k.store.Delete(ctx, id)
}
