//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package role は、認証主体自身に割り当てられたロールの参照ユースケースを提供します。
package role

import (
	"context"
	"fmt"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/xerrors"
)

const (
	// roleCodeAdmin / roleCodeGeneral は、ロールコードに対応する外部向けの安定コードです。
	roleCodeAdmin   = "admin"
	roleCodeGeneral = "general"
)

// RoleView は、ロール 1 件分のユースケース出力 DTO です。Code は安定コード、Name は表示名です。
type RoleView struct {
	Code string
	Name string
}

// RolesView は、認証主体自身のロール一覧のユースケース出力 DTO です。
type RolesView struct {
	// Roles は、割り当てられた全ロールです。1 つも割り当てがない場合は空スライスです。
	Roles []RoleView
}

// Usecase は、認証主体自身のロールの参照ユースケースを定義します。
type Usecase interface {
	// GetMyRoles は、認証主体自身に割り当てられた全ロールを返します。
	// 取得は認証主体の userID に限定され、他ユーザーのロールは含みません。
	// 割り当てがない場合は空のロール一覧を返します。
	GetMyRoles(ctx context.Context, authn *auth.Authn) (RolesView, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer   observability.LayerTracer
	roleRepo user.RoleRepository
}

// New は、認証主体自身のロールの参照ユースケースを生成します。
func New(roleRepo user.RoleRepository, tf observability.TracerFactory) Usecase {
	return &usecase{
		tracer:   tf.Usecase(),
		roleRepo: roleRepo,
	}
}

func (u *usecase) GetMyRoles(ctx context.Context, authn *auth.Authn) (RolesView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return RolesView{}, xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")
	}
	userID, err := authn.UserID()
	if err != nil {
		return RolesView{}, xerrors.Wrap(err, "failed to get user ID from authenticator")
	}

	roles, err := u.roleRepo.FindRolesByUserID(ctx, userID)
	if err != nil {
		return RolesView{}, err
	}

	return toRolesView(roles), nil
}

// toRolesView は、ロールエンティティ列を出力 DTO へ写像します。
func toRolesView(roles user.Roles) RolesView {
	views := make([]RoleView, len(roles))
	for i, r := range roles {
		views[i] = RoleView{
			Code: toRoleCode(r.Code()),
			Name: r.Name(),
		}
	}

	return RolesView{Roles: views}
}

// toRoleCode は、ロールコードを外部向けの安定コードへ写像します。
// 対応を持たないコードは写像できないため、黙って既定値へ倒さず panic で異常を知らせます。
func toRoleCode(code user.RoleCode) string {
	switch code {
	case user.RoleCodeAdmin:
		return roleCodeAdmin
	case user.RoleCodeGeneral:
		return roleCodeGeneral
	default:
		panic(fmt.Sprintf("role: unknown role code: %d", code))
	}
}
