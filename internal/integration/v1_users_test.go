package integration

import (
	"net/http"
	"testing"

	v1users "go-boilerplate/internal/controller/handler/v1/users"
	"go-boilerplate/internal/controller/handler/v1/users/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestV1Users_Integration(t *testing.T) {
	t.Parallel()

	expectedDTO := user.UserView{FirstName: "User1", LastName: "One", Email: "user1@example.com", Phone: "1234567890"}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/usersがユーザーリストを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ListUsersWithTotal(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&user.UserListView{Items: []user.UserView{expectedDTO}, Total: 1}, nil)

			v1users.BindHandler(e, tf, mockApp, idempotency.Deps{})

			expected := gen.UsersResponse{
				Users: []gen.UserResponse{
					{
						FirstName: expectedDTO.FirstName,
						LastName:  expectedDTO.LastName,
						Email:     types.Email(expectedDTO.Email),
						Phone:     expectedDTO.Phone,
					},
				},
			}

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users", nil, nil)
			AssertJSONResponse(t, expected, actual)
		})

		t.Run("POST /v1/usersがユーザー作成を行い201を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				CreateUser(gomock.Any(), gomock.Any()).
				Return(user.UserView{Email: "new@example.com"}, nil)

			v1users.BindHandler(e, tf, mockApp, idempotency.Deps{})

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
					Password:   "secret",
				},
			}

			uuid, err := uuid.Parse("d1f64798-7321-242b-e4ff-115f6a0b7803")
			require.NoError(t, err)
			headers := MakeAvailableUserID(t, e, uuid)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/users", req.Body, headers)
			assert.Equal(t, http.StatusCreated, actual.StatusCode)
		})
	})
}
