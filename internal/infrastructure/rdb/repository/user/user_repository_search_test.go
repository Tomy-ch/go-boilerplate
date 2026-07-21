package user

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_buildLikeTokens(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーワードが空の場合、全件マッチのトークンを返す", func(t *testing.T) {
			t.Parallel()

			actual := buildLikeTokens(nil)
			assert.Equal(t, []string{"%"}, actual)
		})

		t.Run("キーワードを部分一致のLIKEパターンへ変換する", func(t *testing.T) {
			t.Parallel()

			actual := buildLikeTokens([]string{"Grace", "Lee"})
			require.Len(t, actual, 2)
			assert.Equal(t, "%Grace%", actual[0])
			assert.Equal(t, "%Lee%", actual[1])
		})

		t.Run("LIKEワイルドカードを含むキーワードはエスケープされる", func(t *testing.T) {
			t.Parallel()

			actual := buildLikeTokens([]string{"50%_off"})
			require.Len(t, actual, 1)
			assert.Equal(t, `%50\%\_off%`, actual[0])
		})
	})
}

func Test_repository_SearchByKeyword(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	repo := &repository{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("activeがnilの場合、全ての状態のユーザーが対象になり作成日時の降順で返る", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Grace(アクティブ) は Charlie(削除済み) より後に作成されており、created_at DESC で先頭に来る。
			// = キーワードフィルタと並び順（created_at DESC）を同時に検証する。
			actual, err := repo.SearchByKeyword(ctx, []string{"Grace", "Charlie"}, nil, 10, 0)
			require.NoError(t, err)

			require.Len(t, actual, 2)
			assert.Equal(t, "Grace", actual[0].FirstName())
			assert.Equal(t, "Lee", actual[0].LastName())
			assert.Equal(t, "Charlie", actual[1].FirstName())
			assert.Equal(t, "Davis", actual[1].LastName())
		})

		t.Run("activeがtrueの場合、アクティブなユーザーのみが対象になる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := repo.SearchByKeyword(ctx, []string{"Grace"}, new(true), 10, 0)
			require.NoError(t, err)

			require.Len(t, actual, 1)
			assert.Equal(t, "Grace", actual[0].FirstName())
			assert.Nil(t, actual[0].DeletedAt())
		})

		t.Run("activeがfalseの場合、削除済みのユーザーのみが対象になる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := repo.SearchByKeyword(ctx, []string{"Charlie"}, new(false), 10, 0)
			require.NoError(t, err)

			require.Len(t, actual, 1)
			assert.Equal(t, "Charlie", actual[0].FirstName())
			assert.NotNil(t, actual[0].DeletedAt())
		})

		t.Run("keywordsが空の場合、全ユーザーが対象になる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := repo.SearchByKeyword(ctx, []string{}, nil, 20, 0)
			require.NoError(t, err)
			assert.Len(t, actual, 10)
		})

		t.Run("keywordsが空かつactive=trueの場合、アクティブのみが対象になる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := repo.SearchByKeyword(ctx, []string{}, new(true), 20, 0)
			require.NoError(t, err)
			assert.Len(t, actual, 8)
		})

		t.Run("keywordsが空かつactive=falseの場合、削除済みのみが対象になる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := repo.SearchByKeyword(ctx, []string{}, new(false), 20, 0)
			require.NoError(t, err)
			assert.Len(t, actual, 2)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストでは各Active分岐がErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, allErr := repo.SearchByKeyword(ctx, []string{"x"}, nil, 10, 0)
			require.ErrorIs(t, allErr, apperror.ErrCanceled)

			_, activeErr := repo.SearchByKeyword(ctx, []string{"x"}, new(true), 10, 0)
			require.ErrorIs(t, activeErr, apperror.ErrCanceled)

			_, deletedErr := repo.SearchByKeyword(ctx, []string{"x"}, new(false), 10, 0)
			require.ErrorIs(t, deletedErr, apperror.ErrCanceled)
		})

		t.Run("limitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := repo.SearchByKeyword(ctx, []string{}, nil, -1, 0)
			require.Nil(t, actual)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_repository_CountByKeyword(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	repo := &repository{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("activeがnilかつkeywordsが空の場合、全ユーザーの件数が返る", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := repo.CountByKeyword(ctx, []string{}, nil)
			require.NoError(t, err)
			assert.Equal(t, int64(10), actual)
		})

		t.Run("activeがtrueかつkeywordsが空の場合、アクティブなユーザーの件数が返る", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := repo.CountByKeyword(ctx, []string{}, new(true))
			require.NoError(t, err)
			assert.Equal(t, int64(8), actual)
		})

		t.Run("activeがfalseかつkeywordsが空の場合、削除済みユーザーの件数が返る", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := repo.CountByKeyword(ctx, []string{}, new(false))
			require.NoError(t, err)
			assert.Equal(t, int64(2), actual)
		})

		t.Run("キーワードにマッチするユーザーの件数が返る", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := repo.CountByKeyword(ctx, []string{"Grace"}, nil)
			require.NoError(t, err)
			assert.Equal(t, int64(1), actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストでは各Active分岐がErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, allErr := repo.CountByKeyword(ctx, []string{"x"}, nil)
			require.ErrorIs(t, allErr, apperror.ErrCanceled)

			_, activeErr := repo.CountByKeyword(ctx, []string{"x"}, new(true))
			require.ErrorIs(t, activeErr, apperror.ErrCanceled)

			_, deletedErr := repo.CountByKeyword(ctx, []string{"x"}, new(false))
			require.ErrorIs(t, deletedErr, apperror.ErrCanceled)
		})
	})
}
