package integration

import (
	"net/http"
	"testing"

	v1users "boilerplate-go/internal/controller/handler/v1/users"
	"boilerplate-go/internal/controller/handler/v1/users/gen"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/user"
	mock_user "boilerplate-go/internal/usecase/user/mock"
	"boilerplate-go/pkg/ptr"
	"boilerplate-go/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestV1Users_Integration(t *testing.T) {
	t.Parallel()

	expectedDTO := user.MutableFields{FirstName: "User1", LastName: "One", Email: "user1@example.com", Phone: "1234567890"}

	t.Run("GET /v1/usersのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e := echo.New()
		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			ListUsersByKeyword(gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]user.MutableFields{expectedDTO}, nil)

		v1users.BindHandler(e, tf, mockApp)

		expected := gen.ResponseV1Users{
			Users: []gen.UserResponse{
				{
					FirstName: expectedDTO.FirstName,
					LastName:  expectedDTO.LastName,
					Email:     types.Email(expectedDTO.Email),
					Phone:     ptr.To(expectedDTO.Phone),
				},
			},
		}

		actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/users", nil, nil)
		AssertJSONResponse(t, expected, actual)
	})

	t.Run("POST /v1/usersのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e := echo.New()
		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			CreateUser(gomock.Any(), gomock.Any()).
			Return(user.MutableFields{}, nil)

		v1users.BindHandler(e, tf, mockApp)

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

		uuid, err := uuid.Parse("d1f64798-7321-242b-e4ff-115f6a0b7803")
		require.NoError(t, err)
		headers := MakeAvailableUserID(t, e, uuid)
		actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/users", req.Body, headers)
		require.Equal(t, http.StatusCreated, actual.StatusCode)
	})
}
