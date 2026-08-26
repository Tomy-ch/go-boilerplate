package integration

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	usersmeroles "go-boilerplate/internal/controller/handler/v1/users/me/roles"
	"go-boilerplate/internal/controller/handler/v1/users/me/roles/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	roleuc "go-boilerplate/internal/usecase/user/role"
	mock_roleuc "go-boilerplate/internal/usecase/user/role/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const meRolesPath = "/v1/users/me/roles"

func TestV1UsersMeRoles_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みで自分のロールがUserRolesResponseで返る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_roleuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetMyRoles(gomock.Any(), gomock.Any()).Return(roleuc.RolesView{
				Roles: []roleuc.RoleView{
					{Code: "admin", Name: "管理者"},
					{Code: "general", Name: "一般"},
				},
			}, nil)

			usersmeroles.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_rl_user"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, meRolesPath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.UserRolesResponse](t, actual)
		})

		t.Run("取得対象が認証主体のuserIDに限定されている", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			userID := uuidtestkit.NewTestFromSalt(t, "int_rl_owner")
			var capturedUserID uuid.UUID
			uc := mock_roleuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetMyRoles(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, authn *auth.Authn) (roleuc.RolesView, error) {
					id, err := authn.UserID()
					require.NoError(t, err)
					capturedUserID = id
					return roleuc.RolesView{}, nil
				},
			)

			usersmeroles.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, userID)
			actual := StartServer(t, e).DoJSON(http.MethodGet, meRolesPath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			assert.Equal(t, userID, capturedUserID)
		})

		t.Run("ロールが0件でも空配列を200で返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_roleuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetMyRoles(gomock.Any(), gomock.Any()).Return(roleuc.RolesView{}, nil)

			usersmeroles.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_rl_empty"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, meRolesPath, nil, headers)
			AssertJSONResponseType[gen.UserRolesResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証で401を返しUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_roleuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetMyRoles(gomock.Any(), gomock.Any()).Times(0)

			usersmeroles.BindHandler(e, tf, uc)

			actual := StartServer(t, e).DoJSON(http.MethodGet, meRolesPath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("ErrInternalで500を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_roleuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetMyRoles(gomock.Any(), gomock.Any()).Return(roleuc.RolesView{}, apperror.ErrInternal)

			usersmeroles.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_rl_err"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, meRolesPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
