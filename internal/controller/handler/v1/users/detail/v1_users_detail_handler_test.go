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
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/users/:userId"

// subject は、認可付きエンドポイントのテストで使う認証主体の subject です。
const subject = "11111111-1111-1111-1111-111111111111"

func newServer(t *testing.T) (*server, *mock_user.MockUsecase) {
	t.Helper()
	mockApp := mock_user.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockApp}, mockApp
}

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
	tf := observability.NewNoopTracerFactory(t)
	mockApp := mock_user.NewMockUsecase(gomock.NewController(t))

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

	dto := user.UserView{
		FirstName: "User1", LastName: "One", Email: "user1@example.com", Phone: "1234567890",
		PostalCode: "150-0041", PrefectureName: "Tokyo", City: "Shibuya", Street: "1-2-3", Building: new("B1"),
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザーが存在する場合_詳細が取得できる", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)
			mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(dto, nil)

			resp, err := s.GetUsersDetail(
				testauth.MakeAvailableAuthn(context.Background(), t, subject),
				gen.GetUsersDetailRequestObject{UserId: testuuid.RequestUUID(t)},
			)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetUsersDetail200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, wantUserResponse(dto), gen.UserResponse(actual))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証の場合_ErrUnauthenticatedUser", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			resp, err := s.GetUsersDetail(context.Background(), gen.GetUsersDetailRequestObject{UserId: testuuid.RequestUUID(t)})
			require.Nil(t, resp)
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)
			mockApp.EXPECT().GetUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(user.UserView{}, apperror.ErrNotFound)

			resp, err := s.GetUsersDetail(
				testauth.MakeAvailableAuthn(context.Background(), t, subject),
				gen.GetUsersDetailRequestObject{UserId: testuuid.RequestUUID(t)},
			)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}

func Test_server_PutUsersDetail(t *testing.T) {
	t.Parallel()

	body := &gen.PutUsersDetailJSONRequestBody{
		FirstName: "First", LastName: "Last", Email: types.Email("put@example.com"),
		Phone: "09000000000", PostalCode: "123-4567", Prefecture: "Tokyo",
		City: "Shibuya", Street: "1-1-1", Building: new("Building"),
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全更新が成功する場合_リクエストがDTOへ詰め替えられ更新後のユーザーが返る", func(t *testing.T) {
			t.Parallel()
			returned := user.UserView{
				FirstName: "First", LastName: "Last", Email: "put@example.com", Phone: "09000000000",
				PostalCode: "123-4567", PrefectureName: "Tokyo", City: "Shibuya", Street: "1-1-1", Building: new("Building"),
			}

			var got *user.UpdateProfileParams
			s, mockApp := newServer(t)
			mockApp.EXPECT().UpdateUser(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *authbd.Authn, _ uuid.UUID, p *user.UpdateProfileParams) (user.UserView, error) {
					got = p
					return returned, nil
				})

			resp, err := s.PutUsersDetail(
				testauth.MakeAvailableAuthn(context.Background(), t, subject),
				gen.PutUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body},
			)
			require.NoError(t, err)

			wantDTO := &user.UpdateProfileParams{
				FirstName: body.FirstName, LastName: body.LastName, Email: string(body.Email), Phone: body.Phone,
				PostalCode: body.PostalCode, PrefectureName: body.Prefecture, City: body.City, Street: body.Street, Building: body.Building,
			}
			assert.Equal(t, wantDTO, got)

			actual, ok := resp.(gen.PutUsersDetail200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, wantUserResponse(returned), gen.UserResponse(actual))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証の場合_ErrUnauthenticatedUser", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			resp, err := s.PutUsersDetail(context.Background(), gen.PutUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body})
			require.Nil(t, resp)
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)
			mockApp.EXPECT().
				UpdateUser(gomock.Any(), gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.UpdateProfileParams{})).
				Return(user.UserView{}, apperror.ErrInternal)

			resp, err := s.PutUsersDetail(
				testauth.MakeAvailableAuthn(context.Background(), t, subject),
				gen.PutUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body},
			)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_server_PatchUsersDetail(t *testing.T) {
	t.Parallel()

	body := &gen.PatchUsersDetailJSONRequestBody{
		FirstName: new("Patched"),
		Email:     (*types.Email)(new("patch@example.com")),
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("部分更新が成功する場合_リクエストがDTOへ詰め替えられ更新後のユーザーが返る", func(t *testing.T) {
			t.Parallel()
			returned := user.UserView{FirstName: "Patched", Email: "patch@example.com"}

			var got *user.PatchParamsDTO
			s, mockApp := newServer(t)
			mockApp.EXPECT().UpdateUserPartially(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *authbd.Authn, _ uuid.UUID, p *user.PatchParamsDTO) (user.UserView, error) {
					got = p
					return returned, nil
				})

			resp, err := s.PatchUsersDetail(
				testauth.MakeAvailableAuthn(context.Background(), t, subject),
				gen.PatchUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body},
			)
			require.NoError(t, err)

			wantDTO := &user.PatchParamsDTO{
				FirstName: body.FirstName,
				Email:     new("patch@example.com"),
			}
			assert.Equal(t, wantDTO, got)

			actual, ok := resp.(gen.PatchUsersDetail200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, wantUserResponse(returned), gen.UserResponse(actual))
		})

		t.Run("Email未指定の場合はEmailがnilでDTOへ詰め替えられる", func(t *testing.T) {
			t.Parallel()
			noEmailBody := &gen.PatchUsersDetailJSONRequestBody{FirstName: new("OnlyName")}

			var got *user.PatchParamsDTO
			s, mockApp := newServer(t)
			mockApp.EXPECT().UpdateUserPartially(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *authbd.Authn, _ uuid.UUID, p *user.PatchParamsDTO) (user.UserView, error) {
					got = p
					return user.UserView{FirstName: "OnlyName"}, nil
				})

			resp, err := s.PatchUsersDetail(
				testauth.MakeAvailableAuthn(context.Background(), t, subject),
				gen.PatchUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: noEmailBody},
			)
			require.NoError(t, err)

			assert.Equal(t, new("OnlyName"), got.FirstName)
			assert.Nil(t, got.Email)

			_, ok := resp.(gen.PatchUsersDetail200JSONResponse)
			require.True(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証の場合_ErrUnauthenticatedUser", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			resp, err := s.PatchUsersDetail(context.Background(), gen.PatchUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body})
			require.Nil(t, resp)
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)
			mockApp.EXPECT().
				UpdateUserPartially(gomock.Any(), gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&user.PatchParamsDTO{})).
				Return(user.UserView{}, apperror.ErrInternal)

			resp, err := s.PatchUsersDetail(
				testauth.MakeAvailableAuthn(context.Background(), t, subject),
				gen.PatchUsersDetailRequestObject{UserId: testuuid.RequestUUID(t), Body: body},
			)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_server_PutUsersMePassword(t *testing.T) {
	t.Parallel()

	const subject = "11111111-1111-1111-1111-111111111111"
	body := &gen.PutUsersMePasswordJSONRequestBody{ //nolint:gosec // G101: テスト用のダミーパスワードで実際の資格情報ではない
		CurrentPassword: "current_password",
		NewPassword:     "new_valid_password",
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証ユーザーのパスワード変更が成功する場合_204が返る", func(t *testing.T) {
			t.Parallel()
			ctx := testauth.MakeAvailableAuthn(context.Background(), t, subject)
			s, mockApp := newServer(t)
			mockApp.EXPECT().
				ChangePassword(gomock.Any(), gomock.Any(), "current_password", "new_valid_password").
				Return(nil)

			resp, err := s.PutUsersMePassword(ctx, gen.PutUsersMePasswordRequestObject{Body: body})
			require.NoError(t, err)

			_, ok := resp.(gen.PutUsersMePassword204Response)
			assert.True(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報がない場合_エラーが返る", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)
			resp, err := s.PutUsersMePassword(context.Background(), gen.PutUsersMePasswordRequestObject{Body: body})
			require.Nil(t, resp)
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("認証subjectが不正でID取得に失敗する場合_エラーが返る", func(t *testing.T) {
			t.Parallel()
			ctx := testauth.MakeAvailableAuthn(context.Background(), t, "invalid-subject")
			s, _ := newServer(t)
			resp, err := s.PutUsersMePassword(ctx, gen.PutUsersMePasswordRequestObject{Body: body})
			require.Nil(t, resp)
			require.ErrorIs(t, err, authbd.ErrSubjectNotUUID)
		})

		t.Run("Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
			t.Parallel()
			ctx := testauth.MakeAvailableAuthn(context.Background(), t, subject)
			s, mockApp := newServer(t)
			mockApp.EXPECT().
				ChangePassword(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(apperror.ErrValidation)

			resp, err := s.PutUsersMePassword(ctx, gen.PutUsersMePasswordRequestObject{Body: body})
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})
	})
}

func Test_server_DeleteUsersDetail(t *testing.T) {
	t.Parallel()

	const subject = "11111111-1111-1111-1111-111111111111"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("削除が成功する場合_204が返る", func(t *testing.T) {
			t.Parallel()
			ctx := testauth.MakeAvailableAuthn(context.Background(), t, subject)
			s, mockApp := newServer(t)
			mockApp.EXPECT().DeleteUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			resp, err := s.DeleteUsersDetail(ctx, gen.DeleteUsersDetailRequestObject{UserId: testuuid.RequestUUID(t)})
			require.NoError(t, err)
			_, ok := resp.(gen.DeleteUsersDetail204Response)
			assert.True(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証の場合_ErrUnauthenticatedUser", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			resp, err := s.DeleteUsersDetail(context.Background(), gen.DeleteUsersDetailRequestObject{UserId: testuuid.RequestUUID(t)})
			require.Nil(t, resp)
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("Usecaseがエラーを返す場合_エラーが返る", func(t *testing.T) {
			t.Parallel()
			ctx := testauth.MakeAvailableAuthn(context.Background(), t, subject)
			s, mockApp := newServer(t)
			mockApp.EXPECT().DeleteUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(apperror.ErrNotFound)

			resp, err := s.DeleteUsersDetail(ctx, gen.DeleteUsersDetailRequestObject{UserId: testuuid.RequestUUID(t)})
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}
