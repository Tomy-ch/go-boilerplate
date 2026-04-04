package search

import (
	"context"
	"strings"
	"testing"

	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/testkit"
	"boilerplate-go/internal/usecase/tools/paging"
	"boilerplate-go/internal/usecase/user/search/query"
	mock_query "boilerplate-go/internal/usecase/user/search/query/mock"

	"boilerplate-go/pkg/ptr"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	userQS := mock_query.NewMockUserQueryService(ctrl)

	expected := &usecase{
		tracer: tf.Usecase(),
		userQS: userQS,
	}
	actual := New(tf, userQS)

	require.Equal(t, expected, actual)
}

func Test_usecase_ListUsersByKeyword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("キーワードでユーザーを検索できること", func(t *testing.T) {
			t.Parallel()

			page := 1
			perPage := 100
			p, err := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, err)

			expected := query.UserSearchResults{
				{
					FirstName:      "Grace",
					LastName:       "Lee",
					Email:          "grace.lee@example.com",
					Phone:          "090-1234-5678",
					PostalCode:     "123-456-7890",
					PrefectureName: "",
				},
			}

			keyword := "Grace Lee"
			active := ptr.To(true)

			keywords := strings.Split(keyword, " ")

			searchFilter := &query.UserSearchFilter{
				Active:   active,
				Keywords: keywords,
			}

			ctrl := gomock.NewController(t)
			userQS := mock_query.NewMockUserQueryService(ctrl)
			userQS.EXPECT().FindByFilter(gomock.Any(), searchFilter, p.Limit32(), p.Offset32()).Return(expected, nil)

			qs := &usecase{
				tracer: lt,
				userQS: userQS,
			}

			filter := &SearchParams{
				Keyword: ptr.To(keyword),
				Active:  active,
			}

			actual, err := qs.ListUsersByKeyword(ctx, filter, p)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーワードでの検索時にエラーが発生した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			expectedErr := testkit.ExpectedDBError(t)

			page := 1
			perPage := 100
			p, err := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, err)

			keyword := "Grace Lee"
			active := ptr.To(true)

			keywords := strings.Split(keyword, " ")

			searchFilter := &query.UserSearchFilter{
				Active:   active,
				Keywords: keywords,
			}

			ctrl := gomock.NewController(t)
			userQS := mock_query.NewMockUserQueryService(ctrl)
			userQS.EXPECT().FindByFilter(gomock.Any(), searchFilter, p.Limit32(), p.Offset32()).Return(nil, expectedErr)

			uc := &usecase{
				tracer: lt,
				userQS: userQS,
			}

			filter := &SearchParams{
				Keyword: ptr.To(keyword),
				Active:  active,
			}

			actual, err := uc.ListUsersByKeyword(ctx, filter, p)
			require.ErrorIs(t, err, expectedErr)
			require.Nil(t, actual)
		})
	})
}

func Test_usecase_CountUsersByKeyword(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーワードに基づくユーザー数の取得が正常に実行されること", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			userQS := mock_query.NewMockUserQueryService(ctrl)
			tf := observability.NewNoopTracerFactory(t)

			u := &usecase{
				tracer: tf.Usecase(),
				userQS: userQS,
			}

			expectedCount := int64(10)

			active := ptr.To(true)
			keyword := "Grace Lee"
			keywords := strings.Split(keyword, " ")

			searchFilter := &query.UserSearchFilter{
				Active:   active,
				Keywords: keywords,
			}

			userQS.EXPECT().
				CountByFilter(gomock.Any(), searchFilter).
				Return(expectedCount, nil)

			filter := &SearchParams{
				Active:  active,
				Keyword: ptr.To(keyword),
			}

			actualCount, err := u.CountUsersByKeyword(ctx, filter)
			require.NoError(t, err)
			require.Equal(t, expectedCount, actualCount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーワードに基づくユーザー数の取得時にエラーが発生した場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			userQS := mock_query.NewMockUserQueryService(ctrl)
			tf := observability.NewNoopTracerFactory(t)

			u := &usecase{
				tracer: tf.Usecase(),
				userQS: userQS,
			}

			expectedErr := testkit.ExpectedDBError(t)

			active := ptr.To(true)
			keyword := "Grace Lee"
			keywords := strings.Split(keyword, " ")

			searchFilter := &query.UserSearchFilter{
				Active:   active,
				Keywords: keywords,
			}

			userQS.EXPECT().
				CountByFilter(gomock.Any(), searchFilter).
				Return(int64(0), expectedErr)

			filter := &SearchParams{
				Active:  active,
				Keyword: ptr.To(keyword),
			}

			actualCount, err := u.CountUsersByKeyword(ctx, filter)
			require.ErrorIs(t, err, expectedErr)
			require.Equal(t, int64(0), actualCount)
		})
	})
}
