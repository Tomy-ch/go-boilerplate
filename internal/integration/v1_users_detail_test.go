package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	detail "go-boilerplate/internal/controller/handler/v1/users/detail"
	detailgen "go-boilerplate/internal/controller/handler/v1/users/detail/gen"
	domainuser "go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const detailPath = "/v1/users/123e4567-e89b-12d3-a456-426614174000"

func TestV1UsersDetail_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/users/{userId}がUserResponseを返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			expectedDTO := user.UserView{
				FirstName: "User1", LastName: "One", Email: "user1@example.com", Phone: "09000000000",
				PostalCode: "150-0041", PrefectureName: "Tokyo", City: "Shibuya", Street: "1-2-3",
			}
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedDTO, nil)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "me-get"))

			actual := StartServer(t, e).DoJSON(http.MethodGet, detailPath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[detailgen.UserResponse](t, actual)
		})

		t.Run("PUT /v1/users/{userId}が更新後のUserResponseを返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				UpdateUser(gomock.Any(), gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.UpdateProfileParams{})).
				Return(user.UserView{FirstName: "First", Email: "put@example.com"}, nil)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "me-put"))

			body := &detailgen.PutUsersDetailJSONRequestBody{
				FirstName: "First", LastName: "Last", Email: types.Email("put@example.com"),
				Phone: "09000000000", PostalCode: "123-4567", Prefecture: "Tokyo",
				City: "Shibuya", Street: "1-1-1", Building: new("Building"),
			}

			actual := StartServer(t, e).DoJSON(http.MethodPut, detailPath, body, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			// Presenter / 型変換まで含め、レスポンスが gen.UserResponse にデコード可能か検証
			AssertJSONResponseType[detailgen.UserResponse](t, actual)
		})

		t.Run("PATCH /v1/users/{userId}が部分更新後のUserResponseを返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				UpdateUserPartially(gomock.Any(), gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.PatchParamsDTO{})).
				Return(user.UserView{FirstName: "Patched", Email: "patch@example.com"}, nil)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "me-patch"))

			body := &detailgen.PatchUsersDetailJSONRequestBody{
				FirstName: new("Patched"),
			}

			actual := StartServer(t, e).DoJSON(http.MethodPatch, detailPath, body, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			// Presenter / 型変換まで含め、レスポンスが gen.UserResponse にデコード可能か検証
			AssertJSONResponseType[detailgen.UserResponse](t, actual)
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

		t.Run("DELETE /v1/users/{userId}が削除を行い204を返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uid := uuid.NewTestFromSalt(t, "me-delete")
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().DeleteUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uid)

			actual := StartServer(t, e).DoJSON(http.MethodDelete, detailPath, nil, headers)
			assert.Equal(t, http.StatusNoContent, actual.StatusCode)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/users/{userId}がErrNotFoundで404を返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(user.UserView{}, apperror.ErrNotFound)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "me-get-404"))

			actual := StartServer(t, e).DoJSON(http.MethodGet, detailPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusNotFound)
		})

		t.Run("GET /v1/users/{userId}がErrInternalで500を返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(user.UserView{}, apperror.ErrInternal)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "me-get-500"))

			actual := StartServer(t, e).DoJSON(http.MethodGet, detailPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})

		t.Run("PUT /v1/users/{userId}が複数フィールド不正で422とdetailsを返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			validationErr := apperror.WithDetails(xerrors.Join(
				xerrors.Wrap(domainuser.ErrInvalidFirstName, "length must be between 1 and 100 characters (got 0)"),
				xerrors.Wrap(domainuser.ErrInvalidEmail, "length must be between 1 and 100 characters (got 101)"),
			), domainuser.FieldFirstName, domainuser.FieldEmail)
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				UpdateUser(gomock.Any(), gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.UpdateProfileParams{})).
				Return(user.UserView{}, validationErr)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "me-put-422"))

			body := &detailgen.PutUsersDetailJSONRequestBody{
				FirstName: "First", LastName: "Last", Email: types.Email("put@example.com"),
				Phone: "09000000000", PostalCode: "123-4567", Prefecture: "Tokyo",
				City: "Shibuya", Street: "1-1-1",
			}

			actual := StartServer(t, e).DoJSON(http.MethodPut, detailPath, body, headers)
			errResp := AssertErrorResponseBody(t, actual, http.StatusUnprocessableEntity)
			require.NotNil(t, errResp.Details)
			assert.Equal(t, []string{domainuser.FieldFirstName, domainuser.FieldEmail}, *errResp.Details)
			// 理由文はログ専用であり、レスポンスの details には露出しない
			assert.NotContains(t, *errResp.Details, "length must be between 1 and 100 characters (got 0)")
		})

		t.Run("PATCH /v1/users/{userId}が単一フィールド不正で422とdetailsを返す", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			validationErr := apperror.WithDetails(
				xerrors.Wrap(domainuser.ErrInvalidFirstName, "length must be between 1 and 100 characters (got 0)"),
				domainuser.FieldFirstName,
			)
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().
				UpdateUserPartially(gomock.Any(), gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.PatchParamsDTO{})).
				Return(user.UserView{}, validationErr)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "me-patch-422"))

			body := &detailgen.PatchUsersDetailJSONRequestBody{
				FirstName: new(""),
			}

			actual := StartServer(t, e).DoJSON(http.MethodPatch, detailPath, body, headers)
			errResp := AssertErrorResponseBody(t, actual, http.StatusUnprocessableEntity)
			require.NotNil(t, errResp.Details)
			assert.Equal(t, []string{domainuser.FieldFirstName}, *errResp.Details)
		})

		t.Run("details未対応のGETはMeta付きエラーでもdetailsを返さない(fail-closed)", func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			// GET /v1/users/{userId} は OpenAPI で ErrorResponseWithDetails を宣言していない(opt-in 外)。
			// Meta に details が付いていても errorhandler が fail-closed で落とす。
			metaErr := apperror.WithDetails(
				xerrors.Wrap(domainuser.ErrInvalidFirstName, "length must be between 1 and 100 characters (got 0)"),
				domainuser.FieldFirstName,
			)
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(user.UserView{}, metaErr)

			detail.BindHandler(e, tf, mockApp)
			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "me-get-nodetails"))

			actual := StartServer(t, e).DoJSON(http.MethodGet, detailPath, nil, headers)
			errResp := AssertErrorResponseBody(t, actual, http.StatusUnprocessableEntity)
			assert.Nil(t, errResp.Details)
		})
	})
}
