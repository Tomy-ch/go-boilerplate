package prefecture

import (
	"context"
	"testing"

	domainprefecture "go-boilerplate/internal/domain/prefecture"
	mock_prefecture "go-boilerplate/internal/domain/prefecture/mock"
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
		repo := mock_prefecture.NewMockRepository(ctrl)

		expected := &usecase{
			tracer: tf.Usecase(),
			repo:   repo,
		}
		actual := New(repo, tf)

		assert.Equal(t, expected, actual)
	})
}

func Test_usecase_ListPrefectures(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得した都道府県エンティティをcode昇順のDTOへ写像して返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)

			tokyoID, err := uuid.Parse("101caa1e-84e7-4ceb-9108-50d40b6be1a3")
			require.NoError(t, err)
			osakaID, err := uuid.Parse("d647fc85-ff46-4530-88cb-198f4a68a9d7")
			require.NoError(t, err)

			tokyo, err := domainprefecture.New(tokyoID, "東京都", 13)
			require.NoError(t, err)
			osaka, err := domainprefecture.New(osakaID, "大阪府", 27)
			require.NoError(t, err)

			repo := mock_prefecture.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any()).Return(domainprefecture.Prefectures{tokyo, osaka}, nil).Times(1)

			u := &usecase{tracer: lt, repo: repo}

			actual, err := u.ListPrefectures(ctx)
			require.NoError(t, err)
			assert.Equal(t, PrefectureDTOs{
				{ID: tokyoID, Code: 13, Name: "東京都"},
				{ID: osakaID, Code: 27, Name: "大阪府"},
			}, actual)
		})

		t.Run("取得結果が0件の場合、nilではない空のDTO一覧を返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)

			repo := mock_prefecture.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any()).Return(domainprefecture.Prefectures{}, nil).Times(1)

			u := &usecase{tracer: lt, repo: repo}

			actual, err := u.ListPrefectures(ctx)
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

			repo := mock_prefecture.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any()).Return(nil, expectedErr).Times(1)

			u := &usecase{tracer: lt, repo: repo}

			actual, err := u.ListPrefectures(ctx)
			require.ErrorIs(t, err, expectedErr)
			assert.Nil(t, actual)
		})
	})
}
