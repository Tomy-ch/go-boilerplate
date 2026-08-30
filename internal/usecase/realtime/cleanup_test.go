package realtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
)

// sweepNow は、掃除の基準時刻です。cutoff はここから LeaseCleanupMargin を引いた値になります。
var sweepNow = time.Date(2026, time.August, 30, 4, 0, 0, 0, time.UTC)

// sweepCutoff は、sweepNow を基準にしたときの回収対象の締切です。
var sweepCutoff = sweepNow.Add(-LeaseCleanupMargin)

func newOrphanSweeper(t *testing.T) (
	*orphanSweeper, *mock_realtime.MockInstanceLeaseStore, *mock_realtime.MockOrphanReclaimer, *mock_clock.MockClock,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	leases := mock_realtime.NewMockInstanceLeaseStore(ctrl)
	reclaimer := mock_realtime.NewMockOrphanReclaimer(ctrl)
	clk := mock_clock.NewMockClock(ctrl)

	s, ok := NewOrphanSweeper(leases, reclaimer, "cleaner-1", clk, observability.NewNoopTracerFactory(t)).(*orphanSweeper)
	require.True(t, ok)

	return s, leases, reclaimer, clk
}

// expectListExpired は、cutoff で列挙されて ids が返る、という約束を置きます。
func expectListExpired(
	t *testing.T, leases *mock_realtime.MockInstanceLeaseStore, clk *mock_clock.MockClock, ids ...rt.InstanceID,
) {
	t.Helper()

	clk.EXPECT().Now().Return(sweepNow)

	out := make([]rt.InstanceLease, 0, len(ids))
	for _, id := range ids {
		out = append(out, rt.InstanceLease{InstanceID: id})
	}
	leases.EXPECT().ListExpired(gomock.Any(), sweepCutoff).Return(out, nil)
}

func TestNewOrphanSweeper(t *testing.T) {
	t.Parallel()

	s, leases, reclaimer, clk := newOrphanSweeper(t)
	assert.NotNil(t, s)
	assert.NotNil(t, leases)
	assert.NotNil(t, reclaimer)
	assert.NotNil(t, clk)
}

// Test_orphanSweeper_Sweep は、列挙の基準時刻と、複数件にまたがる集計・エラーの束ね方を固定する。
// 1 件ごとの分岐は Test_orphanSweeper_sweepOne が持つ。
func Test_orphanSweeper_Sweep(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限切れから安全余裕を過ぎた時刻で列挙する", func(t *testing.T) {
			t.Parallel()

			s, leases, _, clk := newOrphanSweeper(t)
			expectListExpired(t, leases, clk)

			got, err := s.Sweep(t.Context())
			require.NoError(t, err)
			assert.Equal(t, SweepResult{}, got)
		})

		t.Run("複数件の内訳を合算する", func(t *testing.T) {
			t.Parallel()

			s, leases, reclaimer, clk := newOrphanSweeper(t)
			expectListExpired(t, leases, clk, "inst-1", "inst-2", "inst-3")

			leases.EXPECT().AcquireCleanup(gomock.Any(), gomock.Any()).Return(true, nil).Times(2)
			leases.EXPECT().AcquireCleanup(gomock.Any(), gomock.Any()).Return(false, nil)
			reclaimer.EXPECT().Reclaim(gomock.Any(), gomock.Any()).Return(nil).Times(2)
			leases.EXPECT().ReleaseCleanup(gomock.Any(), gomock.Any()).Return(true, nil).Times(2)

			got, err := s.Sweep(t.Context())
			require.NoError(t, err)
			assert.Equal(t, SweepResult{Detected: 3, Claimed: 2, Reclaimed: 2, Skipped: 1}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("1 件が失敗しても残りを処理し、失敗をまとめて返す", func(t *testing.T) {
			t.Parallel()

			s, leases, reclaimer, clk := newOrphanSweeper(t)
			expectListExpired(t, leases, clk, "inst-1", "inst-2")

			leases.EXPECT().AcquireCleanup(gomock.Any(), gomock.Any()).Return(true, nil).Times(2)
			reclaimer.EXPECT().Reclaim(gomock.Any(), rt.InstanceID("inst-1")).Return(apperror.ErrUnavailable)
			reclaimer.EXPECT().Reclaim(gomock.Any(), rt.InstanceID("inst-2")).Return(nil)
			leases.EXPECT().ReleaseCleanup(gomock.Any(), gomock.Any()).Return(true, nil)

			got, err := s.Sweep(t.Context())
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, SweepResult{Detected: 2, Claimed: 2, Reclaimed: 1, Failed: 1}, got)
		})

		t.Run("列挙が失敗したら内訳を返さない", func(t *testing.T) {
			t.Parallel()

			s, leases, _, clk := newOrphanSweeper(t)
			clk.EXPECT().Now().Return(sweepNow)
			leases.EXPECT().ListExpired(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrUnavailable)

			got, err := s.Sweep(t.Context())
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, SweepResult{}, got)
		})
	})
}

// Test_orphanSweeper_sweepOne は、1 つの instance を処理するときの分岐と、その内訳への加算を固定する。
func Test_orphanSweeper_sweepOne(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引き受けてから受信先を片付け、最後に lease を閉じる", func(t *testing.T) {
			t.Parallel()

			s, leases, reclaimer, _ := newOrphanSweeper(t)
			gomock.InOrder(
				leases.EXPECT().AcquireCleanup(gomock.Any(), rt.CleanupClaim{
					InstanceID:    "inst-1",
					Owner:         "cleaner-1",
					ExpiredBefore: sweepCutoff,
					Now:           sweepNow,
					OwnerUntil:    sweepNow.Add(LeaseCleanupOwnershipTTL),
				}).Return(true, nil),
				reclaimer.EXPECT().Reclaim(gomock.Any(), rt.InstanceID("inst-1")).Return(nil),
				leases.EXPECT().ReleaseCleanup(gomock.Any(), rt.CleanupRelease{
					InstanceID:    "inst-1",
					Owner:         "cleaner-1",
					ExpiredBefore: sweepCutoff,
				}).Return(true, nil),
			)

			var got SweepResult
			require.NoError(t, s.sweepOne(t.Context(), "inst-1", sweepNow, sweepCutoff, &got))
			assert.Equal(t, SweepResult{Claimed: 1, Reclaimed: 1}, got)
		})

		t.Run("他の掃除役が先に引き受けていれば受信先に触らない", func(t *testing.T) {
			t.Parallel()

			s, leases, _, _ := newOrphanSweeper(t)
			leases.EXPECT().AcquireCleanup(gomock.Any(), gomock.Any()).Return(false, nil)

			var got SweepResult
			require.NoError(t, s.sweepOne(t.Context(), "inst-1", sweepNow, sweepCutoff, &got))
			assert.Equal(t, SweepResult{Skipped: 1}, got)
		})

		t.Run("引き受けている間に instance が復帰していれば lease を閉じない", func(t *testing.T) {
			t.Parallel()

			s, leases, reclaimer, _ := newOrphanSweeper(t)
			leases.EXPECT().AcquireCleanup(gomock.Any(), gomock.Any()).Return(true, nil)
			reclaimer.EXPECT().Reclaim(gomock.Any(), gomock.Any()).Return(nil)
			leases.EXPECT().ReleaseCleanup(gomock.Any(), gomock.Any()).Return(false, nil)

			var got SweepResult
			require.NoError(t, s.sweepOne(t.Context(), "inst-1", sweepNow, sweepCutoff, &got))
			assert.Equal(t, SweepResult{Claimed: 1, Skipped: 1}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引き受けの失敗では受信先を片付けない", func(t *testing.T) {
			t.Parallel()

			s, leases, _, _ := newOrphanSweeper(t)
			leases.EXPECT().AcquireCleanup(gomock.Any(), gomock.Any()).Return(false, apperror.ErrUnavailable)

			var got SweepResult
			require.ErrorIs(t, s.sweepOne(t.Context(), "inst-1", sweepNow, sweepCutoff, &got), apperror.ErrUnavailable)
			assert.Equal(t, SweepResult{Failed: 1}, got)
		})

		t.Run("受信先の片付けに失敗したら lease を残す", func(t *testing.T) {
			t.Parallel()

			s, leases, reclaimer, _ := newOrphanSweeper(t)
			leases.EXPECT().AcquireCleanup(gomock.Any(), gomock.Any()).Return(true, nil)
			reclaimer.EXPECT().Reclaim(gomock.Any(), gomock.Any()).Return(apperror.ErrUnavailable)

			var got SweepResult
			require.ErrorIs(t, s.sweepOne(t.Context(), "inst-1", sweepNow, sweepCutoff, &got), apperror.ErrUnavailable)
			assert.Equal(t, SweepResult{Claimed: 1, Failed: 1}, got)
		})

		t.Run("lease を閉じる操作の失敗は失敗として数える", func(t *testing.T) {
			t.Parallel()

			s, leases, reclaimer, _ := newOrphanSweeper(t)
			leases.EXPECT().AcquireCleanup(gomock.Any(), gomock.Any()).Return(true, nil)
			reclaimer.EXPECT().Reclaim(gomock.Any(), gomock.Any()).Return(nil)
			leases.EXPECT().ReleaseCleanup(gomock.Any(), gomock.Any()).Return(false, apperror.ErrUnavailable)

			var got SweepResult
			require.ErrorIs(t, s.sweepOne(t.Context(), "inst-1", sweepNow, sweepCutoff, &got), apperror.ErrUnavailable)
			assert.Equal(t, SweepResult{Claimed: 1, Failed: 1}, got)
		})
	})
}
