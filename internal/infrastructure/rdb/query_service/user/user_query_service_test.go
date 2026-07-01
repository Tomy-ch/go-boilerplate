package user

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user/search/query"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &service{
		tracer: tf.Infra(),
		db:     testDB,
	}
	actual := New(testDB, tf)
	assert.Equal(t, expected, actual)
}

func TestBuildLikeTokens(t *testing.T) {
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

func Test_service_FindByFilter(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	repo := &service{
		tracer: lt,
		db:     testDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーワードにマッチするユーザーが取得できる", func(t *testing.T) {
			t.Parallel()

			t.Run("activeがnilの場合、全てのユーザーが対象になる", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()

				firstName1 := "Grace"
				lastName1 := "Lee"
				firstName2 := "Charlie"
				lastName2 := "Davis"

				keywords := []string{firstName1, firstName2}
				limit := int32(10)
				offset := int32(0)

				expectedLength := 2

				actual, err := repo.FindByFilter(ctx, &query.UserSearchFilter{
					Keywords: keywords,
					Active:   nil,
				}, limit, offset)
				require.NoError(t, err)

				assert.Len(t, actual, expectedLength)
				assert.Equal(t, firstName1, actual[0].FirstName)
				assert.Equal(t, lastName1, actual[0].LastName)
				assert.Equal(t, firstName2, actual[1].FirstName)
				assert.Equal(t, lastName2, actual[1].LastName)
			})

			t.Run("activeがtrueの場合、アクティブなユーザーが対象になる", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()

				firstName := "Grace"
				lastName := "Lee"

				keywords := []string{firstName}
				limit := int32(10)
				offset := int32(0)

				expectedLength := 1

				actual, err := repo.FindByFilter(ctx, &query.UserSearchFilter{
					Keywords: keywords,
					Active:   new(true),
				}, limit, offset)
				require.NoError(t, err)

				assert.Len(t, actual, expectedLength)
				assert.Equal(t, firstName, actual[0].FirstName)
				assert.Equal(t, lastName, actual[0].LastName)
			})

			t.Run("activeがfalseの場合、削除されたユーザーが対象になる", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()

				firstName := "Charlie"
				lastName := "Davis"

				keywords := []string{firstName}
				limit := int32(10)
				offset := int32(0)

				expectedLength := 1

				actual, err := repo.FindByFilter(ctx, &query.UserSearchFilter{
					Keywords: keywords,
					Active:   new(false),
				}, limit, offset)
				require.NoError(t, err)

				assert.Len(t, actual, expectedLength)
				assert.Equal(t, firstName, actual[0].FirstName)
				assert.Equal(t, lastName, actual[0].LastName)
			})
		})

		t.Run("keywordsが空の場合、全件取得できる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			keywords := []string{}
			limit := int32(20)
			offset := int32(0)

			expectedLength := 10

			actual, err := repo.FindByFilter(ctx, &query.UserSearchFilter{
				Keywords: keywords,
				Active:   nil,
			}, limit, offset)
			require.NoError(t, err)

			assert.Len(t, actual, expectedLength)
		})

		t.Run("keywordsが空かつactive=trueの場合、アクティブのみ取得できる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			keywords := []string{}
			limit := int32(20)
			offset := int32(0)
			active := true

			expectedLength := 8

			actual, err := repo.FindByFilter(ctx, &query.UserSearchFilter{
				Keywords: keywords,
				Active:   &active,
			}, limit, offset)
			require.NoError(t, err)

			assert.Len(t, actual, expectedLength)
		})

		t.Run("keywordsが空かつactive=falseの場合、削除済みのみ取得できる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			keywords := []string{}
			limit := int32(20)
			offset := int32(0)
			active := false

			expectedLength := 2

			actual, err := repo.FindByFilter(ctx, &query.UserSearchFilter{
				Keywords: keywords,
				Active:   &active,
			}, limit, offset)
			require.NoError(t, err)

			assert.Len(t, actual, expectedLength)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストでは各Active分岐がErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			active := true
			deleted := false
			keywords := []string{"x"}

			_, allErr := repo.FindByFilter(ctx, &query.UserSearchFilter{Keywords: keywords, Active: nil}, 10, 0)
			require.ErrorIs(t, allErr, apperror.ErrCanceled)

			_, activeErr := repo.FindByFilter(ctx, &query.UserSearchFilter{Keywords: keywords, Active: &active}, 10, 0)
			require.ErrorIs(t, activeErr, apperror.ErrCanceled)

			_, deletedErr := repo.FindByFilter(ctx, &query.UserSearchFilter{Keywords: keywords, Active: &deleted}, 10, 0)
			require.ErrorIs(t, deletedErr, apperror.ErrCanceled)
		})
	})
}

func Test_service_CountByFilter(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	repo := &service{
		tracer: lt,
		db:     testDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("activeがnilかつ、keywordsが空の場合、全ユーザーが対象になる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			keywords := []string{}

			expectedCount := int64(10)

			actual, err := repo.CountByFilter(ctx, &query.UserSearchFilter{
				Keywords: keywords,
				Active:   nil,
			})
			require.NoError(t, err)
			assert.Equal(t, expectedCount, actual)
		})

		t.Run("activeがtrueかつ、keywordsが空の場合、アクティブなユーザーが対象になる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			keywords := []string{}
			active := true

			expectedCount := int64(8)

			actual, err := repo.CountByFilter(ctx, &query.UserSearchFilter{
				Keywords: keywords,
				Active:   &active,
			})
			require.NoError(t, err)
			assert.Equal(t, expectedCount, actual)
		})

		t.Run("activeがfalseかつ、keywordsが空の場合、削除されたユーザーが対象になる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			keywords := []string{}
			active := false

			expectedCount := int64(2)

			actual, err := repo.CountByFilter(ctx, &query.UserSearchFilter{
				Keywords: keywords,
				Active:   &active,
			})
			require.NoError(t, err)
			assert.Equal(t, expectedCount, actual)
		})

		t.Run("Keywordsにマッチするユーザーの総件数が取得できる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			keywords := []string{"Grace"}

			expectedCount := int64(1)

			actual, err := repo.CountByFilter(ctx, &query.UserSearchFilter{
				Keywords: keywords,
				Active:   nil,
			})
			require.NoError(t, err)
			assert.Equal(t, expectedCount, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストでは各Active分岐がErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			active := true
			deleted := false
			keywords := []string{"x"}

			_, allErr := repo.CountByFilter(ctx, &query.UserSearchFilter{Keywords: keywords, Active: nil})
			require.ErrorIs(t, allErr, apperror.ErrCanceled)

			_, activeErr := repo.CountByFilter(ctx, &query.UserSearchFilter{Keywords: keywords, Active: &active})
			require.ErrorIs(t, activeErr, apperror.ErrCanceled)

			_, deletedErr := repo.CountByFilter(ctx, &query.UserSearchFilter{Keywords: keywords, Active: &deleted})
			require.ErrorIs(t, deletedErr, apperror.ErrCanceled)
		})
	})
}
