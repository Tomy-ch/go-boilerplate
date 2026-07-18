// Package userrole は、user_roles テーブルに基づく Authorizer 実装を提供します。
// 管理者ロールを持つ主体は全操作を許可し、それ以外はリソース所有者本人の操作のみを許可します。
package userrole

import (
	"context"

	"go-boilerplate/internal/domain/user"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	authzbd "go-boilerplate/internal/usecase/boundary/authz"
)

// authorizer は、user_roles に基づき認可判断を行う Authorizer です。
type authorizer struct {
	roleRepo user.RoleRepository
}

// New は、user_roles ベースの Authorizer を生成します。
func New(roleRepo user.RoleRepository) authzbd.Authorizer {
	return &authorizer{roleRepo: roleRepo}
}

// Authorize は、認証主体のロールとリソース所有権に基づき許可/拒否を判定します。
// 管理者ロールを持つ場合は許可（nil）、非管理者はリソース所有者本人の場合のみ許可し、
// それ以外は ErrForbidden（HTTP 403）を返します。内部 UserID が未解決の場合も拒否します。
func (a *authorizer) Authorize(
	ctx context.Context,
	authn *authbd.Authn,
	_ authzbd.Action,
	resource *authzbd.Resource,
) error {
	if authn == nil {
		return authzbd.ErrForbidden
	}

	subjectID, err := authn.UserID()
	if err != nil {
		return authzbd.ErrForbidden
	}

	roles, err := a.roleRepo.FindRolesByUserID(ctx, subjectID)
	if err != nil {
		return err
	}

	if roles.HasAdmin() {
		return nil
	}

	if resource != nil && subjectID.EqualPtr(resource.OwnerID()) {
		return nil
	}

	return authzbd.ErrForbidden
}
