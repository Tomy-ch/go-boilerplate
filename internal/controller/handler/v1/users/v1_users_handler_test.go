package users

import (
	"context"
	"net/http"
	"testing"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/controller/handler/testkit/testassert"
	"boilerplate-go/internal/controller/handler/v1/users/gen"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/support/paging"
	"boilerplate-go/internal/usecase/user"
	mock_user "boilerplate-go/internal/usecase/user/mock"
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
	tf := observability.NewNoopTracerFactory(t)

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

	expectedDTO1 := user.MutableFields{FirstName: "User1", LastName: "One", Email: "user1@example.com", Phone: "1234567890"}
	expectedDTO2 := user.MutableFields{FirstName: "User2", LastName: "Two", Email: "user2@example.com", Phone: "0987654321"}

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
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockControllerLayerTracer(t)

			expectedResponse := gen.ResponseV1Users{
				Users: []gen.UserResponse{
					{
						FirstName: expectedDTO1.FirstName,
						LastName:  expectedDTO1.LastName,
						Email:     types.Email(expectedDTO1.Email),
						Phone:     ptr.To(expectedDTO1.Phone),
					},
					{
						FirstName: expectedDTO2.FirstName,
						LastName:  expectedDTO2.LastName,
						Email:     types.Email(expectedDTO2.Email),
						Phone:     ptr.To(expectedDTO2.Phone),
					},
				},
				Limit:  mockPaging.Limit(),
				Offset: mockPaging.Offset(),
			}

			expectedParams := &user.GetParamsDTO{}

			mockDTO := []user.MutableFields{expectedDTO1, expectedDTO2}
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersByKeyword(gomock.Any(), expectedParams, mockPaging).
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
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockControllerLayerTracer(t)

			expectedResponse := gen.ResponseV1Users{
				Users: []gen.UserResponse{
					{
						FirstName: expectedDTO1.FirstName,
						LastName:  expectedDTO1.LastName,
						Email:     types.Email(expectedDTO1.Email),
						Phone:     ptr.To(expectedDTO1.Phone),
					},
				},
				Limit:  mockPaging.Limit(),
				Offset: mockPaging.Offset(),
			}

			expectedParams := &user.GetParamsDTO{}

			mockDTO := []user.MutableFields{expectedDTO1}
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersByKeyword(gomock.Any(), expectedParams, mockPaging).
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
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockControllerLayerTracer(t)

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

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockControllerLayerTracer(t)

			expectedError := apperror.ErrInternal

			expectedParams := &user.GetParamsDTO{}

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersByKeyword(gomock.Any(), expectedParams, mockPaging).
				Return(nil, expectedError)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsers(ctx, mockParams)
			require.Nil(t, resp)
			require.ErrorIs(t, err, expectedError)
		})
	})
}

func Test_server_PostUsers(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		req := gen.PostUsersRequestObject{
			Body: &gen.PostUsersJSONRequestBody{
				FirstName:  "First",
				LastName:   "Last",
				Email:      types.Email("new@example.com"),
				Phone:      "09000000000",
				PostalCode: "123-4567",
				Prefecture: "Tokyo",
				City:       "Shibuya",
				Street:     "1-1-1",
				Building:   ptr.To("Building"),
				Password:   "secret",
			},
		}

		expectedDTO := user.MutableFields{
			FirstName:      req.Body.FirstName,
			LastName:       req.Body.LastName,
			Email:          string(req.Body.Email),
			Phone:          req.Body.Phone,
			PostalCode:     req.Body.PostalCode,
			PrefectureName: req.Body.Prefecture,
			City:           req.Body.City,
			Street:         req.Body.Street,
			Building:       req.Body.Building,
		}

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&user.CreateParamsDTO{})).Return(expectedDTO, nil)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.PostUsers(ctx, req)
		require.NoError(t, err)

		actual, ok := resp.(gen.PostUsers201JSONResponse)
		require.True(t, ok)

		got := gen.UserResponse(actual)
		require.Equal(t, expectedDTO.FirstName, got.FirstName)
		require.Equal(t, expectedDTO.LastName, got.LastName)
		require.Equal(t, types.Email(expectedDTO.Email), got.Email)
		require.Equal(t, ptr.To(expectedDTO.Phone), got.Phone)
	})

	t.Run("異常系: Usecaseがエラーを返す", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		req := gen.PostUsersRequestObject{
			Body: &gen.PostUsersJSONRequestBody{
				FirstName: "A",
				LastName:  "B",
				Email:     types.Email("err@example.com"),
				Password:  "pw",
			},
		}

		mockApp := mock_user.NewMockUsecase(ctrl)
		expectedErr := apperror.ErrInternal
		mockApp.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&user.CreateParamsDTO{})).Return(user.MutableFields{}, expectedErr)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.PostUsers(ctx, req)
		require.Nil(t, resp)
		require.ErrorIs(t, err, expectedErr)
	})
}
