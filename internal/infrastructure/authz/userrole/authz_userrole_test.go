package userrole

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/user"
	mock_user "go-boilerplate/internal/domain/user/mock"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	authzbd "go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newAuthn(t *testing.T, subjectID uuid.UUID) *authbd.Authn {
	t.Helper()

	authn, err := authbd.New(subjectID.String(), authbd.IssuerMock, nil, nil)
	require.NoError(t, err)

	return authn.WithUserID(subjectID)
}

func newRole(t *testing.T, code user.RoleCode, name string) *user.Role {
	t.Helper()

	role, err := user.NewRole(uuid.NewTestFromSalt(t, name), name, int(code))
	require.NoError(t, err)

	return role
}

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	roleRepo := mock_user.NewMockRoleRepository(ctrl)

	expected := &authorizer{roleRepo: roleRepo}
	actual := New(roleRepo)
	assert.Equal(t, expected, actual)
}

func Test_authorizer_Authorize(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("管理者ロールを持つ場合、所有者でなくても許可する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			subjectID := uuid.NewTestFromSalt(t, "admin_subject")
			ownerID := uuid.NewTestFromSalt(t, "other_owner")

			roleRepo := mock_user.NewMockRoleRepository(ctrl)
			roleRepo.EXPECT().FindRolesByUserID(gomock.Any(), subjectID).
				Return(user.Roles{newRole(t, user.RoleCodeAdmin, "管理者")}, nil)

			auth := New(roleRepo)
			err := auth.Authorize(
				context.Background(),
				newAuthn(t, subjectID),
				authzbd.ActionUserDelete,
				authzbd.NewResource("user", &ownerID),
			)
			require.NoError(t, err)
		})

		t.Run("非管理者でもリソース所有者本人なら許可する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			subjectID := uuid.NewTestFromSalt(t, "owner_subject")

			roleRepo := mock_user.NewMockRoleRepository(ctrl)
			roleRepo.EXPECT().FindRolesByUserID(gomock.Any(), subjectID).
				Return(user.Roles{newRole(t, user.RoleCodeGeneral, "一般")}, nil)

			auth := New(roleRepo)
			err := auth.Authorize(
				context.Background(),
				newAuthn(t, subjectID),
				authzbd.ActionUserGet,
				authzbd.NewResource("user", &subjectID),
			)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非管理者かつ所有者でない場合、ErrForbidden を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			subjectID := uuid.NewTestFromSalt(t, "general_subject")
			ownerID := uuid.NewTestFromSalt(t, "other_owner")

			roleRepo := mock_user.NewMockRoleRepository(ctrl)
			roleRepo.EXPECT().FindRolesByUserID(gomock.Any(), subjectID).
				Return(user.Roles{newRole(t, user.RoleCodeGeneral, "一般")}, nil)

			auth := New(roleRepo)
			err := auth.Authorize(
				context.Background(),
				newAuthn(t, subjectID),
				authzbd.ActionUserUpdate,
				authzbd.NewResource("user", &ownerID),
			)
			require.ErrorIs(t, err, authzbd.ErrForbidden)
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("内部 UserID が未解決の場合、ロールを参照せず ErrForbidden を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			roleRepo := mock_user.NewMockRoleRepository(ctrl)

			authn, err := authbd.New("not-a-uuid", authbd.IssuerMock, nil, nil)
			require.NoError(t, err)

			auth := New(roleRepo)
			actualErr := auth.Authorize(
				context.Background(),
				authn,
				authzbd.ActionUserGet,
				authzbd.NewResource("user", nil),
			)
			require.ErrorIs(t, actualErr, authzbd.ErrForbidden)
		})

		t.Run("認証主体が nil の場合、ロールを参照せず ErrForbidden を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			roleRepo := mock_user.NewMockRoleRepository(ctrl)

			auth := New(roleRepo)
			err := auth.Authorize(
				context.Background(),
				nil,
				authzbd.ActionUserGet,
				authzbd.NewResource("user", nil),
			)
			require.ErrorIs(t, err, authzbd.ErrForbidden)
		})

		t.Run("非管理者かつリソースが nil の場合、ErrForbidden を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			subjectID := uuid.NewTestFromSalt(t, "no_resource_subject")

			roleRepo := mock_user.NewMockRoleRepository(ctrl)
			roleRepo.EXPECT().FindRolesByUserID(gomock.Any(), subjectID).
				Return(user.Roles{newRole(t, user.RoleCodeGeneral, "一般")}, nil)

			auth := New(roleRepo)
			err := auth.Authorize(
				context.Background(),
				newAuthn(t, subjectID),
				authzbd.ActionUserGet,
				nil,
			)
			require.ErrorIs(t, err, authzbd.ErrForbidden)
		})

		t.Run("ロール取得がエラーの場合、そのエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			subjectID := uuid.NewTestFromSalt(t, "err_subject")
			expectedErr := xerrors.Wrap(apperror.ErrInternal, "db failed")

			roleRepo := mock_user.NewMockRoleRepository(ctrl)
			roleRepo.EXPECT().FindRolesByUserID(gomock.Any(), subjectID).
				Return(nil, expectedErr)

			auth := New(roleRepo)
			err := auth.Authorize(
				context.Background(),
				newAuthn(t, subjectID),
				authzbd.ActionUserGet,
				authzbd.NewResource("user", &subjectID),
			)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
