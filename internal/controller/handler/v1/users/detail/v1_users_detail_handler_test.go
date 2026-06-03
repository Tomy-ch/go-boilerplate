package detail

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/testkit/testuuid"
	"go-boilerplate/internal/controller/handler/v1/users/detail/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/ptr"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
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

	expectedMethods := []string{
		http.MethodGet,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}

	testassert.AssertEchoRouterPath(t, targetPath, e.Routes())
	testassert.AssertEchoRouterMethods(t, expectedMethods, e.Routes())
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
		require.True(t, ok)
		require.Equal(t, expectedDTO.FirstName, actual.FirstName)
		require.Equal(t, types.Email(expectedDTO.Email), actual.Email)
		require.Equal(t, expectedDTO.PrefectureName, actual.Prefecture)
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
		City: "Shibuya", Street: "1-1-1", Building: ptr.To("Building"), Password: "secretpw",
	}

	t.Run("正常系_全更新が成功する場合_更新後のユーザーが返る", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		expectedDTO := user.MutableFields{FirstName: "First", LastName: "Last", Email: "put@example.com", PrefectureName: "Tokyo"}
		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			UpdateUser(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.UpdateParamsDTO{})).
			Return(expectedDTO, nil)

		s := &server{tracer: lt, uc: mockApp}
		resp, err := s.PutUsersDetail(ctx, gen.PutUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body})
		require.NoError(t, err)

		actual, ok := resp.(gen.PutUsersDetail200JSONResponse)
		require.True(t, ok)
		require.Equal(t, expectedDTO.FirstName, actual.FirstName)
	})

	t.Run("異常系_Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		lt := observability.NewMockControllerLayerTracer(t)

		mockApp := mock_user.NewMockUsecase(ctrl)
		mockApp.EXPECT().
			UpdateUser(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.UpdateParamsDTO{})).
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
		require.True(t, ok)
		require.Equal(t, expectedDTO.FirstName, actual.FirstName)
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
		require.True(t, ok)
		require.Equal(t, "OnlyName", actual.FirstName)
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
		require.True(t, ok)
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
