package status

import (
	"context"
	"testing"

	domainstatus "go-boilerplate/internal/domain/product/status"
	mock_status "go-boilerplate/internal/domain/product/status/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)
		repo := mock_status.NewMockRepository(ctrl)

		expected := &usecase{
			tracer: tf.Usecase(),
			repo:   repo,
		}
		actual := New(repo, tf)

		assert.Equal(t, expected, actual)
	})
}

func Test_usecase_ListStatuses(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得した商品ステータスエンティティをsortKey昇順のDTOへ写像して返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)

			reviewingID, err := uuid.Parse("bdf44f06-227c-4549-b2c8-e57b32f06321")
			require.NoError(t, err)
			inStockID, err := uuid.Parse("093170fb-83a2-4864-a2b3-53236eaf3597")
			require.NoError(t, err)

			// sortKey 昇順（検討中 sortKey=1, 在庫あり sortKey=5）でリポジトリが返す想定。
			reviewing, err := domainstatus.New(reviewingID, "検討中", 8, 1)
			require.NoError(t, err)
			inStock, err := domainstatus.New(inStockID, "在庫あり", 1, 5)
			require.NoError(t, err)

			repo := mock_status.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any()).Return(
				domainstatus.Statuses{reviewing, inStock}, nil,
			).Times(1)

			u := &usecase{tracer: lt, repo: repo}

			actual, err := u.ListStatuses(ctx)
			require.NoError(t, err)
			assert.Equal(t, StatusDTOs{
				{ID: reviewingID, Code: 8, Name: "検討中", SortKey: 1},
				{ID: inStockID, Code: 1, Name: "在庫あり", SortKey: 5},
			}, actual)
		})

		t.Run("取得結果が0件の場合、nilではない空のDTO一覧を返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)

			repo := mock_status.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any()).Return(domainstatus.Statuses{}, nil).Times(1)

			u := &usecase{tracer: lt, repo: repo}

			actual, err := u.ListStatuses(ctx)
			require.NoError(t, err)
			assert.NotNil(t, actual)
			assert.Empty(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リポジトリのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)

			expectedErr := testkit.ExpectedDBError()

			repo := mock_status.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any()).Return(nil, expectedErr).Times(1)

			u := &usecase{tracer: lt, repo: repo}

			actual, err := u.ListStatuses(ctx)
			require.ErrorIs(t, err, expectedErr)
			assert.Nil(t, actual)
		})
	})
}
