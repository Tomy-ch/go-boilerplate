//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import (
	"context"
	"time"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// LeaseCleanupOwnershipTTL は、回収の引き受けが有効な期間です。回収は数回の API 呼び出しで終わるため
// これで足り、引き受けたまま倒れた掃除役が居ても、この期間を過ぎれば次の掃除役が引き受け直せます。
// 死んだと見なすまでの余裕（LeaseCleanupMargin）とは別の量なので、値を共有しません。
const LeaseCleanupOwnershipTTL = 10 * time.Minute

// SweepResult は、1 回の掃除の内訳です。
type SweepResult struct {
	// Detected は、期限切れが安全余裕を過ぎていた lease の数です。
	Detected int
	// Claimed は、回収を引き受けられた数です。
	Claimed int
	// Reclaimed は、受信先を片付けて lease まで閉じられた数です。
	Reclaimed int
	// Skipped は、他の掃除役が先に引き受けていた、または instance が復帰していて閉じなかった数です。
	Skipped int
	// Failed は、引き受け・回収・閉じるのいずれかが失敗した数です。
	Failed int
}

// OrphanSweeper は、死んだ instance が残した受信先を回収する seam です。回収は引き受けを条件付きで
// 取った主体だけが行い、同時に走った掃除役のうち 1 つだけが 1 つの instance を回収します（ADR-0073）。
type OrphanSweeper interface {
	// Sweep は、回収できる lease をすべて処理して内訳を返します。1 件の失敗では止まらず、
	// 最後にまとめて返します。
	Sweep(ctx context.Context) (SweepResult, error)
}

type orphanSweeper struct {
	leases    rt.InstanceLeaseStore
	reclaimer rt.OrphanReclaimer
	owner     string
	clock     clock.Clock
	tracer    observability.LayerTracer
}

// NewOrphanSweeper は、owner の名前で回収を引き受ける OrphanSweeper を生成します。
func NewOrphanSweeper(
	leases rt.InstanceLeaseStore,
	reclaimer rt.OrphanReclaimer,
	owner string,
	clk clock.Clock,
	tf observability.TracerFactory,
) OrphanSweeper {
	return &orphanSweeper{leases: leases, reclaimer: reclaimer, owner: owner, clock: clk, tracer: tf.Usecase()}
}

// Sweep は、期限切れが安全余裕を過ぎた lease を引き受けてから、受信先を片付け、lease を閉じます。
// 順序が固定なのは、lease が受信先を辿る唯一の索引だからです。先に lease を閉じると、片付けに失敗した
// 受信先を誰も見つけられなくなります（docs/design/realtime-delivery.md §2.5 / ADR-0073）。
func (s *orphanSweeper) Sweep(ctx context.Context) (SweepResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	now := s.clock.Now()
	// 期限切れからさらに安全余裕を過ぎたものだけを対象にする。引き受けと閉じるときの条件式にも
	// 同じ時刻を渡し、列挙と条件判定が別の基準を使わないようにする。
	cutoff := now.Add(-LeaseCleanupMargin)

	leases, err := s.leases.ListExpired(ctx, cutoff)
	if err != nil {
		return SweepResult{}, err
	}

	result := SweepResult{Detected: len(leases)}

	var errs []error
	for _, lease := range leases {
		if err := s.sweepOne(ctx, lease.InstanceID, now, cutoff, &result); err != nil {
			errs = append(errs, err)
		}
	}

	return result, xerrors.Join(errs...)
}

// sweepOne は、1 つの instance を引き受けから lease の削除まで処理し、内訳を result へ加算します。
func (s *orphanSweeper) sweepOne(
	ctx context.Context, id rt.InstanceID, now, cutoff time.Time, result *SweepResult,
) error {
	claimed, err := s.leases.AcquireCleanup(ctx, rt.CleanupClaim{
		InstanceID:    id,
		Owner:         s.owner,
		ExpiredBefore: cutoff,
		Now:           now,
		OwnerUntil:    now.Add(LeaseCleanupOwnershipTTL),
	})
	if err != nil {
		result.Failed++

		return err
	}

	if !claimed {
		result.Skipped++

		return nil
	}

	result.Claimed++

	if err := s.reclaimer.Reclaim(ctx, id); err != nil {
		// 受信先が残ったので lease は閉じない（順序の理由は Sweep の doc comment）。次の掃除が引き受け直す。
		result.Failed++

		return err
	}

	released, err := s.leases.ReleaseCleanup(ctx, rt.CleanupRelease{
		InstanceID:    id,
		Owner:         s.owner,
		ExpiredBefore: cutoff,
	})
	if err != nil {
		result.Failed++

		return err
	}

	if !released {
		// 引き受けている間に instance が復帰したか、引き受けが他へ移った。受信先は既に片付いており、
		// 復帰した instance は次の受信で自分の受信先を張り直す。
		result.Skipped++

		return nil
	}

	result.Reclaimed++

	return nil
}
