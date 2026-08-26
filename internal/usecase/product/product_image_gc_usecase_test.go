package product

import (
	"context"
	"testing"
	"time"

	mock_product "go-boilerplate/internal/domain/product/mock"
	"go-boilerplate/internal/observability"
	clocktest "go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/internal/usecase/boundary/objectstorage"
	mock_objectstorage "go-boilerplate/internal/usecase/boundary/objectstorage/mock"
	"go-boilerplate/internal/usecase/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewImageGC(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)

	uc := NewImageGC(tf, clocktest.NewMockClock(t, time.Time{}),
		mock_objectstorage.NewMockStorage(ctrl), mock_product.NewMockRepository(ctrl))
	require.NotNil(t, uc)
}

func Test_imageGCUsecase_SweepOrphans(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)
	grace := time.Hour
	cutoff := now.Add(-grace)

	aged := cutoff.Add(-time.Minute) // 猶予期間を過ぎている
	fresh := cutoff.Add(time.Minute) // 猶予期間内

	newUsecase := func(t *testing.T, ctrl *gomock.Controller) (
		*imageGCUsecase, *mock_objectstorage.MockStorage, *mock_product.MockRepository,
	) {
		t.Helper()
		storage := mock_objectstorage.NewMockStorage(ctrl)
		repo := mock_product.NewMockRepository(ctrl)
		return &imageGCUsecase{
			tracer:      observability.NewNoopTracerFactory(t).Usecase(),
			clock:       clocktest.NewMockClock(t, now),
			storage:     storage,
			productRepo: repo,
		}, storage, repo
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("猶予期間を過ぎた未参照オブジェクトを削除して件数を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), objectstorage.ListQuery{Prefix: imageKeyPrefix, Limit: 2}).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/a.png", ModifiedAt: aged},
					{Key: "products/b.png", ModifiedAt: aged},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), []string{"products/a.png", "products/b.png"}).
				Return(nil, nil)
			storage.EXPECT().Delete(gomock.Any(), []string{"products/a.png", "products/b.png"}).Return(nil)

			got, err := uc.SweepOrphans(context.Background(), grace, 2, false)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{Deleted: 2, Scanned: 2}, got)
		})

		t.Run("商品が参照しているオブジェクトは削除しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/used.png", ModifiedAt: aged},
					{Key: "products/orphan.png", ModifiedAt: aged},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).
				Return([]string{"products/used.png"}, nil)
			storage.EXPECT().Delete(gomock.Any(), []string{"products/orphan.png"}).Return(nil)

			got, err := uc.SweepOrphans(context.Background(), grace, 2, false)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{Deleted: 1, Scanned: 2}, got)
		})

		t.Run("猶予期間内のオブジェクトは未参照でも照合対象にしない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/fresh.png", ModifiedAt: fresh},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Times(0)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.SweepOrphans(context.Background(), grace, 2, false)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{}, got)
		})

		t.Run("猶予期間ちょうどのオブジェクトは削除しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/edge.png", ModifiedAt: cutoff},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Times(0)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.SweepOrphans(context.Background(), grace, 2, false)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{}, got)
		})

		t.Run("接頭辞の異なるオブジェクトは照合も削除もしない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "invoices/2026.pdf", ModifiedAt: aged},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Times(0)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.SweepOrphans(context.Background(), grace, 2, false)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{}, got)
		})

		t.Run("次カーソルがある限り反復し合計件数を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)
			uc.clock = clocktest.NewMockClockOnce(t, now)

			gomock.InOrder(
				storage.EXPECT().List(gomock.Any(), objectstorage.ListQuery{Prefix: imageKeyPrefix, Limit: 1}).
					Return(objectstorage.ListResult{
						Objects:    []objectstorage.Object{{Key: "products/p1.png", ModifiedAt: aged}},
						NextCursor: "cursor-1",
					}, nil),
				storage.EXPECT().List(gomock.Any(), objectstorage.ListQuery{Prefix: imageKeyPrefix, Cursor: "cursor-1", Limit: 1}).
					Return(objectstorage.ListResult{
						Objects: []objectstorage.Object{{Key: "products/p2.png", ModifiedAt: aged}},
					}, nil),
			)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(2)

			got, err := uc.SweepOrphans(context.Background(), grace, 1, false)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{Deleted: 2, Scanned: 2}, got)
		})

		t.Run("dryRunでは削除せず対象件数だけを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/orphan.png", ModifiedAt: aged},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Return(nil, nil)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.SweepOrphans(context.Background(), grace, 2, true)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{Deleted: 1, Scanned: 1}, got)
		})

		t.Run("graceが0以下なら既定の猶予期間で打ち切り時刻を決める", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			// 既定猶予期間の内側にある更新時刻を渡し、既定値が使われたことを削除されないことで示す。
			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/x.png", ModifiedAt: now.Add(-DefaultImageGCGrace).Add(time.Minute)},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.SweepOrphans(context.Background(), 0, 2, false)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{}, got)
		})

		t.Run("batchSizeが0以下なら既定のページ件数を使う", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, _ := newUsecase(t, ctrl)

			storage.EXPECT().
				List(gomock.Any(), objectstorage.ListQuery{Prefix: imageKeyPrefix, Limit: DefaultImageGCBatchSize}).
				Return(objectstorage.ListResult{}, nil)

			got, err := uc.SweepOrphans(context.Background(), grace, 0, false)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{}, got)
		})

		t.Run("オブジェクトが1件も無ければ照合も削除も行わない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).Return(objectstorage.ListResult{}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Times(0)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.SweepOrphans(context.Background(), grace, 2, false)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{}, got)
		})

		t.Run("全件が参照済みなら削除を呼ばない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/used.png", ModifiedAt: aged},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).
				Return([]string{"products/used.png"}, nil)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.SweepOrphans(context.Background(), grace, 2, false)

			require.NoError(t, err)
			assert.Equal(t, ImageGCResult{Scanned: 1}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("列挙の失敗はゼロ値とエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, _ := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{}, testkit.ExpectedDBError())

			got, err := uc.SweepOrphans(context.Background(), grace, 2, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, ImageGCResult{}, got)
		})

		t.Run("参照照合の失敗ではオブジェクトを1件も削除しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/a.png", ModifiedAt: aged},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).
				Return(nil, testkit.ExpectedDBError())
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.SweepOrphans(context.Background(), grace, 2, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, ImageGCResult{}, got)
		})

		t.Run("削除の失敗はゼロ値とエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/a.png", ModifiedAt: aged},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Return(nil, nil)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(testkit.ExpectedDBError())

			got, err := uc.SweepOrphans(context.Background(), grace, 2, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, ImageGCResult{}, got)
		})

		t.Run("2ページ目が失敗しても1ページ目の削除済み件数はエラーと併せて返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)
			uc.clock = clocktest.NewMockClockOnce(t, now)

			gomock.InOrder(
				storage.EXPECT().List(gomock.Any(), objectstorage.ListQuery{Prefix: imageKeyPrefix, Limit: 1}).
					Return(objectstorage.ListResult{
						Objects:    []objectstorage.Object{{Key: "products/p1.png", ModifiedAt: aged}},
						NextCursor: "cursor-1",
					}, nil),
				storage.EXPECT().List(gomock.Any(), objectstorage.ListQuery{Prefix: imageKeyPrefix, Cursor: "cursor-1", Limit: 1}).
					Return(objectstorage.ListResult{}, testkit.ExpectedDBError()),
			)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Return(nil, nil)
			storage.EXPECT().Delete(gomock.Any(), []string{"products/p1.png"}).Return(nil)

			got, err := uc.SweepOrphans(context.Background(), grace, 1, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, ImageGCResult{Deleted: 1, Scanned: 1}, got)
		})
	})
}

func Test_imageGCUsecase_sweepPage(t *testing.T) {
	t.Parallel()

	cutoff := time.Date(2026, time.June, 1, 11, 0, 0, 0, time.UTC)
	aged := cutoff.Add(-time.Minute)

	newUsecase := func(t *testing.T, ctrl *gomock.Controller) (
		*imageGCUsecase, *mock_objectstorage.MockStorage, *mock_product.MockRepository,
	) {
		t.Helper()
		storage := mock_objectstorage.NewMockStorage(ctrl)
		repo := mock_product.NewMockRepository(ctrl)
		return &imageGCUsecase{
			tracer:      observability.NewNoopTracerFactory(t).Usecase(),
			clock:       clocktest.NewMockClock(t, cutoff),
			storage:     storage,
			productRepo: repo,
		}, storage, repo
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受け取ったカーソルをそのまま列挙条件へ渡す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, _ := newUsecase(t, ctrl)

			storage.EXPECT().
				List(gomock.Any(), objectstorage.ListQuery{Prefix: imageKeyPrefix, Cursor: "c1", Limit: 5}).
				Return(objectstorage.ListResult{}, nil)

			_, err := uc.sweepPage(context.Background(), cutoff, "c1", 5, false)

			require.NoError(t, err)
		})

		t.Run("列挙の次カーソルをそのまま結果へ載せる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, _ := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{NextCursor: "c2"}, nil)

			got, err := uc.sweepPage(context.Background(), cutoff, "", 5, false)

			require.NoError(t, err)
			assert.Equal(t, "c2", got.nextCursor)
		})

		t.Run("候補が無ければ照合も削除も行わず次カーソルだけを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{NextCursor: "c2"}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Times(0)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.sweepPage(context.Background(), cutoff, "", 5, false)

			require.NoError(t, err)
			assert.Equal(t, imageGCPageResult{nextCursor: "c2"}, got)
		})

		t.Run("削除件数と検査件数を分けて数える", func(t *testing.T) {
			t.Parallel()

			// 検査件数は候補の全件、削除件数は参照済みを除いた分。片方に寄せると
			// 「照合したが消さなかった」量が運用者から見えなくなる。
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/used.png", ModifiedAt: aged},
					{Key: "products/orphan.png", ModifiedAt: aged},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).
				Return([]string{"products/used.png"}, nil)
			storage.EXPECT().Delete(gomock.Any(), []string{"products/orphan.png"}).Return(nil)

			got, err := uc.sweepPage(context.Background(), cutoff, "", 5, false)

			require.NoError(t, err)
			assert.Equal(t, imageGCPageResult{scanned: 2, deleted: 1}, got)
		})

		t.Run("dryRunでは削除を呼ばず対象件数だけを数える", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/orphan.png", ModifiedAt: aged},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Return(nil, nil)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.sweepPage(context.Background(), cutoff, "", 5, true)

			require.NoError(t, err)
			assert.Equal(t, imageGCPageResult{scanned: 1, deleted: 1}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("参照照合の失敗では次カーソルも返さずゼロ値とエラーを返す", func(t *testing.T) {
			t.Parallel()

			// 次カーソルを返すと呼び手が前進してしまい、照合できなかったページを飛ばすことになる。
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{
					Objects:    []objectstorage.Object{{Key: "products/a.png", ModifiedAt: aged}},
					NextCursor: "c2",
				}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).
				Return(nil, testkit.ExpectedDBError())
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

			got, err := uc.sweepPage(context.Background(), cutoff, "", 5, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, imageGCPageResult{}, got)
		})

		t.Run("削除の失敗はゼロ値とエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			uc, storage, repo := newUsecase(t, ctrl)

			storage.EXPECT().List(gomock.Any(), gomock.Any()).
				Return(objectstorage.ListResult{Objects: []objectstorage.Object{
					{Key: "products/a.png", ModifiedAt: aged},
				}}, nil)
			repo.EXPECT().FilterExistingImagePaths(gomock.Any(), gomock.Any()).Return(nil, nil)
			storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(testkit.ExpectedDBError())

			got, err := uc.sweepPage(context.Background(), cutoff, "", 5, false)

			require.ErrorIs(t, err, testkit.ExpectedDBError())
			assert.Equal(t, imageGCPageResult{}, got)
		})
	})
}

func Test_agedImageKeys(t *testing.T) {
	t.Parallel()

	cutoff := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接頭辞が一致し打ち切り時刻より古いキーだけを返す", func(t *testing.T) {
			t.Parallel()
			got := agedImageKeys([]objectstorage.Object{
				{Key: "products/old.png", ModifiedAt: cutoff.Add(-time.Second)},
				{Key: "products/new.png", ModifiedAt: cutoff.Add(time.Second)},
				{Key: "invoices/old.pdf", ModifiedAt: cutoff.Add(-time.Second)},
			}, cutoff)
			assert.Equal(t, []string{"products/old.png"}, got)
		})

		t.Run("打ち切り時刻ちょうどのキーは含まない", func(t *testing.T) {
			t.Parallel()
			got := agedImageKeys([]objectstorage.Object{
				{Key: "products/edge.png", ModifiedAt: cutoff},
			}, cutoff)
			assert.Empty(t, got)
		})

		t.Run("対象が無ければ空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, agedImageKeys(nil, cutoff))
		})
	})
}

func Test_excludeKeys(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("除外が空なら元の並びをそのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"a", "b"}, excludeKeys([]string{"a", "b"}, nil))
		})

		t.Run("除外に含まれるキーだけを順序を保って取り除く", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"a", "c"}, excludeKeys([]string{"a", "b", "c"}, []string{"b"}))
		})

		t.Run("全件が除外対象なら空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, excludeKeys([]string{"a"}, []string{"a"}))
		})

		t.Run("元の並びに無いキーが除外に含まれても影響しない", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []string{"a"}, excludeKeys([]string{"a"}, []string{"z"}))
		})
	})
}
