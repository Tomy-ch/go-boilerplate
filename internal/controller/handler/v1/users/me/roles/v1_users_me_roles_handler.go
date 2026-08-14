//go:generate oapi-codegen --include-tags=v1/users/me/roles --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users/me/roles --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package usersmeroles は、/v1/users/me/roles エンドポイントに関連するハンドラを提供します。
package usersmeroles

import (
	"context"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/users/me/roles/gen"
	"go-boilerplate/internal/observability"
	roleuc "go-boilerplate/internal/usecase/user/role"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     roleuc.Usecase
}

// BindHandler は、認証ユーザー自身のロールのハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc roleuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetUsersMeRoles は、認証コンテキストの内部 UserID に該当するユーザー自身のロールを取得します。
func (s *server) GetUsersMeRoles(
	ctx context.Context, _ gen.GetUsersMeRolesRequestObject,
) (gen.GetUsersMeRolesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.GetMyRoles(ctx, &authn)
	if err != nil {
		return nil, err
	}

	return gen.GetUsersMeRoles200JSONResponse(toUserRolesResponse(view)), nil
}

// toUserRolesResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toUserRolesResponse(view roleuc.RolesView) gen.UserRolesResponse {
	roles := make([]gen.RoleRef, len(view.Roles))
	for i, r := range view.Roles {
		roles[i] = gen.RoleRef{
			Code: gen.RoleRefCode(r.Code),
			Name: r.Name,
		}
	}

	return gen.UserRolesResponse{Roles: roles}
}
