package user

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/prefecture"
	mock_prefecture "go-boilerplate/internal/domain/prefecture/mock"
	"go-boilerplate/internal/domain/user"
	mock_user "go-boilerplate/internal/domain/user/mock"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	mock_security "go-boilerplate/internal/usecase/boundary/security/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	mockTxManager := mock_tx.NewMockManager(ctrl)
	clock := mock_clock.NewMockClock(ctrl)
	encrypter := mock_security.NewMockHasher(ctrl)
	userRepo := mock_user.NewMockRepository(ctrl)
	pftRepo := mock_prefecture.NewMockRepository(ctrl)

	expected := &usecase{
		tracer:    tf.Usecase(),
		txm:       mockTxManager,
		clock:     clock,
		encrypter: encrypter,
		userRepo:  userRepo,
		pftRepo:   pftRepo,
	}
	actual := New(tf, mockTxManager, clock, encrypter, userRepo, pftRepo)

	assert.Equal(t, expected, actual)
}

func Test_usecase_ListUsers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)

	prefectureID := uuid.NewTestFromSalt(t, "prefecture_domain")

	userDomain, err := user.New(
		uuid.NewTestFromSalt(t, "user_domain"),
		"first_name",
		"last_name",
		"password",
		"email_address",
		"phone_number",
		prefectureID,
		"city_name",
		"town_address",
		nil,
		"p_code",
		now,
		now,
		nil,
	)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		page := 1
		perPage := 100
		p, err := paging.NewPagingFrom1Based(&page, &perPage)
		require.NoError(t, err)

		prefectureDomain, err := prefecture.New(
			prefectureID,
			"prefecture_name",
			1,
		)
		require.NoError(t, err)

		expected := []UserView{
			{
				FirstName:      userDomain.FirstName(),
				LastName:       userDomain.LastName(),
				PostalCode:     userDomain.PostalCode(),
				PrefectureName: prefectureDomain.Name(),
				City:           userDomain.City(),
				Street:         userDomain.Street(),
				Building:       userDomain.Building(),
				Email:          userDomain.Email(),
				Phone:          userDomain.Phone(),
			},
		}

		t.Run("activeがnilの場合、全件取得が実行されユーザー情報がリストで取得できる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByActive(gomock.Any(), nil, p.Limit32(), p.Offset32()).Return(user.Users{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Prefectures{prefectureDomain}, nil)
			uc := &usecase{
				tracer:   lt,
				userRepo: userRepo,
				pftRepo:  pftRepo,
			}

			actual, err := uc.ListUsers(ctx, nil, p)
			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザー取得時にエラーが発生した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			expectedErr := testkit.ExpectedDBError()

			page := 1
			perPage := 100
			p, actualErr := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, actualErr)

			repo := mock_user.NewMockRepository(ctrl)
			repo.EXPECT().FindByActive(gomock.Any(), nil, p.Limit32(), p.Offset32()).Return(nil, expectedErr)
			uc := &usecase{
				tracer:   lt,
				userRepo: repo,
			}

			actual, actualErr := uc.ListUsers(ctx, nil, p)
			require.Nil(t, actual)
			require.ErrorIs(t, expectedErr, actualErr)
		})

		t.Run("都道府県取得時にエラーが発生した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()

			expectedErr := testkit.ExpectedDBError()

			page := 1
			perPage := 100
			p, actualErr := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, actualErr)

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByActive(gomock.Any(), nil, p.Limit32(), p.Offset32()).Return(user.Users{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(nil, expectedErr)
			uc := &usecase{
				tracer:   lt,
				userRepo: userRepo,
				pftRepo:  pftRepo,
			}

			actual, err := uc.ListUsers(ctx, nil, p)
			require.Nil(t, actual)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("ユーザーの都道府県が解決できない場合、ErrInternal が返される", func(t *testing.T) {
			t.Parallel()

			page := 1
			perPage := 100
			p, err := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, err)

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByActive(gomock.Any(), nil, p.Limit32(), p.Offset32()).Return(user.Users{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Prefectures{}, nil)
			uc := &usecase{
				tracer:   lt,
				userRepo: userRepo,
				pftRepo:  pftRepo,
			}

			actual, err := uc.ListUsers(ctx, nil, p)
			require.Nil(t, actual)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("page が nil の場合、ErrInvalidArgument が返される", func(t *testing.T) {
			t.Parallel()

			uc := &usecase{tracer: lt}
			actual, err := uc.ListUsers(ctx, nil, nil)
			require.Nil(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})
	})
}

func Test_usecase_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	mockTxManager := testkit.NewMockTransactionManager(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)

	prefectureID := uuid.NewTestFromSalt(t, "prefecture_domain")

	userDomain, err := user.New(
		uuid.NewTestFromSalt(t, "user_domain"),
		"first_name",
		"last_name",
		"password",
		"email_address",
		"phone_number",
		prefectureID,
		"city_name",
		"town_address",
		nil,
		"p_code",
		now,
		now,
		nil,
	)
	require.NoError(t, err)

	prefectureName := "prefecture_name"

	pftDomain, err := prefecture.New(
		prefectureID,
		prefectureName,
		1,
	)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザー作成が成功した場合、作成したユーザー情報が返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			createDTO := newCreateDTO(userDomain, prefectureName)
			expected := UserView{
				FirstName:      createDTO.FirstName,
				LastName:       createDTO.LastName,
				Email:          createDTO.Email,
				Phone:          createDTO.Phone,
				PostalCode:     createDTO.PostalCode,
				PrefectureName: createDTO.PrefectureName,
				City:           createDTO.City,
				Street:         createDTO.Street,
				Building:       createDTO.Building,
			}

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Hash(createDTO.RawPassword).Return("hashed_password", nil)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().Create(
				gomock.Any(),
				gomock.AssignableToTypeOf(userDomain),
			).Return(nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(
				gomock.Any(),
				prefectureName,
			).Return(pftDomain, nil)

			uc := &usecase{
				tracer:    lt,
				txm:       mockTxManager,
				clock:     clock,
				encrypter: encrypter,
				userRepo:  userRepo,
				pftRepo:   pftRepo,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生パスワードの検証が失敗した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			createDTO := newCreateDTO(userDomain, prefectureName)
			createDTO.RawPassword = strings.Repeat("a", user.MaxRawPasswordLength+1) // パスワードを最大長+1にしてエラーを発生させる

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)

			uc := &usecase{
				tracer: lt,
				clock:  clock,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			assert.Equal(t, UserView{}, actual)
			require.ErrorIs(t, err, user.ErrInvalidRawPassword)
		})

		t.Run("暗号化が失敗した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			expectedErr := xerrors.New("encryption failed")

			createDTO := newCreateDTO(userDomain, prefectureName)

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Hash(createDTO.RawPassword).Return("", expectedErr)

			uc := &usecase{
				tracer:    lt,
				clock:     clock,
				encrypter: encrypter,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			assert.Equal(t, UserView{}, actual)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("都道府県の取得に失敗した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			expectedErr := testkit.ExpectedDBError()

			createDTO := newCreateDTO(userDomain, prefectureName)

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Hash(createDTO.RawPassword).Return("hashed_password", nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(
				gomock.Any(),
				prefectureName,
			).Return(nil, expectedErr)

			uc := &usecase{
				tracer:    lt,
				txm:       mockTxManager,
				clock:     clock,
				encrypter: encrypter,
				pftRepo:   pftRepo,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			assert.Equal(t, UserView{}, actual)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("ユーザードメインの生成に失敗した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			createDTO := newCreateDTO(userDomain, prefectureName)
			createDTO.FirstName = "" // FirstNameを空にしてエラーを発生させる

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Hash(createDTO.RawPassword).Return("hashed_password", nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(
				gomock.Any(),
				prefectureName,
			).Return(pftDomain, nil)

			uc := &usecase{
				tracer:    lt,
				txm:       mockTxManager,
				clock:     clock,
				encrypter: encrypter,
				pftRepo:   pftRepo,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			assert.Equal(t, UserView{}, actual)
			require.ErrorIs(t, err, user.ErrInvalidFirstName)
		})

		t.Run("ユーザー作成に失敗した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			expectedErr := testkit.ExpectedDBError()

			createDTO := newCreateDTO(userDomain, prefectureName)

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Hash(createDTO.RawPassword).Return("hashed_password", nil)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().Create(
				gomock.Any(),
				gomock.AssignableToTypeOf(userDomain),
			).Return(expectedErr)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(
				gomock.Any(),
				prefectureName,
			).Return(pftDomain, nil)

			uc := &usecase{
				tracer:    lt,
				txm:       mockTxManager,
				clock:     clock,
				encrypter: encrypter,
				userRepo:  userRepo,
				pftRepo:   pftRepo,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			assert.Equal(t, UserView{}, actual)
			require.ErrorIs(t, err, expectedErr)
		})
	})
}

func Test_usecase_ListUsersWithTotal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)

	prefectureID := uuid.NewTestFromSalt(t, "prefecture_domain")
	userDomain, err := user.New(
		uuid.NewTestFromSalt(t, "user_domain"),
		"first_name", "last_name", "password", "email_address", "phone_number",
		prefectureID, "city_name", "town_address", nil, "p_code", now, now, nil,
	)
	require.NoError(t, err)
	prefectureDomain, err := prefecture.New(prefectureID, "prefecture_name", 1)
	require.NoError(t, err)

	page := 1
	perPage := 100
	p, err := paging.NewPagingFrom1Based(&page, &perPage)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("一覧と総件数をまとめて取得できる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByActive(gomock.Any(), nil, p.Limit32(), p.Offset32()).Return(user.Users{userDomain}, nil)
			userRepo.EXPECT().CountByActive(gomock.Any(), nil).Return(int64(1), nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Prefectures{prefectureDomain}, nil)

			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			actual, err := uc.ListUsersWithTotal(ctx, nil, p)
			require.NoError(t, err)
			assert.Len(t, actual.Items, 1)
			assert.Equal(t, int64(1), actual.Total)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一覧取得でエラーが発生した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()
			expectedErr := testkit.ExpectedDBError()
			ctrl := gomock.NewController(t)

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByActive(gomock.Any(), nil, p.Limit32(), p.Offset32()).Return(nil, expectedErr)
			uc := &usecase{tracer: lt, userRepo: userRepo}

			actual, err := uc.ListUsersWithTotal(ctx, nil, p)
			require.ErrorIs(t, err, expectedErr)
			require.Nil(t, actual)
		})

		t.Run("総件数取得でエラーが発生した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()
			expectedErr := testkit.ExpectedDBError()
			ctrl := gomock.NewController(t)

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByActive(gomock.Any(), nil, p.Limit32(), p.Offset32()).Return(user.Users{userDomain}, nil)
			userRepo.EXPECT().CountByActive(gomock.Any(), nil).Return(int64(0), expectedErr)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Prefectures{prefectureDomain}, nil)
			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			actual, err := uc.ListUsersWithTotal(ctx, nil, p)
			require.ErrorIs(t, err, expectedErr)
			require.Nil(t, actual)
		})

		t.Run("page が nil の場合、ErrInvalidArgument が返る", func(t *testing.T) {
			t.Parallel()
			uc := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase()}
			actual, err := uc.ListUsersWithTotal(ctx, nil, nil)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})
	})
}

func Test_usecase_CountUsers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	userRepo := mock_user.NewMockRepository(ctrl)

	u := &usecase{
		tracer:   tf.Usecase(),
		userRepo: userRepo,
	}

	t.Run("ユーザーの総件数が正常に取得できること", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		active := new(true)
		expectedCount := int64(42)

		userRepo.
			EXPECT().
			CountByActive(gomock.Any(), active).
			Return(expectedCount, nil)

		actualCount, err := u.CountUsers(ctx, active)
		require.NoError(t, err)
		assert.Equal(t, expectedCount, actualCount)
	})
}

func Test_usecase_ListUsersFeed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)

	prefectureID := uuid.NewTestFromSalt(t, "prefecture_domain")
	prefectureDomain, err := prefecture.New(prefectureID, "prefecture_name", 1)
	require.NoError(t, err)

	// newFeedUser は、作成日時違いのフィードユーザーを生成するヘルパーです。
	newFeedUser := func(salt string, createdAt time.Time) *user.User {
		u, uErr := user.New(
			uuid.NewTestFromSalt(t, salt),
			"first_name", "last_name", "password", "email_address", "phone_number",
			prefectureID, "city_name", "town_address", nil, "p_code", createdAt, createdAt, nil,
		)
		require.NoError(t, uErr)
		return u
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭ページの場合、afterがnilでリポジトリが呼ばれ次ページが無ければNextCursorはnilになる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			first := 20
			cursor, cErr := paging.NewCursor(nil, &first)
			require.NoError(t, cErr)

			u1 := newFeedUser("feed_user_1", now)

			userRepo := mock_user.NewMockRepository(ctrl)
			// limit+1（=21）件を要求し、先頭ページは after=nil で呼ばれる。
			userRepo.EXPECT().FindFeed(gomock.Any(), (*user.FeedCursor)(nil), int32(21)).Return(user.Users{u1}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Prefectures{prefectureDomain}, nil)

			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			actual, actualErr := uc.ListUsersFeed(ctx, cursor)
			require.NoError(t, actualErr)
			assert.Len(t, actual.Items, 1)
			assert.Nil(t, actual.NextCursor)
		})

		t.Run("カーソル指定の場合、afterが解釈されてリポジトリが呼ばれる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			afterUser := newFeedUser("feed_user_after", now)
			encoded := paging.EncodeCursor(afterUser.CreatedAt().Format(time.RFC3339Nano), afterUser.ID().String())
			first := 20
			cursor, cErr := paging.NewCursor(&encoded, &first)
			require.NoError(t, cErr)

			u1 := newFeedUser("feed_user_1", now)

			// カーソルの復号正当性は feed_cursor_test.go が担保するため、ここでは
			// 「after 指定時に limit+1 件でリポジトリが呼ばれる」オーケストレーションのみ検証する。
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindFeed(
				gomock.Any(),
				gomock.Any(),
				int32(21),
			).Return(user.Users{u1}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Prefectures{prefectureDomain}, nil)

			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			actual, actualErr := uc.ListUsersFeed(ctx, cursor)
			require.NoError(t, actualErr)
			assert.Len(t, actual.Items, 1)
			assert.Nil(t, actual.NextCursor)
		})

		t.Run("limit+1件取得できた場合、limit件に切り詰められ末尾行からNextCursorが生成される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			first := 1
			cursor, cErr := paging.NewCursor(nil, &first)
			require.NoError(t, cErr)

			// limit=1 に対し 2 件返す。先頭が表示分、2件目は次ページ存在判定用。
			head := newFeedUser("feed_user_head", now)
			tail := newFeedUser("feed_user_tail", now.Add(-time.Hour))

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindFeed(gomock.Any(), (*user.FeedCursor)(nil), int32(2)).Return(user.Users{head, tail}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			// 切り詰め後の 1 件分のみ都道府県解決される。
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Prefectures{prefectureDomain}, nil)

			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			actual, actualErr := uc.ListUsersFeed(ctx, cursor)
			require.NoError(t, actualErr)
			assert.Len(t, actual.Items, 1)
			require.NotNil(t, actual.NextCursor)
			// NextCursor は切り詰め後の末尾（head）のソートキーから生成される。
			expectedCursor := paging.EncodeCursor(head.CreatedAt().Format(time.RFC3339Nano), head.ID().String())
			assert.Equal(t, expectedCursor, *actual.NextCursor)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cursorがnilの場合、ErrInvalidArgumentが返る", func(t *testing.T) {
			t.Parallel()

			uc := &usecase{tracer: lt}
			actual, actualErr := uc.ListUsersFeed(ctx, nil)
			require.ErrorIs(t, actualErr, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})

		t.Run("カーソルキーが2個でない場合、ErrInvalidArgumentが返る", func(t *testing.T) {
			t.Parallel()

			// キー1個の不正カーソル。
			encoded := paging.EncodeCursor("only_one_key")
			first := 20
			cursor, cErr := paging.NewCursor(&encoded, &first)
			require.NoError(t, cErr)

			uc := &usecase{tracer: lt}
			actual, actualErr := uc.ListUsersFeed(ctx, cursor)
			require.ErrorIs(t, actualErr, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})

		t.Run("カーソルのcreated_atがRFC3339Nanoでない場合、ErrInvalidArgumentが返る", func(t *testing.T) {
			t.Parallel()

			encoded := paging.EncodeCursor("not-a-time", uuid.NewTestFromSalt(t, "any").String())
			first := 20
			cursor, cErr := paging.NewCursor(&encoded, &first)
			require.NoError(t, cErr)

			uc := &usecase{tracer: lt}
			actual, actualErr := uc.ListUsersFeed(ctx, cursor)
			require.ErrorIs(t, actualErr, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})

		t.Run("カーソルのidがUUIDでない場合、ErrInvalidArgumentが返る", func(t *testing.T) {
			t.Parallel()

			encoded := paging.EncodeCursor(now.Format(time.RFC3339Nano), "not-a-uuid")
			first := 20
			cursor, cErr := paging.NewCursor(&encoded, &first)
			require.NoError(t, cErr)

			uc := &usecase{tracer: lt}
			actual, actualErr := uc.ListUsersFeed(ctx, cursor)
			require.ErrorIs(t, actualErr, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})

		t.Run("リポジトリ取得でエラーが発生した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			expectedErr := testkit.ExpectedDBError()
			first := 20
			cursor, cErr := paging.NewCursor(nil, &first)
			require.NoError(t, cErr)

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindFeed(gomock.Any(), (*user.FeedCursor)(nil), int32(21)).Return(nil, expectedErr)
			uc := &usecase{tracer: lt, userRepo: userRepo}

			actual, actualErr := uc.ListUsersFeed(ctx, cursor)
			require.ErrorIs(t, actualErr, expectedErr)
			require.Nil(t, actual)
		})

		t.Run("ユーザーの都道府県が解決できない場合、ErrInternalが返る", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			first := 20
			cursor, cErr := paging.NewCursor(nil, &first)
			require.NoError(t, cErr)

			u1 := newFeedUser("feed_user_1", now)

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindFeed(gomock.Any(), (*user.FeedCursor)(nil), int32(21)).Return(user.Users{u1}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Prefectures{}, nil)
			uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}

			actual, actualErr := uc.ListUsersFeed(ctx, cursor)
			require.ErrorIs(t, actualErr, apperror.ErrInternal)
			require.Nil(t, actual)
		})
	})
}

// newCreateDTO は、テスト用のCreateParamsDTOを生成するヘルパー関数です。
func newCreateDTO(u *user.User, pName string) *CreateParamsDTO {
	return &CreateParamsDTO{
		UserID:      u.ID(),
		RawPassword: "password",
		UpdateProfileParams: UpdateProfileParams{
			FirstName:      u.FirstName(),
			LastName:       u.LastName(),
			Email:          u.Email(),
			Phone:          u.Phone(),
			PostalCode:     u.PostalCode(),
			PrefectureName: pName,
			City:           u.City(),
			Street:         u.Street(),
			Building:       u.Building(),
		},
	}
}
