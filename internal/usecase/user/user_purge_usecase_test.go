package user

import (
	"context"
	"testing"
	"time"

	mock_purchase "go-boilerplate/internal/domain/purchase/mock"
	mock_user "go-boilerplate/internal/domain/user/mock"
	"go-boilerplate/internal/observability"
	clocktest "go-boilerplate/internal/usecase/boundary/clock/testkit"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewPurge(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を渡すと同じ依存を保持したPurgeUsecaseを生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)
			txm := mock_tx.NewMockManager(ctrl)
			clk := clocktest.NewMockClock(t, time.Time{})
			userRepo := mock_user.NewMockRepository(ctrl)
			purchaseRepo := mock_purchase.NewMockRepository(ctrl)

			expected := &purgeUsecase{
				tracer:       tf.Usecase(),
				txm:          txm,
				clock:        clk,
				userRepo:     userRepo,
				purchaseRepo: purchaseRepo,
			}

			assert.Equal(t, expected, NewPurge(tf, txm, clk, userRepo, purchaseRepo))
		})
	})
}

func Test_purgeUsecase_PurgeDeleted(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	retention := 24 * time.Hour
	cutoff := now.Add(-retention)

	// ID の昇順が keyset の前進順になるよう、明示的に順序付けた候補 ID を用意する。
	id1 := uuidtestkit.NewTestFromSalt(t, "purge_user_1")
	id2 := uuidtestkit.NewTestFromSalt(t, "purge_user_2")
	id3 := uuidtestkit.NewTestFromSalt(t, "purge_user_3")

	newUsecase := func(t *testing.T, ctrl *gomock.Controller, currentTime time.Time) (*purgeUsecase, *mock_user.MockRepository, *mock_purchase.MockRepository) {
		t.Helper()
		userRepo := mock_user.NewMockRepository(ctrl)
		purchaseRepo := mock_purchase.NewMockRepository(ctrl)
		return &purgeUsecase{
			tracer:       observability.NewMockUsecaseLayerTracer(t),
			txm:          testkit.NewMockTransactionManager(t),
			clock:        clocktest.NewMockClock(t, currentTime),
			userRepo:     userRepo,
			purchaseRepo: purchaseRepo,
		}, userRepo, purchaseRepo
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入を持たない候補をすべて物理削除して件数を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, userRepo, purchaseRepo := newUsecase(t, ctrl, now)

			userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(2)).
				Return([]uuid.UUID{id1}, nil)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id1}).
				Return(nil, nil)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{id1}).Return(int64(1), nil)

			got, err := uc.PurgeDeleted(context.Background(), retention, 2, false)

			require.NoError(t, err)
			assert.Equal(t, PurgeResult{Purged: 1}, got)
		})

		t.Run("購入を持つ候補は削除せずスキップ件数に計上する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, userRepo, purchaseRepo := newUsecase(t, ctrl, now)

			userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(3)).
				Return([]uuid.UUID{id1, id2}, nil)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id1, id2}).
				Return([]uuid.UUID{id2}, nil)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{id1}).Return(int64(1), nil)

			got, err := uc.PurgeDeleted(context.Background(), retention, 3, false)

			require.NoError(t, err)
			assert.Equal(t, PurgeResult{Purged: 1, SkippedWithPurchases: 1}, got)
		})

		t.Run("1バッチが全件スキップでも境界を進めて次バッチへ移り無限ループしない", func(t *testing.T) {
			t.Parallel()

			// 先頭バッチの候補がすべて購入保持（削除 0 件）でも、境界を候補末尾まで進めることで
			// 同じ候補を取り直さずに前進することを固定する。境界を進めない実装は 1 バッチ目と
			// 同じ引数（afterID=nil）で再取得するため、以下の InOrder 期待に一致せず失敗する。
			ctrl := gomock.NewController(t)
			uc, userRepo, purchaseRepo := newUsecase(t, ctrl, now)
			// 打ち切り時刻はバッチ数によらず 1 度だけ決める。反復中に取り直すと保持期間の境界が
			// 実行中に前進し、削除対象がジョブの途中で変わってしまう。
			uc.clock = clocktest.NewMockClockOnce(t, now)

			gomock.InOrder(
				userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(2)).
					Return([]uuid.UUID{id1, id2}, nil),
				userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, &id2, int32(2)).
					Return([]uuid.UUID{id3}, nil),
			)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id1, id2}).
				Return([]uuid.UUID{id1, id2}, nil)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id3}).
				Return(nil, nil)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{}).Return(int64(0), nil)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{id3}).Return(int64(1), nil)

			got, err := uc.PurgeDeleted(context.Background(), retention, 2, false)

			require.NoError(t, err)
			assert.Equal(t, PurgeResult{Purged: 1, SkippedWithPurchases: 2}, got)
		})

		t.Run("バッチが満ちる限り境界を進めて反復し合計件数を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, userRepo, purchaseRepo := newUsecase(t, ctrl, now)
			uc.clock = clocktest.NewMockClockOnce(t, now)

			gomock.InOrder(
				userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(2)).
					Return([]uuid.UUID{id1, id2}, nil),
				userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, &id2, int32(2)).
					Return([]uuid.UUID{id3}, nil),
			)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), gomock.Any()).
				Return(nil, nil).Times(2)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{id1, id2}).Return(int64(2), nil)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{id3}).Return(int64(1), nil)

			got, err := uc.PurgeDeleted(context.Background(), retention, 2, false)

			require.NoError(t, err)
			assert.Equal(t, PurgeResult{Purged: 3}, got)
		})

		t.Run("retentionが0以下なら既定の保持期間で打ち切り時刻を決める", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, userRepo, _ := newUsecase(t, ctrl, now)

			userRepo.EXPECT().
				FindDeletedBefore(gomock.Any(), now.Add(-DefaultPurgeRetention), nil, int32(1)).
				Return(nil, nil)

			got, err := uc.PurgeDeleted(context.Background(), 0, 1, false)

			require.NoError(t, err)
			assert.Equal(t, PurgeResult{}, got)
		})

		t.Run("batchSizeが0以下なら既定のバッチサイズを使う", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, userRepo, _ := newUsecase(t, ctrl, now)

			userRepo.EXPECT().
				FindDeletedBefore(gomock.Any(), cutoff, nil, DefaultPurgeBatchSize).
				Return(nil, nil)

			got, err := uc.PurgeDeleted(context.Background(), retention, 0, false)

			require.NoError(t, err)
			assert.Equal(t, PurgeResult{}, got)
		})

		t.Run("dryRunでは削除せず対象件数だけを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, userRepo, purchaseRepo := newUsecase(t, ctrl, now)

			userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(3)).
				Return([]uuid.UUID{id1, id2}, nil)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id1, id2}).
				Return([]uuid.UUID{id2}, nil)
			// PurgeByIDs の EXPECT を置かないため、呼ばれた時点で gomock が失敗させる。

			got, err := uc.PurgeDeleted(context.Background(), retention, 3, true)

			require.NoError(t, err)
			assert.Equal(t, PurgeResult{Purged: 1, SkippedWithPurchases: 1}, got)
		})

		t.Run("候補が1件も無ければ何も削除せずゼロ件を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, userRepo, _ := newUsecase(t, ctrl, now)

			userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(5)).
				Return(nil, nil)

			got, err := uc.PurgeDeleted(context.Background(), retention, 5, false)

			require.NoError(t, err)
			assert.Equal(t, PurgeResult{}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("候補列挙の失敗はゼロ値とエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, userRepo, _ := newUsecase(t, ctrl, now)

			userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(2)).
				Return(nil, testkit.ExpectedDBError())

			got, err := uc.PurgeDeleted(context.Background(), retention, 2, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, PurgeResult{}, got)
		})

		t.Run("購入照会の失敗はゼロ値とエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, userRepo, purchaseRepo := newUsecase(t, ctrl, now)

			userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(2)).
				Return([]uuid.UUID{id1}, nil)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id1}).
				Return(nil, testkit.ExpectedDBError())

			got, err := uc.PurgeDeleted(context.Background(), retention, 2, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, PurgeResult{}, got)
		})

		t.Run("物理削除の失敗はゼロ値とエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, userRepo, purchaseRepo := newUsecase(t, ctrl, now)

			userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(2)).
				Return([]uuid.UUID{id1}, nil)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id1}).
				Return(nil, nil)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{id1}).
				Return(int64(0), testkit.ExpectedDBError())

			got, err := uc.PurgeDeleted(context.Background(), retention, 2, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, PurgeResult{}, got)
		})

		t.Run("2バッチ目が失敗しても1バッチ目のコミット済み件数はエラーと併せて返す", func(t *testing.T) {
			t.Parallel()

			// 失敗したバッチは巻き戻るが、コミット済みのバッチの物理削除は取り消せない。
			// 累計を捨てると、実際に消えた件数が呼び手から見えなくなる。
			ctrl := gomock.NewController(t)
			uc, userRepo, purchaseRepo := newUsecase(t, ctrl, now)

			gomock.InOrder(
				userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(1)).
					Return([]uuid.UUID{id1}, nil),
				userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, &id1, int32(1)).
					Return(nil, testkit.ExpectedDBError()),
			)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id1}).
				Return(nil, nil)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{id1}).Return(int64(1), nil)

			got, err := uc.PurgeDeleted(context.Background(), retention, 1, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, PurgeResult{Purged: 1}, got)
		})

		t.Run("2バッチ目が失敗しても1バッチ目のスキップ件数はエラーと併せて返す", func(t *testing.T) {
			t.Parallel()

			// スキップは削除の副産物ではなく「消せなかった対象」の報告なので、削除件数と同じく失っては困る。
			ctrl := gomock.NewController(t)
			uc, userRepo, purchaseRepo := newUsecase(t, ctrl, now)

			gomock.InOrder(
				userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(1)).
					Return([]uuid.UUID{id1}, nil),
				userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, &id1, int32(1)).
					Return(nil, testkit.ExpectedDBError()),
			)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id1}).
				Return([]uuid.UUID{id1}, nil)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{}).Return(int64(0), nil)

			got, err := uc.PurgeDeleted(context.Background(), retention, 1, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, PurgeResult{SkippedWithPurchases: 1}, got)
		})
	})
}

func Test_purgeUsecase_purgeBatch(t *testing.T) {
	t.Parallel()

	cutoff := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	id1 := uuidtestkit.NewTestFromSalt(t, "purge_batch_user_1")
	id2 := uuidtestkit.NewTestFromSalt(t, "purge_batch_user_2")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("候補が無ければ購入照会も削除も行わず空の結果を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			// 購入 Repository に EXPECT を置かないため、空の候補で照会した時点で失敗する。
			purchaseRepo := mock_purchase.NewMockRepository(ctrl)
			uc := &purgeUsecase{
				tracer:       observability.NewMockUsecaseLayerTracer(t),
				txm:          testkit.NewMockTransactionManager(t),
				clock:        clocktest.NewMockClock(t, cutoff),
				userRepo:     userRepo,
				purchaseRepo: purchaseRepo,
			}
			userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(2)).
				Return(nil, nil)

			got, err := uc.purgeBatch(context.Background(), cutoff, nil, 2, false)

			require.NoError(t, err)
			assert.Equal(t, purgeBatchResult{}, got)
		})

		t.Run("次バッチの境界に候補の末尾を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			purchaseRepo := mock_purchase.NewMockRepository(ctrl)
			uc := &purgeUsecase{
				tracer:       observability.NewMockUsecaseLayerTracer(t),
				txm:          testkit.NewMockTransactionManager(t),
				clock:        clocktest.NewMockClock(t, cutoff),
				userRepo:     userRepo,
				purchaseRepo: purchaseRepo,
			}
			userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(2)).
				Return([]uuid.UUID{id1, id2}, nil)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id1, id2}).
				Return([]uuid.UUID{id1}, nil)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{id2}).Return(int64(1), nil)

			got, err := uc.purgeBatch(context.Background(), cutoff, nil, 2, false)

			require.NoError(t, err)
			assert.Equal(t, purgeBatchResult{candidates: 2, purged: 1, skipped: 1, nextAfter: &id2}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トランザクションのコミット失敗はゼロ値とエラーを返す", func(t *testing.T) {
			t.Parallel()

			wantErr := xerrors.New("commit failed")
			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			purchaseRepo := mock_purchase.NewMockRepository(ctrl)
			txm := mock_tx.NewMockManager(ctrl)
			txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, fn func(ctx context.Context) error) error {
					if err := fn(ctx); err != nil {
						return err
					}
					return wantErr
				},
			)
			uc := &purgeUsecase{
				tracer:       observability.NewMockUsecaseLayerTracer(t),
				txm:          txm,
				clock:        clocktest.NewMockClock(t, cutoff),
				userRepo:     userRepo,
				purchaseRepo: purchaseRepo,
			}
			userRepo.EXPECT().FindDeletedBefore(gomock.Any(), cutoff, nil, int32(2)).
				Return([]uuid.UUID{id1}, nil)
			purchaseRepo.EXPECT().FindUserIDsWithPurchases(gomock.Any(), []uuid.UUID{id1}).
				Return(nil, nil)
			userRepo.EXPECT().PurgeByIDs(gomock.Any(), []uuid.UUID{id1}).Return(int64(1), nil)

			got, err := uc.purgeBatch(context.Background(), cutoff, nil, 2, false)

			require.ErrorIs(t, err, wantErr)
			assert.Equal(t, purgeBatchResult{}, got)
		})
	})
}

func Test_excludeIDs(t *testing.T) {
	t.Parallel()

	id1 := uuidtestkit.NewTestFromSalt(t, "exclude_id_1")
	id2 := uuidtestkit.NewTestFromSalt(t, "exclude_id_2")
	id3 := uuidtestkit.NewTestFromSalt(t, "exclude_id_3")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("除外が空なら元の並びをそのまま返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, []uuid.UUID{id1, id2}, excludeIDs([]uuid.UUID{id1, id2}, nil))
		})

		t.Run("除外に含まれるIDだけを順序を保って取り除く", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, []uuid.UUID{id1, id3}, excludeIDs([]uuid.UUID{id1, id2, id3}, []uuid.UUID{id2}))
		})

		t.Run("全件が除外対象なら空を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, excludeIDs([]uuid.UUID{id1, id2}, []uuid.UUID{id2, id1}))
		})

		t.Run("元の並びに無いIDが除外に含まれても影響しない", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, []uuid.UUID{id1}, excludeIDs([]uuid.UUID{id1}, []uuid.UUID{id2, id3}))
		})
	})
}
