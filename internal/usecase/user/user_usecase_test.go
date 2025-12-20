package user

import (
	"context"
	"testing"

	"boilerplate-go/internal/domain/prefecture"
	mock_prefecture "boilerplate-go/internal/domain/prefecture/mock"
	"boilerplate-go/internal/domain/user"
	mock_user "boilerplate-go/internal/domain/user/mock"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/support/paging"
	mock_tx "boilerplate-go/internal/usecase/tx/mock"
	"boilerplate-go/internal/usecase/usecasetest"
	"boilerplate-go/pkg/ptr"
	"boilerplate-go/pkg/uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	mockTxManager := mock_tx.NewMockManager(ctrl)
	userRepo := mock_user.NewMockRepository(ctrl)
	pftRepo := mock_prefecture.NewMockRepository(ctrl)

	expected := &usecase{
		tracer:   tf.Usecase(),
		txm:      mockTxManager,
		userRepo: userRepo,
		pftRepo:  pftRepo,
	}
	actual := New(tf, mockTxManager, userRepo, pftRepo)

	require.Equal(t, expected, actual)
}

func TestGetAllUsers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	lt := observability.NewMockUsecaseLayerTracer(t)

	prefectureID := uuid.NewTestFromSalt(t, "prefecture_domain")

	userDomain, err := user.New(
		uuid.NewTestFromSalt(t, "user_domain").String(),
		"first_name",
		"last_name",
		"password",
		"email_address",
		"phone_number",
		prefectureID.String(),
		"city_name",
		"town_address",
		nil,
		"p_code",
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

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindAll(gomock.Any(), p.Limit(), p.Offset()).Return(user.Entities{userDomain}, nil)
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

			params := &GetParamsDTO{
				Keyword: ptr.To("first"),
				Active:  nil,
			}

			keywords := []string{*params.Keyword}

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByKeyword(gomock.Any(), keywords, params.Active, p.Limit(), p.Offset()).Return(user.Entities{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(prefecture.Entities{prefectureDomain}, nil)
			uc := &usecase{
				tracer:   lt,
				userRepo: userRepo,
				pftRepo:  pftRepo,
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

			expectedErr := usecasetest.ExpectedDBError(t)

			page := 1
			perPage := 100
			p, actualErr := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, actualErr)

			repo := mock_user.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any(), p.Limit(), p.Offset()).Return(nil, expectedErr)
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

			expectedErr := usecasetest.ExpectedDBError(t)

			page := 1
			perPage := 100
			p, actualErr := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, actualErr)

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindAll(gomock.Any(), p.Limit(), p.Offset()).Return(user.Entities{userDomain}, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{prefectureID}).Return(nil, expectedErr)
			uc := &usecase{
				tracer:   lt,
				userRepo: userRepo,
				pftRepo:  pftRepo,
			}

			actual, err := uc.ListUsersByKeyword(ctx, nil, p)
			require.ErrorIs(t, expectedErr, err)
			require.Nil(t, actual)
		})
	})
}

func TestCreateUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	lt := observability.NewMockUsecaseLayerTracer(t)
	mockTxManager := usecasetest.NewMockTransactionManager(t)

	prefectureID := uuid.NewTestFromSalt(t, "prefecture_domain")

	userDomain, err := user.New(
		uuid.NewTestFromSalt(t, "user_domain").String(),
		"first_name",
		"last_name",
		"password",
		"email_address",
		"phone_number",
		prefectureID.String(),
		"city_name",
		"town_address",
		nil,
		"p_code",
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

			expected := MutableFields{
				FirstName:      userDomain.FirstName(),
				LastName:       userDomain.LastName(),
				Email:          userDomain.Email(),
				Phone:          userDomain.Phone(),
				PostalCode:     userDomain.PostalCode(),
				PrefectureName: prefectureName,
				City:           userDomain.City(),
				Street:         userDomain.Street(),
				Building:       userDomain.Building(),
			}

			createDTO := &CreateParamsDTO{}
			createDTO.UserID = userDomain.ID()
			createDTO.FirstName = userDomain.FirstName()
			createDTO.LastName = userDomain.LastName()
			createDTO.Password = userDomain.Password()
			createDTO.Email = userDomain.Email()
			createDTO.Phone = userDomain.Phone()
			createDTO.PostalCode = userDomain.PostalCode()
			createDTO.PrefectureName = prefectureName
			createDTO.City = userDomain.City()
			createDTO.Street = userDomain.Street()
			createDTO.Building = userDomain.Building()

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any(), userDomain).Return(nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(gomock.Any(), createDTO.PrefectureName).Return(pftDomain, nil)

			uc := &usecase{
				tracer:   lt,
				txm:      mockTxManager,
				userRepo: userRepo,
				pftRepo:  pftRepo,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("都道府県の取得に失敗した場合、エラーが返される", func(t *testing.T) {
			t.Parallel()

			expectedErr := usecasetest.ExpectedDBError(t)

			createDTO := &CreateParamsDTO{}
			createDTO.PrefectureName = prefectureName

			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(gomock.Any(), createDTO.PrefectureName).Return(nil, expectedErr)

			uc := &usecase{
				tracer:  lt,
				txm:     mockTxManager,
				pftRepo: pftRepo,
			}

			actual, err := uc.CreateUser(ctx, createDTO)
			require.Equal(t, MutableFields{}, actual)
			require.ErrorIs(t, expectedErr, err)
		})
	})

	t.Run("ユーザードメインの生成に失敗した場合、エラーが返される", func(t *testing.T) {
		t.Parallel()

		createDTO := &CreateParamsDTO{}
		createDTO.UserID = userDomain.ID()
		createDTO.FirstName = "" // FirstNameを空にしてエラーを発生させる
		createDTO.LastName = userDomain.LastName()
		createDTO.Password = userDomain.Password()
		createDTO.Email = userDomain.Email()
		createDTO.Phone = userDomain.Phone()
		createDTO.PostalCode = userDomain.PostalCode()
		createDTO.PrefectureName = prefectureName
		createDTO.City = userDomain.City()
		createDTO.Street = userDomain.Street()
		createDTO.Building = userDomain.Building()

		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByName(gomock.Any(), createDTO.PrefectureName).Return(pftDomain, nil)

		uc := &usecase{
			tracer:  lt,
			txm:     mockTxManager,
			pftRepo: pftRepo,
		}

		actual, err := uc.CreateUser(ctx, createDTO)
		require.Equal(t, MutableFields{}, actual)
		require.ErrorIs(t, err, user.ErrInvalidFirstName)
	})

	t.Run("ユーザー作成に失敗した場合、エラーが返される", func(t *testing.T) {
		t.Parallel()

		expectedErr := usecasetest.ExpectedDBError(t)

		createDTO := &CreateParamsDTO{}
		createDTO.UserID = userDomain.ID()
		createDTO.FirstName = userDomain.FirstName()
		createDTO.LastName = userDomain.LastName()
		createDTO.Password = userDomain.Password()
		createDTO.Email = userDomain.Email()
		createDTO.Phone = userDomain.Phone()
		createDTO.PostalCode = userDomain.PostalCode()
		createDTO.PrefectureName = prefectureName
		createDTO.City = userDomain.City()
		createDTO.Street = userDomain.Street()
		createDTO.Building = userDomain.Building()

		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().CreateUser(gomock.Any(), gomock.Any(), userDomain).Return(expectedErr)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByName(gomock.Any(), createDTO.PrefectureName).Return(pftDomain, nil)

		uc := &usecase{
			tracer:   lt,
			txm:      mockTxManager,
			userRepo: userRepo,
			pftRepo:  pftRepo,
		}

		actual, err := uc.CreateUser(ctx, createDTO)
		require.Equal(t, MutableFields{}, actual)
		require.ErrorIs(t, expectedErr, err)
	})
}
