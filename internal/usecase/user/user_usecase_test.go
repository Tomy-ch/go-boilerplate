package user

import (
	"context"
	"strings"
	"testing"
	"time"

	"boilerplate-go/internal/domain/prefecture"
	mock_prefecture "boilerplate-go/internal/domain/prefecture/mock"
	"boilerplate-go/internal/domain/user"
	mock_user "boilerplate-go/internal/domain/user/mock"
	"boilerplate-go/internal/observability"
	mock_clock "boilerplate-go/internal/usecase/boundary/clock/mock"
	mock_security "boilerplate-go/internal/usecase/boundary/security/mock"
	mock_tx "boilerplate-go/internal/usecase/boundary/tx/mock"
	"boilerplate-go/internal/usecase/testkit"
	"boilerplate-go/internal/usecase/tools/paging"
	mock_query "boilerplate-go/internal/usecase/user/query/mock"
	"boilerplate-go/pkg/ptr"
	"boilerplate-go/pkg/uuid"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	mockTxManager := mock_tx.NewMockManager(ctrl)
	clock := mock_clock.NewMockClock(ctrl)
	byencrypter := mock_security.NewMockBcrypter(ctrl)
	userRepo := mock_user.NewMockRepository(ctrl)
	pftRepo := mock_prefecture.NewMockRepository(ctrl)
	userQS := mock_query.NewMockUserQueryService(ctrl)

	expected := &usecase{
		tracer:      tf.Usecase(),
		txm:         mockTxManager,
		clock:       clock,
		byencrypter: byencrypter,
		userRepo:    userRepo,
		pftRepo:     pftRepo,
		userQS:      userQS,
	}
	actual := New(tf, mockTxManager, clock, byencrypter, userRepo, pftRepo, userQS)

	require.Equal(t, expected, actual)
}

func TestGetAllUsers(t *testing.T) {
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
			prefectureID.String(),
			"prefecture_name",
			1,
		)
		require.NoError(t, err)

		expected := []MutableFields{
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

		t.Run("paramsがない場合、全件取得が実行されユーザー情報がリストで取得できる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindAll(gomock.Any(), p.Limit32(), p.Offset32()).Return(user.Users{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Entities{prefectureDomain}, nil)
			uc := &usecase{
				tracer:   lt,
				userRepo: userRepo,
				pftRepo:  pftRepo,
			}

			actual, err := uc.ListUsersByKeyword(ctx, nil, p)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})

		t.Run("paramsがある場合、キーワード検索が実行されユーザー情報がリストで取得できる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			params := &GetParamsDTO{
				Keyword: ptr.To("first"),
				Active:  nil,
			}

			keywords := []string{*params.Keyword}

			userQS := mock_query.NewMockUserQueryService(ctrl)
			userQS.EXPECT().FindByKeyword(gomock.Any(), keywords, params.Active, p.Limit32(), p.Offset32()).Return(user.Users{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Entities{prefectureDomain}, nil)
			uc := &usecase{
				tracer:  lt,
				userQS:  userQS,
				pftRepo: pftRepo,
			}

			actual, err := uc.ListUsersByKeyword(ctx, params, p)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザー取得時にエラーが発生した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			expectedErr := testkit.ExpectedDBError(t)

			page := 1
			perPage := 100
			p, actualErr := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, actualErr)

			repo := mock_user.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any(), p.Limit32(), p.Offset32()).Return(nil, expectedErr)
			uc := &usecase{
				tracer:   lt,
				userRepo: repo,
			}

			actual, actualErr := uc.ListUsersByKeyword(ctx, nil, p)
			require.Nil(t, actual)
			require.ErrorIs(t, expectedErr, actualErr)
		})

		t.Run("都道府県取得時にエラーが発生した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()

			expectedErr := testkit.ExpectedDBError(t)

			page := 1
			perPage := 100
			p, actualErr := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, actualErr)

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindAll(gomock.Any(), p.Limit32(), p.Offset32()).Return(user.Users{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(nil, expectedErr)
			uc := &usecase{
				tracer:   lt,
				userRepo: userRepo,
				pftRepo:  pftRepo,
			}

			actual, err := uc.ListUsersByKeyword(ctx, nil, p)
			require.Nil(t, actual)
			require.ErrorIs(t, err, expectedErr)
		})
	})
}

func TestCreate(t *testing.T) {
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
		prefectureID.String(),
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
			expected := createDTO.MutableFields

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)
			byencrypter := mock_security.NewMockBcrypter(ctrl)
			byencrypter.EXPECT().Hash(createDTO.RawPassword).Return("hashed_password", nil)
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
				tracer:      lt,
				txm:         mockTxManager,
				clock:       clock,
				byencrypter: byencrypter,
				userRepo:    userRepo,
				pftRepo:     pftRepo,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
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
			require.Equal(t, MutableFields{}, actual)
			require.ErrorIs(t, err, user.ErrPassword)
		})

		t.Run("暗号化が失敗した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			expectedErr := xerrors.New("encryption failed")

			createDTO := newCreateDTO(userDomain, prefectureName)

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)
			byencrypter := mock_security.NewMockBcrypter(ctrl)
			byencrypter.EXPECT().Hash(createDTO.RawPassword).Return("", expectedErr)

			uc := &usecase{
				tracer:      lt,
				clock:       clock,
				byencrypter: byencrypter,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			require.Equal(t, MutableFields{}, actual)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("都道府県の取得に失敗した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			expectedErr := testkit.ExpectedDBError(t)

			createDTO := newCreateDTO(userDomain, prefectureName)

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)
			byencrypter := mock_security.NewMockBcrypter(ctrl)
			byencrypter.EXPECT().Hash(createDTO.RawPassword).Return("hashed_password", nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(
				gomock.Any(),
				prefectureName,
			).Return(nil, expectedErr)

			uc := &usecase{
				tracer:      lt,
				txm:         mockTxManager,
				clock:       clock,
				byencrypter: byencrypter,
				pftRepo:     pftRepo,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			require.Equal(t, MutableFields{}, actual)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("ユーザードメインの生成に失敗した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			createDTO := newCreateDTO(userDomain, prefectureName)
			createDTO.FirstName = "" // FirstNameを空にしてエラーを発生させる

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)
			byencrypter := mock_security.NewMockBcrypter(ctrl)
			byencrypter.EXPECT().Hash(createDTO.RawPassword).Return("hashed_password", nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(
				gomock.Any(),
				prefectureName,
			).Return(pftDomain, nil)

			uc := &usecase{
				tracer:      lt,
				txm:         mockTxManager,
				clock:       clock,
				byencrypter: byencrypter,
				pftRepo:     pftRepo,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			require.Equal(t, MutableFields{}, actual)
			require.ErrorIs(t, err, user.ErrInvalidFirstName)
		})

		t.Run("ユーザー作成に失敗した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			expectedErr := testkit.ExpectedDBError(t)

			createDTO := newCreateDTO(userDomain, prefectureName)

			clock := mock_clock.NewMockClock(ctrl)
			clock.EXPECT().Now().Return(now)
			byencrypter := mock_security.NewMockBcrypter(ctrl)
			byencrypter.EXPECT().Hash(createDTO.RawPassword).Return("hashed_password", nil)
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
				tracer:      lt,
				txm:         mockTxManager,
				clock:       clock,
				byencrypter: byencrypter,
				userRepo:    userRepo,
				pftRepo:     pftRepo,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			require.Equal(t, MutableFields{}, actual)
			require.ErrorIs(t, err, expectedErr)
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
		active := ptr.To(true)
		expectedCount := int64(42)

		userRepo.
			EXPECT().
			CountByActive(gomock.Any(), active).
			Return(expectedCount, nil)

		actualCount, err := u.CountUsers(ctx, active)
		require.NoError(t, err)
		require.Equal(t, expectedCount, actualCount)
	})
}

// newCreateDTO は、テスト用のCreateParamsDTOを生成するヘルパー関数です。
func newCreateDTO(u *user.User, pName string) *CreateParamsDTO {
	return &CreateParamsDTO{
		UserID:      u.ID(),
		RawPassword: "password",
		MutableFields: MutableFields{
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
