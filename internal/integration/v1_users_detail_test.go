package integration

import (
	"net/http"
	"testing"

	detail "go-boilerplate/internal/controller/handler/v1/users/detail"
	detailgen "go-boilerplate/internal/controller/handler/v1/users/detail/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/ptr"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const detailPath = "/v1/users/123e4567-e89b-12d3-a456-426614174000"

func TestV1UsersDetail_Integration(t *testing.T) {
	t.Parallel()

	t.Run("GET /v1/users/{user_id} のエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)

		expectedDTO := user.MutableFields{
			FirstName: "User1", LastName: "One", Email: "user1@example.com", Phone: "09000000000",
			PostalCode: "150-0041", PrefectureName: "Tokyo", City: "Shibuya", Street: "1-2-3",
		}
		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(expectedDTO, nil)

		detail.BindHandler(e, tf, mockApp)

		expected := detailgen.UserResponse{
			FirstName:  expectedDTO.FirstName,
			LastName:   expectedDTO.LastName,
			Email:      types.Email(expectedDTO.Email),
			Phone:      expectedDTO.Phone,
			PostalCode: expectedDTO.PostalCode,
			Prefecture: expectedDTO.PrefectureName,
			City:       expectedDTO.City,
			Street:     expectedDTO.Street,
		}

		actual := StartServer(t, e).DoJSON(http.MethodGet, detailPath, nil, nil)
		require.Equal(t, http.StatusOK, actual.StatusCode)
		AssertJSONResponse(t, expected, actual)
	})

	t.Run("PUT /v1/users/{user_id} のエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			UpdateUser(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.UpdateParamsDTO{})).
			Return(user.MutableFields{FirstName: "First", Email: "put@example.com"}, nil)

		detail.BindHandler(e, tf, mockApp)

		body := &detailgen.PutUsersDetailJSONRequestBody{
			FirstName: "First", LastName: "Last", Email: types.Email("put@example.com"),
			Phone: "09000000000", PostalCode: "123-4567", Prefecture: "Tokyo",
			City: "Shibuya", Street: "1-1-1", Building: ptr.To("Building"), Password: "secretpw",
		}

		actual := StartServer(t, e).DoJSON(http.MethodPut, detailPath, body, nil)
		require.Equal(t, http.StatusOK, actual.StatusCode)
	})

	t.Run("PATCH /v1/users/{user_id} のエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			UpdateUserPartially(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.PatchParamsDTO{})).
			Return(user.MutableFields{FirstName: "Patched", Email: "patch@example.com"}, nil)

		detail.BindHandler(e, tf, mockApp)

		body := &detailgen.PatchUsersDetailJSONRequestBody{
			FirstName: ptr.To("Patched"),
		}

		actual := StartServer(t, e).DoJSON(http.MethodPatch, detailPath, body, nil)
		require.Equal(t, http.StatusOK, actual.StatusCode)
	})

	t.Run("DELETE /v1/users/{user_id} のエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().DeleteUser(gomock.Any(), gomock.Any()).Return(nil)

		detail.BindHandler(e, tf, mockApp)

		actual := StartServer(t, e).DoJSON(http.MethodDelete, detailPath, nil, nil)
		require.Equal(t, http.StatusNoContent, actual.StatusCode)
	})
}
