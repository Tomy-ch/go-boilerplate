package users

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/testkit/testauth"
	"go-boilerplate/internal/controller/handler/v1/users/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
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
	expectedTotal := int64(2)

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

			expectedResponse := gen.UsersResponse{
				Users: []gen.UserResponse{
					{
						FirstName: expectedDTO1.FirstName,
						LastName:  expectedDTO1.LastName,
						Email:     types.Email(expectedDTO1.Email),
						Phone:     expectedDTO1.Phone,
					},
					{
						FirstName: expectedDTO2.FirstName,
						LastName:  expectedDTO2.LastName,
						Email:     types.Email(expectedDTO2.Email),
						Phone:     expectedDTO2.Phone,
					},
				},
				Limit:  mockPaging.Limit(),
				Offset: mockPaging.Offset(),
				Total:  expectedTotal,
			}

			mockDTO := []user.MutableFields{expectedDTO1, expectedDTO2}
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsers(gomock.Any(), mockParams.Params.Active, mockPaging).
				Return(mockDTO, nil)
			mockApp.EXPECT().
				CountUsers(gomock.Any(), mockParams.Params.Active).
				Return(expectedTotal, nil)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsers(ctx, mockParams)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetUsers200JSONResponse)
			assert.True(t, ok)

			assert.Equal(t, expectedResponse, gen.UsersResponse(actual))
		})

		t.Run("単一のユーザーが存在する場合、ユーザー情報のリストが取得できる", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockControllerLayerTracer(t)

			expectedResponse := gen.UsersResponse{
				Users: []gen.UserResponse{
					{
						FirstName: expectedDTO1.FirstName,
						LastName:  expectedDTO1.LastName,
						Email:     types.Email(expectedDTO1.Email),
						Phone:     expectedDTO1.Phone,
					},
				},
				Limit:  mockPaging.Limit(),
				Offset: mockPaging.Offset(),
				Total:  expectedTotal,
			}

			mockDTO := []user.MutableFields{expectedDTO1}
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsers(gomock.Any(), mockParams.Params.Active, mockPaging).
				Return(mockDTO, nil)
			mockApp.EXPECT().
				CountUsers(gomock.Any(), mockParams.Params.Active).
				Return(expectedTotal, nil)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsers(ctx, mockParams)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetUsers200JSONResponse)
			assert.True(t, ok)

			assert.Equal(t, expectedResponse, gen.UsersResponse(actual))
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

			t.Run("ListUsersByKeywordがエラーを返す場合", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				ctrl := gomock.NewController(t)
				lt := observability.NewMockControllerLayerTracer(t)

				expectedError := apperror.ErrInternal

				mockApp := mock_user.NewMockUsecase(ctrl)
				mockApp.EXPECT().
					ListUsers(gomock.Any(), mockParams.Params.Active, mockPaging).
					Return(nil, expectedError)

				s := &server{tracer: lt, uc: mockApp}
				resp, err := s.GetUsers(ctx, mockParams)
				require.Nil(t, resp)
				require.ErrorIs(t, err, expectedError)
			})

			t.Run("CountUsersがエラーを返す場合", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				ctrl := gomock.NewController(t)
				lt := observability.NewMockControllerLayerTracer(t)

				expectedError := apperror.ErrInternal

				mockDTO := []user.MutableFields{expectedDTO1, expectedDTO2}
				mockApp := mock_user.NewMockUsecase(ctrl)
				mockApp.EXPECT().
					ListUsers(gomock.Any(), mockParams.Params.Active, mockPaging).
					Return(mockDTO, nil)
				mockApp.EXPECT().
					CountUsers(gomock.Any(), mockParams.Params.Active).
					Return(int64(0), expectedError)

				s := &server{tracer: lt, uc: mockApp}
				resp, err := s.GetUsers(ctx, mockParams)
				require.Nil(t, resp)
				require.ErrorIs(t, err, expectedError)
			})
		})
	})
}

func Test_server_PostUsers(t *testing.T) {
	t.Parallel()

	userID, err := uuid.New()
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx = testauth.MakeAvailableAuthn(ctx, t, userID.String())
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
		assert.True(t, ok)

		got := gen.UserResponse(actual)
		assert.Equal(t, expectedDTO.FirstName, got.FirstName)
		assert.Equal(t, expectedDTO.LastName, got.LastName)
		assert.Equal(t, types.Email(expectedDTO.Email), got.Email)
		assert.Equal(t, expectedDTO.Phone, got.Phone)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報が存在しない場合、ErrUnauthenticatedUserが返る", func(t *testing.T) {
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

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.PostUsers(ctx, req)

			require.Nil(t, resp)
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("認証データのsubjectにuuidが含まれない場合、エラーが返る", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = testauth.MakeAvailableAuthn(ctx, t, "invalid-subject")

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

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.PostUsers(ctx, req)

			require.Nil(t, resp)
			require.Error(t, err)
		})

		t.Run("Usecaseがエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = testauth.MakeAvailableAuthn(ctx, t, userID.String())
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
	})
}
