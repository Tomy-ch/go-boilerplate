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
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const detailPath = "/v1/users/123e4567-e89b-12d3-a456-426614174000"

func TestV1UsersDetail_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/users/{user_id}がUserResponseを返す", func(t *testing.T) {
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
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponse(t, expected, actual)
		})

		t.Run("PUT /v1/users/{user_id}が更新後のUserResponseを返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				UpdateUser(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.MutableFields{})).
				Return(user.MutableFields{FirstName: "First", Email: "put@example.com"}, nil)

			detail.BindHandler(e, tf, mockApp)

			body := &detailgen.PutUsersDetailJSONRequestBody{
				FirstName: "First", LastName: "Last", Email: types.Email("put@example.com"),
				Phone: "09000000000", PostalCode: "123-4567", Prefecture: "Tokyo",
				City: "Shibuya", Street: "1-1-1", Building: ptr.To("Building"),
			}

			actual := StartServer(t, e).DoJSON(http.MethodPut, detailPath, body, nil)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			// Presenter / 型変換まで含め、レスポンスが gen.UserResponse にデコード可能か検証
			AssertJSONResponse(t, detailgen.UserResponse{}, actual)
		})

		t.Run("PATCH /v1/users/{user_id}が部分更新後のUserResponseを返す", func(t *testing.T) {
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
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			// Presenter / 型変換まで含め、レスポンスが gen.UserResponse にデコード可能か検証
			AssertJSONResponse(t, detailgen.UserResponse{}, actual)
		})

		t.Run("PUT /v1/users/me/passwordがパスワード変更を行い204を返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uid := uuid.NewTestFromSalt(t, "me-password")
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				ChangePassword(gomock.Any(), uid, "current_password", "new_valid_password").
				Return(nil)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uid)

			body := &detailgen.PutUsersMePasswordJSONRequestBody{ //nolint:gosec // G101: テスト用のダミーパスワードで実際の資格情報ではない
				CurrentPassword: "current_password",
				NewPassword:     "new_valid_password",
			}

			actual := StartServer(t, e).DoJSON(http.MethodPut, "/v1/users/me/password", body, headers)
			assert.Equal(t, http.StatusNoContent, actual.StatusCode)
		})

		t.Run("DELETE /v1/users/{user_id}が削除を行い204を返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().DeleteUser(gomock.Any(), gomock.Any()).Return(nil)

			detail.BindHandler(e, tf, mockApp)

			actual := StartServer(t, e).DoJSON(http.MethodDelete, detailPath, nil, nil)
			assert.Equal(t, http.StatusNoContent, actual.StatusCode)
		})
	})
}
