package role

import (
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/user"
	mock_user "go-boilerplate/internal/domain/user/mock"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newAuthn(t *testing.T, userID uuid.UUID) *authbd.Authn {
	t.Helper()
	a, err := authbd.New("sub-"+userID.String(), authbd.IssuerMock, nil, nil)
	require.NoError(t, err)

	resolved, err := a.WithUserID(userID)
	require.NoError(t, err)
	return resolved
}

func newRole(t *testing.T, salt, name string, code user.RoleCode) *user.Role {
	t.Helper()
	r, err := user.NewRole(uuidtestkit.NewTestFromSalt(t, salt), name, int(code))
	require.NoError(t, err)
	return r
}

func Test_usecase_GetMyRoles(t *testing.T) {
	t.Parallel()

	newUsecase := func(t *testing.T, roleRepo user.RoleRepository) *usecase {
		t.Helper()
		return &usecase{
			tracer:   observability.NewNoopTracerFactory(t).Usecase(),
			roleRepo: roleRepo,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体のuserIDをRepositoryへ渡し全ロールを安定コード付きで返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "rl_user")

			roleRepo := mock_user.NewMockRoleRepository(ctrl)
			roleRepo.EXPECT().FindRolesByUserID(gomock.Any(), userID).Return(user.Roles{
				newRole(t, "rl_admin", "管理者", user.RoleCodeAdmin),
				newRole(t, "rl_general", "一般", user.RoleCodeGeneral),
			}, nil)

			actual, err := newUsecase(t, roleRepo).GetMyRoles(t.Context(), newAuthn(t, userID))
			require.NoError(t, err)

			assert.Equal(t, []RoleView{
				{Code: "admin", Name: "管理者"},
				{Code: "general", Name: "一般"},
			}, actual.Roles)
		})

		t.Run("ロールが1件もない場合はnilではない空のロール一覧を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "rl_empty_user")

			roleRepo := mock_user.NewMockRoleRepository(ctrl)
			roleRepo.EXPECT().FindRolesByUserID(gomock.Any(), userID).Return(user.Roles{}, nil)

			actual, err := newUsecase(t, roleRepo).GetMyRoles(t.Context(), newAuthn(t, userID))
			require.NoError(t, err)
			assert.NotNil(t, actual.Roles)
			assert.Empty(t, actual.Roles)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証コンテキストがnilの場合、ErrUnauthenticatedを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			roleRepo := mock_user.NewMockRoleRepository(ctrl)

			actual, err := newUsecase(t, roleRepo).GetMyRoles(t.Context(), nil)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
			assert.Equal(t, RolesView{}, actual)
		})

		t.Run("内部UserIDが未解決の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			roleRepo := mock_user.NewMockRoleRepository(ctrl)

			// WithUserID を経ていない Authn は内部 UserID を解決できない。
			authn, err := authbd.New("sub-unresolved", authbd.IssuerMock, nil, nil)
			require.NoError(t, err)

			actual, err := newUsecase(t, roleRepo).GetMyRoles(t.Context(), authn)
			require.ErrorIs(t, err, authbd.ErrUserIDUnresolved)
			assert.Equal(t, RolesView{}, actual)
		})

		t.Run("Repositoryがエラーを返した場合、そのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "rl_err_user")
			expected := xerrors.Wrap(apperror.ErrInternal, "repository failed")

			roleRepo := mock_user.NewMockRoleRepository(ctrl)
			roleRepo.EXPECT().FindRolesByUserID(gomock.Any(), userID).Return(nil, expected)

			actual, err := newUsecase(t, roleRepo).GetMyRoles(t.Context(), newAuthn(t, userID))
			require.ErrorIs(t, err, expected)
			assert.Equal(t, RolesView{}, actual)
		})
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を注入したユースケース実装を生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			roleRepo := mock_user.NewMockRoleRepository(ctrl)
			tf := observability.NewNoopTracerFactory(t)

			actual, ok := New(roleRepo, tf).(*usecase)
			require.True(t, ok)
			assert.Equal(t, roleRepo, actual.roleRepo)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func Test_toRolesView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ロールの順序を保持しコードと名称をそれぞれ写像する", func(t *testing.T) {
			t.Parallel()

			actual := toRolesView(user.Roles{
				newRole(t, "tv_general", "一般", user.RoleCodeGeneral),
				newRole(t, "tv_admin", "管理者", user.RoleCodeAdmin),
			})

			assert.Equal(t, []RoleView{
				{Code: "general", Name: "一般"},
				{Code: "admin", Name: "管理者"},
			}, actual.Roles)
		})

		t.Run("ロールが空の場合はnilではない空のロール一覧を返す", func(t *testing.T) {
			t.Parallel()

			actual := toRolesView(nil)

			assert.NotNil(t, actual.Roles)
			assert.Empty(t, actual.Roles)
		})
	})
}

func Test_toRoleCode(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既知のロールコードを外部向けの安定コードへ写像する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "admin", toRoleCode(user.RoleCodeAdmin))
			assert.Equal(t, "general", toRoleCode(user.RoleCodeGeneral))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のロールコードはpanicする", func(t *testing.T) {
			t.Parallel()

			assert.PanicsWithValue(t, "role: unknown role code: 0", func() {
				toRoleCode(user.RoleCode(0))
			})
		})
	})
}
