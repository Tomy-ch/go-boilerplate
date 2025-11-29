package user

import (
	"context"
	"errors"
	"testing"

	"boilerplate-go/internal/domain/user"
	mock_user "boilerplate-go/internal/domain/user/mock"
	"boilerplate-go/internal/usecase/paging"
	"boilerplate-go/pkg/uuid"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := mock_user.NewMockRepository(ctrl)

	expected := &usecase{
		userRepo: repo,
	}
	actual := New(repo)

	require.Equal(t, expected, actual)
}

func TestGetAllUsers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

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

			ctrl := gomock.NewController(t)
			repo := mock_user.NewMockRepository(ctrl)
			repo.EXPECT().GetAllUsers(ctx, p.Limit(), p.Offset()).Return(user.Entities{*domain}, nil)
			uc := New(repo)

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

			page := 1
			perPage := 100
			p, err := paging.NewPagingFrom1Based(&page, &perPage)
			require.NoError(t, err)

			ctrl := gomock.NewController(t)
			repo := mock_user.NewMockRepository(ctrl)
			repo.EXPECT().GetAllUsers(ctx, p.Limit(), p.Offset()).Return(nil, errors.New("error"))
			uc := New(repo)

			actual, err := uc.GetAllUsers(ctx, p)
			require.Error(t, err)
			require.Nil(t, actual)
		})
	})
}
