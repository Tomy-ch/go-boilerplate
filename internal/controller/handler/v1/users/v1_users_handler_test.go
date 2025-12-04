package users

import (
	"net/http"
	"testing"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/controller/handler/handlertest/testassert"
	"boilerplate-go/internal/controller/handler/handlertest/testinstance"
	"boilerplate-go/internal/controller/handler/v1/users/gen"
	"boilerplate-go/internal/usecase/paging"
	"boilerplate-go/internal/usecase/user"
	mock_user "boilerplate-go/internal/usecase/user/mock"
	"boilerplate-go/pkg/ptr"

	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/users"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e, ctrl, tf, _ := testinstance.NewTestInstanceForBindHandler(t)
	mockApp := mock_user.NewMockUsecase(ctrl)

	BindHandler(e, tf, mockApp)

	expectedMethods := []string{
		http.MethodGet,
		http.MethodPost,
	}

	testassert.AssertEchoRouterPath(
		t, targetPath, e.Routes(),
	)
	testassert.AssertEchoRouterMethods(
		t, expectedMethods, e.Routes(),
	)
}

func Test_server_GetUsers(t *testing.T) {
	t.Parallel()

	expectedPage := 1
	expectedPerPage := 10

	expectedDTO1 := user.DTO{Name: "User1", Email: "user1@example.com", Phone: "1234567890"}
	expectedDTO2 := user.DTO{Name: "User2", Email: "user2@example.com", Phone: "0987654321"}

	mockPaging, err := paging.NewPagingFrom1Based(ptr.To(expectedPage), ptr.To(expectedPerPage))
	require.NoError(t, err)

	mockParams := gen.GetUsersRequestObject{
		Params: gen.GetUsersParams{
			Page:    ptr.To(expectedPage),
			PerPage: ptr.To(expectedPerPage),
		},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数のユーザーが存在する場合、ユーザー情報のリストが取得できる", func(t *testing.T) {
			t.Parallel()
			ctx, ctrl, _, lt := testinstance.NewTestInstancesForImplementedUsecase(t)

			expectedResponse := gen.ResponseV1Users{
				Users: []gen.UserResponse{
					{
						Name:  expectedDTO1.Name,
						Email: types.Email(expectedDTO1.Email),
						Phone: ptr.To(expectedDTO1.Phone),
					},
					{
						Name:  expectedDTO2.Name,
						Email: types.Email(expectedDTO2.Email),
						Phone: ptr.To(expectedDTO2.Phone),
					},
				},
				Limit:  mockPaging.Limit(),
				Offset: mockPaging.Offset(),
			}

			mockDTO := []user.DTO{expectedDTO1, expectedDTO2}
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				GetAllUsers(gomock.Any(), mockPaging).
				Return(mockDTO, nil)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsers(ctx, mockParams)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetUsers200JSONResponse)
			require.True(t, ok)

			require.Equal(t, expectedResponse, gen.ResponseV1Users(actual))
		})

		t.Run("単一のユーザーが存在する場合、ユーザー情報のリストが取得できる", func(t *testing.T) {
			t.Parallel()
			ctx, ctrl, _, lt := testinstance.NewTestInstancesForImplementedUsecase(t)

			expectedResponse := gen.ResponseV1Users{
				Users: []gen.UserResponse{
					{
						Name:  expectedDTO1.Name,
						Email: types.Email(expectedDTO1.Email),
						Phone: ptr.To(expectedDTO1.Phone),
					},
				},
				Limit:  mockPaging.Limit(),
				Offset: mockPaging.Offset(),
			}

			mockDTO := []user.DTO{expectedDTO1}
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				GetAllUsers(gomock.Any(), mockPaging).
				Return(mockDTO, nil)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsers(ctx, mockParams)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetUsers200JSONResponse)
			require.True(t, ok)

			require.Equal(t, expectedResponse, gen.ResponseV1Users(actual))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ページング処理が失敗した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()
			ctx, ctrl, _, lt := testinstance.NewTestInstancesForImplementedUsecase(t)

			invalidPage := 1_000_000
			invalidParams := gen.GetUsersRequestObject{
				Params: gen.GetUsersParams{
					Page:    ptr.To(invalidPage),
					PerPage: mockParams.Params.PerPage,
				},
			}

			mockApp := mock_user.NewMockUsecase(ctrl)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsers(ctx, invalidParams)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("Usecaseがエラーを返した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()

			ctx, ctrl, _, lt := testinstance.NewTestInstancesForImplementedUsecase(t)

			expectedError := apperror.ErrInternal

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				GetAllUsers(gomock.Any(), mockPaging).
				Return(nil, expectedError)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsers(ctx, mockParams)
			require.Nil(t, resp)
			require.ErrorIs(t, err, expectedError)
		})
	})
}
