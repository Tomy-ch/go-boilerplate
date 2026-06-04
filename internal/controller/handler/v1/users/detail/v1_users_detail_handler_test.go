package detail

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testauth"
	"go-boilerplate/internal/controller/handler/testkit/testuuid"
	"go-boilerplate/internal/controller/handler/v1/users/detail/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/ptr"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/users/:user_id"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)

	mockApp := mock_user.NewMockUsecase(ctrl)

	BindHandler(e, tf, mockApp)

	got := make(map[string]bool, len(e.Routes()))
	for _, r := range e.Routes() {
		got[r.Method+" "+r.Path] = true
	}

	expected := []string{
		http.MethodGet + " " + targetPath,
		http.MethodPut + " " + targetPath,
		http.MethodPatch + " " + targetPath,
		http.MethodDelete + " " + targetPath,
		http.MethodPut + " /v1/users/me/password",
	}

	assert.Len(t, e.Routes(), len(expected))
	for _, route := range expected {
		assert.Contains(t, got, route)
	}
}

func Test_server_GetUsersDetail(t *testing.T) {
	t.Parallel()

	expectedDTO := user.MutableFields{
		FirstName: "User1", LastName: "One", Email: "user1@example.com", Phone: "1234567890",
		PostalCode: "150-0041", PrefectureName: "Tokyo", City: "Shibuya", Street: "1-2-3",
	}

	t.Run("正常系_ユーザーが存在する場合_詳細が取得できる", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(expectedDTO, nil)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.GetUsersDetail(ctx, gen.GetUsersDetailRequestObject{UserId: testuuid.RequestUUID(t)})
		require.NoError(t, err)

		actual, ok := resp.(gen.GetUsersDetail200JSONResponse)
		assert.True(t, ok)
		assert.Equal(t, expectedDTO.FirstName, actual.FirstName)
		assert.Equal(t, types.Email(expectedDTO.Email), actual.Email)
		assert.Equal(t, expectedDTO.PrefectureName, actual.Prefecture)
	})

	t.Run("異常系_Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(user.MutableFields{}, apperror.ErrNotFound)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.GetUsersDetail(ctx, gen.GetUsersDetailRequestObject{UserId: testuuid.RequestUUID(t)})
		require.Nil(t, resp)
		require.ErrorIs(t, err, apperror.ErrNotFound)
	})
}

func Test_server_PutUsersDetail(t *testing.T) {
	t.Parallel()

	body := &gen.PutUsersDetailJSONRequestBody{
		FirstName: "First", LastName: "Last", Email: types.Email("put@example.com"),
		Phone: "09000000000", PostalCode: "123-4567", Prefecture: "Tokyo",
		City: "Shibuya", Street: "1-1-1", Building: ptr.To("Building"),
	}

	t.Run("正常系_全更新が成功する場合_更新後のユーザーが返る", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		expectedDTO := user.MutableFields{FirstName: "First", LastName: "Last", Email: "put@example.com", PrefectureName: "Tokyo"}
		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			UpdateUser(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.MutableFields{})).
			Return(expectedDTO, nil)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.PutUsersDetail(ctx, gen.PutUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body})
		require.NoError(t, err)

		actual, ok := resp.(gen.PutUsersDetail200JSONResponse)
		assert.True(t, ok)
		assert.Equal(t, expectedDTO.FirstName, actual.FirstName)
	})

	t.Run("異常系_Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			UpdateUser(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.MutableFields{})).
			Return(user.MutableFields{}, apperror.ErrInternal)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.PutUsersDetail(ctx, gen.PutUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body})
		require.Nil(t, resp)
		require.ErrorIs(t, err, apperror.ErrInternal)
	})
}

func Test_server_PatchUsersDetail(t *testing.T) {
	t.Parallel()

	body := &gen.PatchUsersDetailJSONRequestBody{
		FirstName: ptr.To("Patched"),
		Email:     (*types.Email)(ptr.To("patch@example.com")),
	}

	t.Run("正常系_部分更新が成功する場合_更新後のユーザーが返る", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		expectedDTO := user.MutableFields{FirstName: "Patched", Email: "patch@example.com"}
		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			UpdateUserPartially(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.PatchParamsDTO{})).
			Return(expectedDTO, nil)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.PatchUsersDetail(ctx, gen.PatchUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body})
		require.NoError(t, err)

		actual, ok := resp.(gen.PatchUsersDetail200JSONResponse)
		assert.True(t, ok)
		assert.Equal(t, expectedDTO.FirstName, actual.FirstName)
	})

	t.Run("正常系_Email未指定の場合も部分更新できる", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		// Email を含まない部分更新（emailToStringPtr の nil 経路）
		noEmailBody := &gen.PatchUsersDetailJSONRequestBody{FirstName: ptr.To("OnlyName")}

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			UpdateUserPartially(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.PatchParamsDTO{})).
			Return(user.MutableFields{FirstName: "OnlyName"}, nil)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.PatchUsersDetail(ctx, gen.PatchUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: noEmailBody})
		require.NoError(t, err)

		actual, ok := resp.(gen.PatchUsersDetail200JSONResponse)
		assert.True(t, ok)
		assert.Equal(t, "OnlyName", actual.FirstName)
	})

	t.Run("異常系_Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			UpdateUserPartially(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.PatchParamsDTO{})).
			Return(user.MutableFields{}, apperror.ErrInternal)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.PatchUsersDetail(ctx, gen.PatchUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body})
		require.Nil(t, resp)
		require.ErrorIs(t, err, apperror.ErrInternal)
	})
}

func Test_server_PutUsersMePassword(t *testing.T) {
	t.Parallel()

	const subject = "11111111-1111-1111-1111-111111111111"
	body := &gen.PutUsersMePasswordJSONRequestBody{ //nolint:gosec // G101: テスト用のダミーパスワードで実際の資格情報ではない
		CurrentPassword: "current_password",
		NewPassword:     "new_valid_password",
	}

	t.Run("正常系_認証ユーザーのパスワード変更が成功する場合_204が返る", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)
		ctx := testauth.MakeAvailableAuthn(context.Background(), t, subject)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			ChangePassword(gomock.Any(), gomock.Any(), "current_password", "new_valid_password").
			Return(nil)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.PutUsersMePassword(ctx, gen.PutUsersMePasswordRequestObject{Body: body})
		require.NoError(t, err)

		_, ok := resp.(gen.PutUsersMePassword204Response)
		assert.True(t, ok)
	})

	t.Run("異常系_認証情報がない場合_エラーが返る", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		s := &server{tracer: lt, uc: mock_user.NewMockUsecase(ctrl)}
		resp, err := s.PutUsersMePassword(context.Background(), gen.PutUsersMePasswordRequestObject{Body: body})
		require.Nil(t, resp)
		require.ErrorIs(t, err, ErrUnauthenticatedUser)
	})

	t.Run("異常系_認証subjectが不正でID取得に失敗する場合_エラーが返る", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)
		ctx := testauth.MakeAvailableAuthn(context.Background(), t, "invalid-subject")

		s := &server{tracer: lt, uc: mock_user.NewMockUsecase(ctrl)}
		resp, err := s.PutUsersMePassword(ctx, gen.PutUsersMePasswordRequestObject{Body: body})
		require.Nil(t, resp)
		require.Error(t, err)
	})

	t.Run("異常系_Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)
		ctx := testauth.MakeAvailableAuthn(context.Background(), t, subject)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			ChangePassword(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(apperror.ErrValidation)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.PutUsersMePassword(ctx, gen.PutUsersMePasswordRequestObject{Body: body})
		require.Nil(t, resp)
		require.ErrorIs(t, err, apperror.ErrValidation)
	})
}

func Test_server_DeleteUsersDetail(t *testing.T) {
	t.Parallel()

	t.Run("正常系_削除が成功する場合_204が返る", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().DeleteUser(gomock.Any(), gomock.Any()).Return(nil)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.DeleteUsersDetail(ctx, gen.DeleteUsersDetailRequestObject{UserId: testuuid.RequestUUID(t)})
		require.NoError(t, err)

		_, ok := resp.(gen.DeleteUsersDetail204Response)
		assert.True(t, ok)
	})

	t.Run("異常系_Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().DeleteUser(gomock.Any(), gomock.Any()).Return(apperror.ErrNotFound)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.DeleteUsersDetail(ctx, gen.DeleteUsersDetailRequestObject{UserId: testuuid.RequestUUID(t)})
		require.Nil(t, resp)
		require.ErrorIs(t, err, apperror.ErrNotFound)
	})
}
