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
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newAllowAuthorizer は、Authorize が常に許可（nil）を返す Authorizer モックを返します。
func newAllowAuthorizer(ctrl *gomock.Controller) *mock_authz.MockAuthorizer {
	a := mock_authz.NewMockAuthorizer(ctrl)
	a.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	return a
}

// newDenyAuthorizer は、Authorize が常に拒否（ErrForbidden）を返す Authorizer モックを返します。
func newDenyAuthorizer(ctrl *gomock.Controller) *mock_authz.MockAuthorizer {
	a := mock_authz.NewMockAuthorizer(ctrl)
	a.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(authz.ErrForbidden)
	return a
}

// newTestAuthn は、テスト用の認証主体（Authn）を返します。
func newTestAuthn(t *testing.T) *authbd.Authn {
	t.Helper()
	authn, err := authbd.New(uuidtestkit.NewTestFromSalt(t, "caller").String(), authbd.IssuerMock, nil, nil)
	require.NoError(t, err)
	return authn
}

// newSearchTestUser は、検索テスト用のユーザーエンティティを生成するヘルパーです。
func newSearchTestUser(t *testing.T, prefectureID uuid.UUID, createdAt time.Time) *user.User {
	t.Helper()
	u, err := user.New(uuidtestkit.NewTestFromSalt(t, "search_user_domain"), user.Attributes{
		Profile: user.Profile{
			FirstName:    "Grace",
			LastName:     "Lee",
			Email:        "grace.lee@example.com",
			Phone:        "090-1234-5678",
			PrefectureID: prefectureID,
			City:         "city_name",
			Street:       "town_address",
			PostalCode:   "150-0001",
		},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	})
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
		authorizer := mock_authz.NewMockAuthorizer(ctrl)

		expected := &usecase{
			tracer:     tf.Usecase(),
			userRepo:   userRepo,
			pftRepo:    pftRepo,
			authorizer: authorizer,
		}
		actual := New(tf, userRepo, pftRepo, authorizer)

		assert.Equal(t, expected, actual)
	})
}

func Test_usecase_ListUsersByKeyword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	prefectureID := uuidtestkit.NewTestFromSalt(t, "prefecture_domain")
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

			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo, authorizer: newAllowAuthorizer(gomock.NewController(t))}

			filter := &SearchParams{Keyword: new(keyword), Active: active}
			actual, err := uc.ListUsersByKeyword(ctx, newTestAuthn(t), filter, p)
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

			uc := &usecase{tracer: lt, userRepo: userRepo, authorizer: newAllowAuthorizer(gomock.NewController(t))}

			filter := &SearchParams{Keyword: new(keyword), Active: active}
			actual, err := uc.ListUsersByKeyword(ctx, newTestAuthn(t), filter, p)
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

			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo, authorizer: newAllowAuthorizer(gomock.NewController(t))}

			filter := &SearchParams{Keyword: new(keyword), Active: active}
			actual, err := uc.ListUsersByKeyword(ctx, newTestAuthn(t), filter, p)
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

			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo, authorizer: newAllowAuthorizer(gomock.NewController(t))}

			filter := &SearchParams{Keyword: new(keyword), Active: active}
			actual, err := uc.ListUsersByKeyword(ctx, newTestAuthn(t), filter, p)
			require.ErrorIs(t, err, apperror.ErrInternal)
			require.Nil(t, actual)
		})

		t.Run("filter が nil の場合、ErrInvalidArgument エラーになる", func(t *testing.T) {
			t.Parallel()

			page := 1
			perPage := 100
			p, err := paging.NewPageFrom1Based(&page, &perPage)
			require.NoError(t, err)

			uc := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase(), authorizer: newAllowAuthorizer(gomock.NewController(t))}
			actual, err := uc.ListUsersByKeyword(ctx, newTestAuthn(t), nil, p)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})

		t.Run("page が nil の場合、ErrInvalidArgument エラーになる", func(t *testing.T) {
			t.Parallel()

			uc := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase(), authorizer: newAllowAuthorizer(gomock.NewController(t))}
			actual, err := uc.ListUsersByKeyword(ctx, newTestAuthn(t), &SearchParams{}, nil)
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

			u := &usecase{tracer: tf.Usecase(), userRepo: userRepo, authorizer: newAllowAuthorizer(gomock.NewController(t))}

			filter := &SearchParams{Active: active, Keyword: new(keyword)}
			actualCount, err := u.CountUsersByKeyword(ctx, newTestAuthn(t), filter)
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

			u := &usecase{tracer: tf.Usecase(), userRepo: userRepo, authorizer: newAllowAuthorizer(gomock.NewController(t))}

			filter := &SearchParams{Active: active, Keyword: new(keyword)}
			actualCount, err := u.CountUsersByKeyword(ctx, newTestAuthn(t), filter)
			require.ErrorIs(t, err, expectedErr)
			assert.Equal(t, int64(0), actualCount)
		})

		t.Run("filter が nil の場合、ErrInvalidArgument エラーになる", func(t *testing.T) {
			t.Parallel()

			u := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase(), authorizer: newAllowAuthorizer(gomock.NewController(t))}
			actualCount, err := u.CountUsersByKeyword(context.Background(), newTestAuthn(t), nil)
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

	prefectureID := uuidtestkit.NewTestFromSalt(t, "prefecture_domain")
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

			u := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo, authorizer: newAllowAuthorizer(gomock.NewController(t))}

			actual, err := u.ListUsersByKeywordWithTotal(ctx, newTestAuthn(t), filter, p)
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

			u := &usecase{tracer: lt, userRepo: userRepo, authorizer: newAllowAuthorizer(gomock.NewController(t))}

			actual, err := u.ListUsersByKeywordWithTotal(ctx, newTestAuthn(t), filter, p)
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

			u := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo, authorizer: newAllowAuthorizer(gomock.NewController(t))}

			actual, err := u.ListUsersByKeywordWithTotal(ctx, newTestAuthn(t), filter, p)
			require.ErrorIs(t, err, expectedErr)
			require.Nil(t, actual)
		})

		t.Run("filter が nil の場合、ErrInvalidArgument が返る", func(t *testing.T) {
			t.Parallel()
			u := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase(), authorizer: newAllowAuthorizer(gomock.NewController(t))}
			actual, err := u.ListUsersByKeywordWithTotal(ctx, newTestAuthn(t), nil, p)
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

	prefectureID := uuidtestkit.NewTestFromSalt(t, "to_search_results_prefecture")

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

func Test_toSearchResult(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	building := "building_name"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エンティティの属性と引数の都道府県名から検索結果DTOを構築する", func(t *testing.T) {
			t.Parallel()

			u, err := user.New(uuidtestkit.NewTestFromSalt(t, "to_search_result_user"), user.Attributes{
				Profile: user.Profile{
					FirstName:    "Grace",
					LastName:     "Lee",
					Email:        "grace.lee@example.com",
					Phone:        "090-1234-5678",
					PrefectureID: uuidtestkit.NewTestFromSalt(t, "to_search_result_prefecture"),
					City:         "city_name",
					Street:       "town_address",
					Building:     &building,
					PostalCode:   "150-0001",
				},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			})
			require.NoError(t, err)

			actual := toSearchResult(u, "prefecture_name")
			require.NotNil(t, actual)

			assert.Equal(t, "Grace", actual.FirstName)
			assert.Equal(t, "Lee", actual.LastName)
			assert.Equal(t, "grace.lee@example.com", actual.Email)
			assert.Equal(t, "090-1234-5678", actual.Phone)
			assert.Equal(t, "150-0001", actual.PostalCode)
			assert.Equal(t, "prefecture_name", actual.PrefectureName)
			assert.Equal(t, "city_name", actual.City)
			assert.Equal(t, "town_address", actual.Street)
			require.NotNil(t, actual.Building)
			assert.Equal(t, building, *actual.Building)
			assert.Equal(t, createdAt, actual.RegisteredAt)
			assert.Nil(t, actual.DeletedAt)
		})

		t.Run("削除済みエンティティの場合、DeletedAtを引き継ぐ", func(t *testing.T) {
			t.Parallel()

			deletedAt := updatedAt.Add(time.Hour)
			u, err := user.New(uuidtestkit.NewTestFromSalt(t, "to_search_result_deleted"), user.Attributes{
				Profile: user.Profile{
					FirstName:    "Grace",
					LastName:     "Lee",
					Email:        "deleted@example.com",
					Phone:        "090-1234-5678",
					PrefectureID: uuidtestkit.NewTestFromSalt(t, "to_search_result_prefecture"),
					City:         "city_name",
					Street:       "town_address",
					PostalCode:   "150-0001",
				},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				DeletedAt: &deletedAt,
			})
			require.NoError(t, err)

			actual := toSearchResult(u, "prefecture_name")
			require.NotNil(t, actual)

			assert.Nil(t, actual.Building)
			require.NotNil(t, actual.DeletedAt)
			assert.Equal(t, deletedAt, *actual.DeletedAt)
		})
	})
}

func Test_usecase_authorizeUserCollection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	authn, err := authbd.New("subject", "mock", nil, nil)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("所有者を持たないリソースとして問い合わせ、許可された場合はnilが返る", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			authorizer.EXPECT().
				Authorize(gomock.Any(), authn, authz.ActionUserList, authz.NewResource("user", nil)).
				Return(nil)

			u := &usecase{authorizer: authorizer}

			require.NoError(t, u.authorizeUserCollection(ctx, authn))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("authnがnilの場合、ErrUnauthenticatedが返り認可判定は行われない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			u := &usecase{authorizer: mock_authz.NewMockAuthorizer(ctrl)}

			require.ErrorIs(t, u.authorizeUserCollection(ctx, nil), apperror.ErrUnauthenticated)
		})

		t.Run("認可が拒否された場合、認可エラーが伝播される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			u := &usecase{authorizer: newDenyAuthorizer(ctrl)}

			require.ErrorIs(t, u.authorizeUserCollection(ctx, authn), authz.ErrForbidden)
		})
	})
}

func Test_usecase_collectionAuthorizationGuard(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)

	page := 1
	perPage := 100
	p, err := paging.NewPageFrom1Based(&page, &perPage)
	require.NoError(t, err)

	filter := &SearchParams{}

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// userRepo は EXPECT を持たないため、認可より先にリポジトリを呼ぶ実装に戻ると失敗する。
		newDeniedUsecase := func(t *testing.T) *usecase {
			t.Helper()
			ctrl := gomock.NewController(t)
			return &usecase{
				tracer:     lt,
				authorizer: newDenyAuthorizer(ctrl),
				userRepo:   mock_user.NewMockRepository(ctrl),
				pftRepo:    mock_prefecture.NewMockRepository(ctrl),
			}
		}

		t.Run("ListUsersByKeywordがForbiddenを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := newDeniedUsecase(t).ListUsersByKeyword(ctx, newTestAuthn(t), filter, p)
			require.Nil(t, actual)
			require.ErrorIs(t, err, authz.ErrForbidden)
		})

		t.Run("CountUsersByKeywordがForbiddenを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := newDeniedUsecase(t).CountUsersByKeyword(ctx, newTestAuthn(t), filter)
			assert.Equal(t, int64(0), actual)
			require.ErrorIs(t, err, authz.ErrForbidden)
		})

		t.Run("ListUsersByKeywordWithTotalがForbiddenを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := newDeniedUsecase(t).ListUsersByKeywordWithTotal(ctx, newTestAuthn(t), filter, p)
			require.Nil(t, actual)
			require.ErrorIs(t, err, authz.ErrForbidden)
		})
	})
}
