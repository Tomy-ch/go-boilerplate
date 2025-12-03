package user

import (
	"testing"

	"boilerplate-go/internal/domain/user"
	mock_user "boilerplate-go/internal/domain/user/mock"
	"boilerplate-go/internal/usecase/paging"
	"boilerplate-go/internal/usecase/usecasetest"
	"boilerplate-go/pkg/uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl, tf := usecasetest.NewTestInstanceForNew(t)
	repo := mock_user.NewMockRepository(ctrl)

	expected := &usecase{
		tracer:   tf.Usecase(),
		userRepo: repo,
	}
	actual := New(repo, tf)

	require.Equal(t, expected, actual)
}

func TestGetAllUsers(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単一のユーザーが存在する場合、単一のユーザー情報がリストで取得できる", func(t *testing.T) {
			t.Parallel()

			page := 1
			perPage := 100
			p, err := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, err)

			domain, err := user.New(
				uuid.NewTestFromSalt(t, "user_domain").String(),
				"first_name",
				"last_name",
				"password",
				"email_address",
				"phone_number",
				uuid.NewTestFromSalt(t, "prefecture_domain").String(),
				"prefecture_name",
				"city_name",
				"town_address",
				nil,
				"p_code",
				nil,
			)
			require.NoError(t, err)

			ctx, ctrl, _, lt := usecasetest.NewTestInstanceForImplementedUsecase(t)
			repo := mock_user.NewMockRepository(ctrl)
			repo.EXPECT().GetAllUsers(gomock.Any(), p.Limit(), p.Offset()).Return(user.Entities{*domain}, nil)
			uc := &usecase{
				tracer:   lt,
				userRepo: repo,
			}

			expected := []DTO{
				{
					Name:  domain.FullName(),
					Email: domain.Email(),
					Phone: domain.Phone(),
				},
			}
			actual, err := uc.GetAllUsers(ctx, p)
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

			ctx, ctrl, _, lt := usecasetest.NewTestInstanceForImplementedUsecase(t)

			repo := mock_user.NewMockRepository(ctrl)
			repo.EXPECT().GetAllUsers(gomock.Any(), p.Limit(), p.Offset()).Return(nil, expectedErr)
			uc := &usecase{
				tracer:   lt,
				userRepo: repo,
			}

			actual, actualErr := uc.GetAllUsers(ctx, p)
			require.Nil(t, actual)
			require.ErrorIs(t, expectedErr, actualErr)
		})
	})
}
