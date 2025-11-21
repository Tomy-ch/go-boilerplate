//go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package v1users は、/v1/users エンドポイントに関連するハンドラを提供します。
package v1users

import (
	"context"

	"boilerplate-go/internal/controller/handler/v1/users/gen"
	"boilerplate-go/internal/usecase/paging"
	useruc "boilerplate-go/internal/usecase/user"
	"boilerplate-go/pkg/ptr"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
)

type server struct {
	uc useruc.Usecase
}

func BindHandler(e *echo.Echo, uc useruc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		uc: uc,
	}, nil))
}

// GetUsers は、ユーザー一覧を取得します。
func (s *server) GetUsers(ctx context.Context, request gen.GetUsersRequestObject) (gen.GetUsersResponseObject, error) {
	// WARN: 本来はここで認可を行うべきですが、今回は省略します。
	page, err := paging.NewPagingFrom1Based(request.Params.Page, request.Params.PerPage)
	if err != nil {
		return nil, err
	}

	dtos, err := s.uc.GetAllUsers(ctx, page)
	if err != nil {
		return nil, err
	}

	users := make([]gen.UserResponse, len(dtos))
	for i, dto := range dtos {
		users[i] = gen.UserResponse{
			Name:  dto.Name,
			Email: types.Email(dto.Email),
			Phone: ptr.To(dto.Phone),
		}
	}

	res := gen.ResponseV1Users{
		Users:  users,
		Limit:  page.Limit(),
		Offset: page.Offset(),
	}

	return gen.GetUsers200JSONResponse(res), nil
}

// PostUsers implements gen.ServerInterface.
func (s *server) PostUsers(_ context.Context, _ gen.PostUsersRequestObject) (gen.PostUsersResponseObject, error) {
	panic("unimplemented")
}
