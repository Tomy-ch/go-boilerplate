package usersmeroles

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
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

// authnContext は、内部ユーザー ID を解決済みの認証コンテキストを返すテストヘルパーです。
func authnContext(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx := ctxhelper.WithAuthn(context.Background())
	authn, err := auth.New("subject", "issuer", nil, nil)
	require.NoError(t, err)

	resolved, err := authn.WithUserID(userID)
	require.NoError(t, err)
	require.True(t, ctxhelper.SetAuthn(ctx, *resolved))
	return ctx
}

// rolesViewFixture は、ロール一覧ビューを生成するテストヘルパーです。
func rolesViewFixture() roleuc.RolesView {
	return roleuc.RolesView{
		Roles: []roleuc.RoleView{
			{Code: "admin", Name: "管理者"},
			{Code: "general", Name: "一般"},
		},
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	uc := mock_roleuc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, uc)

	routes := e.Router().Routes()
	require.Len(t, routes, 1)
	assert.Equal(t, http.MethodGet, routes[0].Method)
	assert.Equal(t, "/v1/users/me/roles", routes[0].Path)
}

func Test_server_GetUsersMeRoles(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体のAuthnをユースケースへ渡しロール一覧を200で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_roleuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuidtestkit.NewTestFromSalt(t, "hr_user")
			uc.EXPECT().GetMyRoles(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, authn *auth.Authn) (roleuc.RolesView, error) {
					uid, uerr := authn.UserID()
					require.NoError(t, uerr)
					assert.Equal(t, userID, uid)
					return rolesViewFixture(), nil
				})

			resp, err := s.GetUsersMeRoles(authnContext(t, userID), gen.GetUsersMeRolesRequestObject{})
			require.NoError(t, err)

			r, ok := resp.(gen.GetUsersMeRoles200JSONResponse)
			require.True(t, ok)
			require.Len(t, r.Roles, 2)
			assert.Equal(t, gen.Admin, r.Roles[0].Code)
			assert.Equal(t, "管理者", r.Roles[0].Name)
			assert.Equal(t, gen.General, r.Roles[1].Code)
			assert.Equal(t, "一般", r.Roles[1].Name)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報が無い場合、ErrUnauthenticatedUserを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_roleuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.GetUsersMeRoles(context.Background(), gen.GetUsersMeRolesRequestObject{})
			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
		})

		t.Run("ユースケースがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_roleuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().GetMyRoles(gomock.Any(), gomock.Any()).
				Return(roleuc.RolesView{}, apperror.ErrInternal)

			_, err := s.GetUsersMeRoles(authnContext(t, uuidtestkit.NewTestFromSalt(t, "hr_user_err")),
				gen.GetUsersMeRolesRequestObject{})
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_toUserRolesResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ロールの順序を保持しコードと名称をレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()

			view := rolesViewFixture()
			r := toUserRolesResponse(view)

			require.Len(t, r.Roles, len(view.Roles))
			for i, v := range view.Roles {
				assert.Equal(t, gen.RoleRefCode(v.Code), r.Roles[i].Code)
				assert.Equal(t, v.Name, r.Roles[i].Name)
			}
		})

		t.Run("ロールが空の場合はnilではない空配列のレスポンスを返す", func(t *testing.T) {
			t.Parallel()

			r := toUserRolesResponse(roleuc.RolesView{})

			assert.NotNil(t, r.Roles)
			assert.Empty(t, r.Roles)
		})
	})
}
