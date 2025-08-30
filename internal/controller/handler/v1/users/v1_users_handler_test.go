package v1users

import (
	"net/http"
	"testing"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/controller/handler/handlertest"
	"boilerplate-go/internal/controller/handler/v1/users/gen"
	"boilerplate-go/internal/usecase/paging"
	useruc "boilerplate-go/internal/usecase/user"
	mock_useruc "boilerplate-go/internal/usecase/user/mock"
	"boilerplate-go/pkg/ptr"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/users"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockApp := mock_useruc.NewMockUsecase(ctrl)

	BindHandler(e, mockApp)

	expectedMethods := []string{
		http.MethodGet,
		http.MethodPost,
	}

	handlertest.AssertEchoRouterPath(
		t, targetPath, e.Routes(),
	)
	handlertest.AssertEchoRouterMethods(
		t, expectedMethods, e.Routes(),
	)
}

func Test_server_GetUsers(t *testing.T) {
	t.Parallel()

	expectedPage := 1
	expectedPerPage := 10

	expectedDTO1 := useruc.DTO{Name: "User1", Email: "user1@example.com", Phone: "1234567890"}
	expectedDTO2 := useruc.DTO{Name: "User2", Email: "user2@example.com", Phone: "0987654321"}

	mockPaging, err := paging.NewPagingFrom1Based(ptr.To(expectedPage), ptr.To(expectedPerPage))
	require.NoError(t, err)

	mockParams := gen.GetUsersParams{
		Page:    ptr.To(expectedPage),
		PerPage: ptr.To(expectedPerPage),
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数のユーザーが存在する場合、ユーザー情報のリストが取得できる", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

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

			mockDTO := []useruc.DTO{expectedDTO1, expectedDTO2}
			mockApp := mock_useruc.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				GetAllUsers(gomock.Any(), mockPaging).
				Return(mockDTO, nil)

			BindHandler(e, mockApp)

			_, res, ctx := handlertest.
				NewEchoTestClient(t, e).
				Method(http.MethodGet).
				RequestURL(targetPath).
				Build()

			s := &server{uc: mockApp}
			err := s.GetUsers(ctx, mockParams)
			require.NoError(t, err)

			handlertest.AssertJSONEqual(t, http.StatusOK, expectedResponse, res)
		})

		t.Run("単一のユーザーが存在する場合、ユーザー情報のリストが取得できる", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

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

			mockDTO := []useruc.DTO{expectedDTO1}
			mockApp := mock_useruc.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				GetAllUsers(gomock.Any(), mockPaging).
				Return(mockDTO, nil)

			BindHandler(e, mockApp)

			_, res, ctx := handlertest.
				NewEchoTestClient(t, e).
				Method(http.MethodGet).
				RequestURL(targetPath).
				Build()

			s := &server{uc: mockApp}
			err := s.GetUsers(ctx, mockParams)
			require.NoError(t, err)

			handlertest.AssertJSONEqual(t, http.StatusOK, expectedResponse, res)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ページング処理が失敗した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			invalidPage := 1_000_000
			invalidParams := mockParams
			invalidParams.Page = ptr.To(invalidPage)

			mockApp := mock_useruc.NewMockUsecase(ctrl)
			BindHandler(e, mockApp)

			_, _, ctx := handlertest.
				NewEchoTestClient(t, e).
				Method(http.MethodGet).
				RequestURL(targetPath).
				Build()

			s := &server{uc: mockApp}
			err := s.GetUsers(ctx, invalidParams)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("Usecaseがエラーを返した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockParams := gen.GetUsersParams{
				Page:    ptr.To(expectedPage),
				PerPage: ptr.To(expectedPerPage),
			}

			expectedError := apperror.ErrInternal

			mockApp := mock_useruc.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				GetAllUsers(gomock.Any(), mockPaging).
				Return(nil, expectedError)

			BindHandler(e, mockApp)

			_, _, ctx := handlertest.
				NewEchoTestClient(t, e).
				Method(http.MethodGet).
				RequestURL(targetPath).
				Build()

			s := &server{uc: mockApp}
			err := s.GetUsers(ctx, mockParams)
			require.ErrorIs(t, err, expectedError)
		})
	})
}
