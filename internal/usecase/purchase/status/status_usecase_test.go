package status

import (
	"context"
	"testing"

	domainstatus "go-boilerplate/internal/domain/purchase/status"
	mock_status "go-boilerplate/internal/domain/purchase/status/mock"
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

		t.Run("取得した購入ステータスエンティティをsortKey昇順のDTOへ写像して返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)

			canceledID, err := uuid.Parse("e9d72547-adfe-48d9-9037-bd1f55d4158b")
			require.NoError(t, err)
			paidID, err := uuid.Parse("4b8f0e2a-1c3d-4a5e-8b7f-2d9c0e1a3b4c")
			require.NoError(t, err)

			// sortKey 昇順（キャンセル sortKey=6, 支払い済み sortKey=7）でリポジトリが返す想定。
			canceled, err := domainstatus.New(canceledID, domainstatus.Attributes{Name: "キャンセル", Code: 6, SortKey: 6})
			require.NoError(t, err)
			paid, err := domainstatus.New(paidID, domainstatus.Attributes{Name: "支払い済み", Code: 7, SortKey: 7})
			require.NoError(t, err)

			repo := mock_status.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any()).Return(
				domainstatus.Statuses{canceled, paid}, nil,
			).Times(1)

			u := &usecase{tracer: lt, repo: repo}

			actual, err := u.ListStatuses(ctx)
			require.NoError(t, err)
			assert.Equal(t, StatusDTOs{
				{ID: canceledID, Code: 6, Name: "キャンセル"},
				{ID: paidID, Code: 7, Name: "支払い済み"},
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
