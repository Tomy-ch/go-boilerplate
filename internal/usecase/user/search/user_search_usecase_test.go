package search

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/prefecture"
	mock_prefecture "go-boilerplate/internal/domain/prefecture/mock"
	"go-boilerplate/internal/domain/user"
	mock_user "go-boilerplate/internal/domain/user/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newSearchTestUser は、検索テスト用のユーザーエンティティを生成するヘルパーです。
func newSearchTestUser(t *testing.T, prefectureID uuid.UUID, createdAt time.Time) *user.User {
	t.Helper()
	u, err := user.New(
		uuid.NewTestFromSalt(t, "search_user_domain"),
		"Grace",
		"Lee",
		"grace.lee@example.com",
		"090-1234-5678",
		prefectureID,
		"city_name",
		"town_address",
		nil,
		"150-0001",
		createdAt,
		createdAt,
		nil,
	)
	require.NoError(t, err)
	return u
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)
		userRepo := mock_user.NewMockRepository(ctrl)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)

		expected := &usecase{
			tracer:   tf.Usecase(),
			userRepo: userRepo,
			pftRepo:  pftRepo,
		}
		actual := New(tf, userRepo, pftRepo)

		assert.Equal(t, expected, actual)
	})
}

func Test_usecase_ListUsersByKeyword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	prefectureID := uuid.NewTestFromSalt(t, "prefecture_domain")
	keyword := "Grace Lee"
	keywords := []string{"Grace", "Lee"}
	active := new(true)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーワードでユーザーを検索でき、都道府県名を解決した結果が返る", func(t *testing.T) {
			t.Parallel()

			page := 1
			perPage := 100
			p, err := paging.NewPageFrom1Based(&page, &perPage)
			require.NoError(t, err)

			userDomain := newSearchTestUser(t, prefectureID, now)
			prefectureDomain, err := prefecture.New(prefectureID, "Tokyo", 1)
			require.NoError(t, err)

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().
				SearchByKeyword(gomock.Any(), keywords, active, p.Limit32(), p.Offset32()).
				Return(user.Users{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().
				FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).
				Return(prefecture.Prefectures{prefectureDomain}, nil)

			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			filter := &SearchParams{Keyword: new(keyword), Active: active}
			actual, err := uc.ListUsersByKeyword(ctx, filter, p)
			require.NoError(t, err)

			expected := UserSearchResults{
				{
					FirstName:      userDomain.FirstName(),
					LastName:       userDomain.LastName(),
					Email:          userDomain.Email(),
					Phone:          userDomain.Phone(),
					PostalCode:     userDomain.PostalCode(),
					PrefectureName: prefectureDomain.Name(),
					City:           userDomain.City(),
					Street:         userDomain.Street(),
					Building:       userDomain.Building(),
					RegisteredAt:   userDomain.CreatedAt(),
					DeletedAt:      userDomain.DeletedAt(),
				},
			}
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザー検索時にエラーが発生した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			expectedErr := testkit.ExpectedDBError()

			page := 1
			perPage := 100
			p, err := paging.NewPageFrom1Based(&page, &perPage)
			require.NoError(t, err)

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().
				SearchByKeyword(gomock.Any(), keywords, active, p.Limit32(), p.Offset32()).
				Return(nil, expectedErr)

			uc := &usecase{tracer: lt, userRepo: userRepo}

			filter := &SearchParams{Keyword: new(keyword), Active: active}
			actual, err := uc.ListUsersByKeyword(ctx, filter, p)
			require.ErrorIs(t, err, expectedErr)
			require.Nil(t, actual)
		})

		t.Run("都道府県取得時にエラーが発生した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			expectedErr := testkit.ExpectedDBError()

			page := 1
			perPage := 100
			p, err := paging.NewPageFrom1Based(&page, &perPage)
			require.NoError(t, err)

			userDomain := newSearchTestUser(t, prefectureID, now)

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().
				SearchByKeyword(gomock.Any(), keywords, active, p.Limit32(), p.Offset32()).
				Return(user.Users{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().
				FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).
				Return(nil, expectedErr)

			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			filter := &SearchParams{Keyword: new(keyword), Active: active}
			actual, err := uc.ListUsersByKeyword(ctx, filter, p)
			require.ErrorIs(t, err, expectedErr)
			require.Nil(t, actual)
		})

		t.Run("ユーザーの都道府県が解決できない場合、ErrInternal が返される", func(t *testing.T) {
			t.Parallel()

			page := 1
			perPage := 100
			p, err := paging.NewPageFrom1Based(&page, &perPage)
			require.NoError(t, err)

			userDomain := newSearchTestUser(t, prefectureID, now)

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().
				SearchByKeyword(gomock.Any(), keywords, active, p.Limit32(), p.Offset32()).
				Return(user.Users{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().
				FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).
				Return(prefecture.Prefectures{}, nil)

			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			filter := &SearchParams{Keyword: new(keyword), Active: active}
			actual, err := uc.ListUsersByKeyword(ctx, filter, p)
			require.ErrorIs(t, err, apperror.ErrInternal)
			require.Nil(t, actual)
		})

		t.Run("filter が nil の場合、ErrInvalidArgument エラーになる", func(t *testing.T) {
			t.Parallel()

			page := 1
			perPage := 100
			p, err := paging.NewPageFrom1Based(&page, &perPage)
			require.NoError(t, err)

			uc := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase()}
			actual, err := uc.ListUsersByKeyword(ctx, nil, p)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})

		t.Run("page が nil の場合、ErrInvalidArgument エラーになる", func(t *testing.T) {
			t.Parallel()

			uc := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase()}
			actual, err := uc.ListUsersByKeyword(ctx, &SearchParams{}, nil)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})
	})
}

func Test_usecase_CountUsersByKeyword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	keyword := "Grace Lee"
	keywords := []string{"Grace", "Lee"}
	active := new(true)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キーワードに基づくユーザー数の取得が正常に実行されること", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			tf := observability.NewNoopTracerFactory(t)

			expectedCount := int64(10)
			userRepo.EXPECT().
				CountByKeyword(gomock.Any(), keywords, active).
				Return(expectedCount, nil)

			u := &usecase{tracer: tf.Usecase(), userRepo: userRepo}

			filter := &SearchParams{Active: active, Keyword: new(keyword)}
			actualCount, err := u.CountUsersByKeyword(ctx, filter)
			require.NoError(t, err)
			assert.Equal(t, expectedCount, actualCount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザー数の取得時にエラーが発生した場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			tf := observability.NewNoopTracerFactory(t)

			expectedErr := testkit.ExpectedDBError()
			userRepo.EXPECT().
				CountByKeyword(gomock.Any(), keywords, active).
				Return(int64(0), expectedErr)

			u := &usecase{tracer: tf.Usecase(), userRepo: userRepo}

			filter := &SearchParams{Active: active, Keyword: new(keyword)}
			actualCount, err := u.CountUsersByKeyword(ctx, filter)
			require.ErrorIs(t, err, expectedErr)
			assert.Equal(t, int64(0), actualCount)
		})

		t.Run("filter が nil の場合、ErrInvalidArgument エラーになる", func(t *testing.T) {
			t.Parallel()

			u := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase()}
			actualCount, err := u.CountUsersByKeyword(context.Background(), nil)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Equal(t, int64(0), actualCount)
		})
	})
}

func Test_usecase_ListUsersByKeywordWithTotal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	prefectureID := uuid.NewTestFromSalt(t, "prefecture_domain")
	keyword := "Grace Lee"
	keywords := []string{"Grace", "Lee"}
	active := new(true)

	page := 1
	perPage := 100
	p, err := paging.NewPageFrom1Based(&page, &perPage)
	require.NoError(t, err)

	filter := &SearchParams{Keyword: new(keyword), Active: active}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一覧と総件数をまとめて取得できること", func(t *testing.T) {
			t.Parallel()

			userDomain := newSearchTestUser(t, prefectureID, now)
			prefectureDomain, err := prefecture.New(prefectureID, "Tokyo", 1)
			require.NoError(t, err)

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().
				SearchByKeyword(gomock.Any(), keywords, active, p.Limit32(), p.Offset32()).
				Return(user.Users{userDomain}, nil)
			userRepo.EXPECT().
				CountByKeyword(gomock.Any(), keywords, active).
				Return(int64(1), nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().
				FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).
				Return(prefecture.Prefectures{prefectureDomain}, nil)

			u := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			actual, err := u.ListUsersByKeywordWithTotal(ctx, filter, p)
			require.NoError(t, err)
			require.NotNil(t, actual)
			require.Len(t, actual.Items, 1)
			assert.Equal(t, int64(1), actual.Total)
			assert.Equal(t, prefectureDomain.Name(), actual.Items[0].PrefectureName)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一覧取得でエラーが発生した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()
			expectedErr := testkit.ExpectedDBError()

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().
				SearchByKeyword(gomock.Any(), keywords, active, p.Limit32(), p.Offset32()).
				Return(nil, expectedErr)

			u := &usecase{tracer: lt, userRepo: userRepo}

			actual, err := u.ListUsersByKeywordWithTotal(ctx, filter, p)
			require.ErrorIs(t, err, expectedErr)
			require.Nil(t, actual)
		})

		t.Run("総件数取得でエラーが発生した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()
			expectedErr := testkit.ExpectedDBError()

			userDomain := newSearchTestUser(t, prefectureID, now)
			prefectureDomain, err := prefecture.New(prefectureID, "Tokyo", 1)
			require.NoError(t, err)

			ctrl := gomock.NewController(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().
				SearchByKeyword(gomock.Any(), keywords, active, p.Limit32(), p.Offset32()).
				Return(user.Users{userDomain}, nil)
			userRepo.EXPECT().
				CountByKeyword(gomock.Any(), keywords, active).
				Return(int64(0), expectedErr)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().
				FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).
				Return(prefecture.Prefectures{prefectureDomain}, nil)

			u := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			actual, err := u.ListUsersByKeywordWithTotal(ctx, filter, p)
			require.ErrorIs(t, err, expectedErr)
			require.Nil(t, actual)
		})

		t.Run("filter が nil の場合、ErrInvalidArgument が返る", func(t *testing.T) {
			t.Parallel()
			u := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase()}
			actual, err := u.ListUsersByKeywordWithTotal(ctx, nil, p)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})
	})
}

func Test_usecase_toSearchResults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	prefectureID := uuid.NewTestFromSalt(t, "to_search_results_prefecture")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全ユーザーの都道府県が解決できる場合、検索結果のリストが返る", func(t *testing.T) {
			t.Parallel()

			userDomain := newSearchTestUser(t, prefectureID, now)
			prefectureDomain, err := prefecture.New(prefectureID, "Tokyo", 1)
			require.NoError(t, err)

			ctrl := gomock.NewController(t)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().
				FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).
				Return(prefecture.Prefectures{prefectureDomain}, nil)

			u := &usecase{tracer: lt, pftRepo: pftRepo}

			actual, err := u.toSearchResults(ctx, user.Users{userDomain})
			require.NoError(t, err)

			expected := UserSearchResults{
				{
					FirstName:      userDomain.FirstName(),
					LastName:       userDomain.LastName(),
					Email:          userDomain.Email(),
					Phone:          userDomain.Phone(),
					PostalCode:     userDomain.PostalCode(),
					PrefectureName: prefectureDomain.Name(),
					City:           userDomain.City(),
					Street:         userDomain.Street(),
					Building:       userDomain.Building(),
					RegisteredAt:   userDomain.CreatedAt(),
					DeletedAt:      userDomain.DeletedAt(),
				},
			}
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("都道府県取得でエラーが発生した場合、エラーが伝播される", func(t *testing.T) {
			t.Parallel()
			expectedErr := testkit.ExpectedDBError()

			userDomain := newSearchTestUser(t, prefectureID, now)

			ctrl := gomock.NewController(t)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().
				FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).
				Return(nil, expectedErr)

			u := &usecase{tracer: lt, pftRepo: pftRepo}

			actual, err := u.toSearchResults(ctx, user.Users{userDomain})
			require.ErrorIs(t, err, expectedErr)
			require.Nil(t, actual)
		})

		t.Run("ユーザーが参照する都道府県が解決できない場合、参照整合性破れが返る", func(t *testing.T) {
			t.Parallel()

			userDomain := newSearchTestUser(t, prefectureID, now)

			ctrl := gomock.NewController(t)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().
				FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).
				Return(prefecture.Prefectures{}, nil)

			u := &usecase{tracer: lt, pftRepo: pftRepo}

			actual, err := u.toSearchResults(ctx, user.Users{userDomain})
			require.ErrorIs(t, err, errOrphanPrefecture)
			require.Nil(t, actual)
		})
	})
}
