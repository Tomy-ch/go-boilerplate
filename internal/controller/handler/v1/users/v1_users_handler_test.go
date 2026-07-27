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
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/users"

// wantUserResponse は、本番 toUserResponse とは独立な検証用オラクル（フィールド取り違え検出）。
func wantUserResponse(dto user.UserView) gen.UserResponse {
	return gen.UserResponse{
		FirstName:  dto.FirstName,
		LastName:   dto.LastName,
		Email:      types.Email(dto.Email),
		Phone:      dto.Phone,
		PostalCode: dto.PostalCode,
		Prefecture: dto.PrefectureName,
		City:       dto.City,
		Street:     dto.Street,
		Building:   dto.Building,
		DeletedAt:  dto.DeletedAt,
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)

	mockApp := mock_user.NewMockUsecase(ctrl)

	BindHandler(e, tf, mockApp, idempotency.Deps{})

	// /v1/users (GET, POST) が登録される。
	routes := e.Routes()

	expectedMethods := []string{
		http.MethodGet,  // GetUsers
		http.MethodPost, // PostUsers
	}
	testassert.AssertEchoRouterMethods(t, expectedMethods, routes)

	actualPaths := make([]string, len(routes))
	for i, r := range routes {
		actualPaths[i] = r.Path
	}
	expectedPaths := []string{
		targetPath,
		targetPath,
	}
	assert.ElementsMatch(t, expectedPaths, actualPaths)
}

func Test_server_GetUsers(t *testing.T) {
	t.Parallel()

	expectedPage := 1
	expectedPerPage := 10

	expectedDTO1 := user.UserView{
		FirstName: "User1", LastName: "One", Email: "user1@example.com", Phone: "1234567890",
		PostalCode: "100-0001", PrefectureName: "Tokyo", City: "Chiyoda", Street: "1-1", Building: new("B1"),
	}
	expectedDTO2 := user.UserView{
		FirstName: "User2", LastName: "Two", Email: "user2@example.com", Phone: "0987654321",
		PostalCode: "200-0002", PrefectureName: "Osaka", City: "Kita", Street: "2-2", Building: new("B2"),
	}

	mockPage, err := paging.NewPageFrom1Based(new(expectedPage), new(expectedPerPage))
	require.NoError(t, err)

	mockParams := gen.GetUsersRequestObject{
		Params: gen.GetUsersParams{
			Page:    new(expectedPage),
			PerPage: new(expectedPerPage),
		},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		exec := func(t *testing.T, dtos []user.UserView, total int64) {
			t.Helper()
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockControllerLayerTracer(t)

			wantUsers := make([]gen.UserResponse, len(dtos))
			for i, d := range dtos {
				wantUsers[i] = wantUserResponse(d)
			}
			expectedResponse := gen.UsersResponse{
				Users:  wantUsers,
				Limit:  mockPage.Limit(),
				Offset: mockPage.Offset(),
				Total:  total,
			}

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersWithTotal(gomock.Any(), mockParams.Params.Active, mockPage).
				Return(&user.UserListView{Items: dtos, Total: total}, nil)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.GetUsers(ctx, mockParams)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetUsers200JSONResponse)
			require.True(t, ok)

			assert.Equal(t, expectedResponse, gen.UsersResponse(actual))
		}

		t.Run("複数のユーザーが存在する場合、ユーザー情報のリストが取得できる", func(t *testing.T) {
			t.Parallel()
			exec(t, []user.UserView{expectedDTO1, expectedDTO2}, 2)
		})

		t.Run("単一のユーザーが存在する場合、ユーザー情報のリストが取得できる", func(t *testing.T) {
			t.Parallel()
			exec(t, []user.UserView{expectedDTO1}, 1)
		})

		t.Run("ユーザーが0件の場合、空リストと200が返る", func(t *testing.T) {
			t.Parallel()
			exec(t, []user.UserView{}, 0)
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
					Page:    new(invalidPage),
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

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersWithTotal(gomock.Any(), mockParams.Params.Active, mockPage).
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

	userID, err := uuid.New()
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みユーザーが登録した場合_201とUserViewが返る", func(t *testing.T) {
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
					Building:   new("Building"),
				},
			}

			wantParams := user.UpdateProfileParams{
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
			wantView := user.UserView{
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

			// 認証 subject 由来の UserID 注入とリクエスト→DTO 詰め替えを引数捕捉で検証する。
			var gotParams *user.CreateParamsDTO
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().CreateUser(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, p *user.CreateParamsDTO) (user.UserView, error) {
					gotParams = p
					return wantView, nil
				})

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.PostUsers(ctx, req)
			require.NoError(t, err)

			expectedParams := &user.CreateParamsDTO{
				UserID:              userID,
				UpdateProfileParams: wantParams,
			}
			assert.Equal(t, expectedParams, gotParams)

			actual, ok := resp.(gen.PostUsers201JSONResponse)
			require.True(t, ok)

			assert.Equal(t, wantUserResponse(wantView), gen.UserResponse(actual))
		})
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
				},
			}

			mockApp := mock_user.NewMockUsecase(ctrl)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.PostUsers(ctx, req)

			require.Nil(t, resp)
			require.ErrorContains(t, err, "failed to get user ID from authenticator")
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
				},
			}

			mockApp := mock_user.NewMockUsecase(ctrl)
			expectedErr := apperror.ErrInternal
			mockApp.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&user.CreateParamsDTO{})).Return(user.UserView{}, expectedErr)

			s := &server{tracer: lt, uc: mockApp}
			resp, err := s.PostUsers(ctx, req)
			require.Nil(t, resp)
			require.ErrorIs(t, err, expectedErr)
		})
	})
}

func Test_toUserResponse(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
